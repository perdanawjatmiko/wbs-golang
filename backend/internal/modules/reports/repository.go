package reports

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	"wbs-golang/backend/internal/modules/notifications"
	"wbs-golang/backend/internal/platform/problem"
)

type Repository struct {
	pool        *pgxpool.Pool
	zone        *time.Location
	now         func() time.Time
	prefix      func() (string, error)
	maxAttempts int
}

func NewRepository(pool *pgxpool.Pool, zone *time.Location, now func() time.Time, prefix func() (string, error)) *Repository {
	return &Repository{pool, zone, now, prefix, 5}
}

func (r *Repository) NotificationAttempts(n int) *Repository { r.maxAttempts = n; return r }

// Insert owns one transaction: contact lock -> quota -> active type lock -> counter -> report -> publish -> commit.
func (r *Repository) Insert(ctx context.Context, input Submission, attachment *string, publish func() error) (Created, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Created{}, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, QuotaLockKey(input.ContactType, input.Contact)); err != nil {
		return Created{}, err
	}
	var active int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM reports WHERE contact_type=$1 AND contact=$2 AND status NOT IN ('Done','Rejected')`, input.ContactType, input.Contact).Scan(&active); err != nil {
		return Created{}, err
	}
	if active >= ActiveLimit {
		return Created{}, problem.New(409, "REPORT_ACTIVE_LIMIT_REACHED", "Anda telah mencapai batas maksimal 5 laporan aktif.")
	}
	var valid bool
	err = tx.QueryRow(ctx, `SELECT is_active FROM report_types WHERE id=$1 FOR SHARE`, input.ReportTypeID).Scan(&valid)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !valid) {
		return Created{}, validation("report_type_id", "Jenis laporan tidak tersedia.")
	}
	if err != nil {
		return Created{}, err
	}
	created := r.now().UTC()
	date := created.In(r.zone).Format("2006-01-02")
	for attempt := 0; attempt < 3; attempt++ {
		var seq int64
		if err = tx.QueryRow(ctx, `INSERT INTO report_code_counters(counter_date,last_sequence) VALUES($1::date,1) ON CONFLICT(counter_date) DO UPDATE SET last_sequence=report_code_counters.last_sequence+1,updated_at=now() RETURNING last_sequence`, date).Scan(&seq); err != nil {
			return Created{}, err
		}
		prefix, err := r.prefix()
		if err != nil {
			return Created{}, err
		}
		code, err := FormatCode(created, r.zone, prefix, seq)
		if err != nil {
			return Created{}, err
		}
		result := Created{ReportCode: code, Status: "Open", CreatedAt: created, TrackingInstructions: "Simpan kode laporan. Gunakan kode dan contact yang sama pada halaman tracking."}
		var saved, reportID string
		err = tx.QueryRow(ctx, `INSERT INTO reports(code,name,contact,contact_type,report_type_id,detail,attachment_url,status,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,'Open',$8,$8) ON CONFLICT(code) DO NOTHING RETURNING code,created_at,id::text`, code, input.Name, input.Contact, input.ContactType, input.ReportTypeID, input.Detail, attachment, created).Scan(&saved, &result.CreatedAt, &reportID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return Created{}, err
		}
		if err = notifications.Record(ctx, tx, reportID, "REPORT_CREATED", "report.created", "Open", "", r.maxAttempts); err != nil {
			return Created{}, err
		}
		if publish != nil {
			if err = publish(); err != nil {
				return Created{}, err
			}
		}
		if err = tx.Commit(ctx); err != nil {
			var postgres *pgconn.PgError
			if errors.Is(err, pgx.ErrTxCommitRollback) || errors.As(err, &postgres) {
				return Created{}, err
			}
			return Created{}, &CommitError{err}
		}
		return result, nil
	}
	return Created{}, problem.New(503, "REPORT_CODE_UNAVAILABLE", "Kode laporan belum dapat dibuat. Silakan coba lagi nanti.")
}

// CommitError distinguishes an ambiguous commit outcome; never delete a potentially referenced file blindly.
type CommitError struct{ Cause error }

func (e *CommitError) Error() string { return "report commit outcome uncertain" }
func (e *CommitError) Unwrap() error { return e.Cause }
func (r *Repository) Track(ctx context.Context, code, kind, contact string) (Tracked, error) {
	var result Tracked
	var timeline []byte
	err := r.pool.QueryRow(ctx, `SELECT r.code,r.name,rt.name,r.detail,r.status,r.created_at,r.updated_at,COALESCE((SELECT jsonb_agg(jsonb_build_object('new_status',h.new_status,'external_note',h.external_note,'created_at',h.created_at) ORDER BY h.created_at,h.id) FROM report_histories h WHERE h.report_id=r.id AND (h.old_status<>h.new_status OR btrim(h.external_note)<>'')),'[]'::jsonb) FROM reports r JOIN report_types rt ON rt.id=r.report_type_id WHERE r.code=$1 AND r.contact_type=$2 AND r.contact=$3`, code, kind, contact).Scan(&result.ReportCode, &result.Name, &result.ReportType, &result.Detail, &result.Status, &result.CreatedAt, &result.UpdatedAt, &timeline)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, NotFound()
	}
	if err != nil {
		return result, err
	}
	if err = json.Unmarshal(timeline, &result.Timeline); err != nil {
		return result, err
	}
	if len(result.Timeline) == 0 || result.Timeline[0].NewStatus != "Open" || !result.Timeline[0].CreatedAt.Equal(result.CreatedAt) {
		result.Timeline = append([]Event{{NewStatus: "Open", ExternalNote: "", CreatedAt: result.CreatedAt}}, result.Timeline...)
	}
	return result, nil
}
