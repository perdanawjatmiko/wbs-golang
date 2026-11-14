package reports

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/modules/notifications"
)

const adminColumns = `r.id::text,r.code,r.name,r.contact,r.report_type_id::text,rt.name,r.status,r.created_at,r.updated_at`

func itemTargets(item *AdminItem) []any {
	return []any{&item.ID, &item.Code, &item.ReporterName, &item.Contact, &item.ReportTypeID, &item.ReportType, &item.Status, &item.CreatedAt, &item.UpdatedAt}
}
func rollback(ctx context.Context, tx pgx.Tx) {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	_ = tx.Rollback(cleanup)
}
func (r *Repository) readSnapshot(ctx context.Context) (pgx.Tx, error) {
	return r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
}
func adminWhere(q AdminQuery) (string, []any) {
	conditions := []string{"true"}
	args := []any{}
	param := func(v any) string { args = append(args, v); return fmt.Sprintf("$%d", len(args)) }
	if q.Search != "" {
		p := param("%" + q.Search + "%")
		conditions = append(conditions, "(r.code ILIKE "+p+" OR r.name ILIKE "+p+" OR r.contact ILIKE "+p+")")
	}
	if q.Status != "" {
		conditions = append(conditions, "r.status="+param(q.Status))
	}
	if q.TypeID != "" {
		conditions = append(conditions, "r.report_type_id="+param(q.TypeID)+"::uuid")
	}
	if q.From != nil {
		conditions = append(conditions, "r.created_at>="+param(*q.From))
	}
	if q.To != nil {
		conditions = append(conditions, "r.created_at<"+param(*q.To))
	}
	return strings.Join(conditions, " AND "), args
}
func (r *Repository) AdminList(ctx context.Context, q AdminQuery) (AdminList, error) {
	tx, err := r.readSnapshot(ctx)
	if err != nil {
		return AdminList{}, err
	}
	defer rollback(ctx, tx)
	where, args := adminWhere(q)
	result := AdminList{Items: []AdminItem{}}
	var total int
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM reports r WHERE "+where, args...).Scan(&total); err != nil {
		return result, err
	}
	result.Pagination = pagination(q.Page, q.PerPage, total)
	// Both identifiers and direction are validated again at the persistence boundary.
	column := map[string]string{"created_at": "r.created_at", "updated_at": "r.updated_at", "status": "r.status"}[q.Sort]
	if column == "" {
		return result, validation("sort", "Kolom tidak valid.")
	}
	direction := "DESC"
	if q.Direction == "asc" {
		direction = "ASC"
	} else if q.Direction != "desc" {
		return result, validation("direction", "Urutan tidak valid.")
	}
	args = append(args, q.PerPage, (q.Page-1)*q.PerPage)
	rows, err := tx.Query(ctx, "SELECT "+adminColumns+" FROM reports r JOIN report_types rt ON rt.id=r.report_type_id WHERE "+where+" ORDER BY "+column+" "+direction+",r.id "+direction+fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item AdminItem
		if err = rows.Scan(itemTargets(&item)...); err != nil {
			rows.Close()
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	return result, tx.Commit(ctx)
}
func (r *Repository) AdminDetail(ctx context.Context, id string, page, size int) (AdminDetail, *string, error) {
	tx, err := r.readSnapshot(ctx)
	if err != nil {
		return AdminDetail{}, nil, err
	}
	defer rollback(ctx, tx)
	result := AdminDetail{Histories: []AdminHistory{}}
	var key *string
	targets := append(itemTargets(&result.AdminItem), &result.ContactType, &result.Detail, &key)
	if err = tx.QueryRow(ctx, "SELECT "+adminColumns+",r.contact_type,r.detail,r.attachment_url FROM reports r JOIN report_types rt ON rt.id=r.report_type_id WHERE r.id=$1", id).Scan(targets...); err != nil {
		return result, nil, err
	}
	var total int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM report_histories WHERE report_id=$1`, id).Scan(&total); err != nil {
		return result, nil, err
	}
	result.HistoryPagination = pagination(page, size, total)
	rows, err := tx.Query(ctx, `SELECT h.id::text,h.old_status,h.new_status,h.internal_note,h.external_note,h.created_at,u.id::text,u.name FROM report_histories h JOIN admin_users u ON u.id=h.admin_id WHERE h.report_id=$1 ORDER BY h.created_at,h.id LIMIT $2 OFFSET $3`, id, size, (page-1)*size)
	if err != nil {
		return result, nil, err
	}
	for rows.Next() {
		var h AdminHistory
		if err = rows.Scan(&h.ID, &h.OldStatus, &h.NewStatus, &h.InternalNote, &h.ExternalNote, &h.CreatedAt, &h.Admin.ID, &h.Admin.Name); err != nil {
			rows.Close()
			return result, nil, err
		}
		result.Histories = append(result.Histories, h)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, nil, err
	}
	return result, key, tx.Commit(ctx)
}
func (r *Repository) AttachmentKey(ctx context.Context, id string) (*string, error) {
	var key *string
	err := r.pool.QueryRow(ctx, `SELECT attachment_url FROM reports WHERE id=$1`, id).Scan(&key)
	return key, err
}
func (r *Repository) TypeOptions(ctx context.Context, page, size int) (TypeOptions, error) {
	tx, err := r.readSnapshot(ctx)
	if err != nil {
		return TypeOptions{}, err
	}
	defer rollback(ctx, tx)
	result := TypeOptions{Items: []TypeOption{}}
	var total int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM report_types`).Scan(&total); err != nil {
		return result, err
	}
	result.Pagination = pagination(page, size, total)
	rows, err := tx.Query(ctx, `SELECT id::text,name,is_active FROM report_types ORDER BY name,id LIMIT $1 OFFSET $2`, size, (page-1)*size)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var v TypeOption
		if err = rows.Scan(&v.ID, &v.Name, &v.Active); err != nil {
			rows.Close()
			return result, err
		}
		result.Items = append(result.Items, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	return result, tx.Commit(ctx)
}
func (r *Repository) Dashboard(ctx context.Context, limit int) (Dashboard, error) {
	tx, err := r.readSnapshot(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	defer rollback(ctx, tx)
	d := Dashboard{Recent: []AdminItem{}, ByType: []TypeCount{}}
	if err = tx.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE status='Open'),count(*) FILTER(WHERE status='On Progress'),count(*) FILTER(WHERE status='Done'),count(*) FILTER(WHERE status='Rejected') FROM reports`).Scan(&d.Total, &d.Open, &d.OnProgress, &d.Done, &d.Rejected); err != nil {
		return d, err
	}
	d.ByStatus = []StatusCount{{"Open", d.Open}, {"On Progress", d.OnProgress}, {"Done", d.Done}, {"Rejected", d.Rejected}}
	rows, err := tx.Query(ctx, `SELECT rt.id::text,rt.name,count(*) FROM reports r JOIN report_types rt ON rt.id=r.report_type_id GROUP BY rt.id,rt.name ORDER BY count(*) DESC,rt.id`)
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var v TypeCount
		if err = rows.Scan(&v.ID, &v.Name, &v.Total); err != nil {
			rows.Close()
			return d, err
		}
		d.ByType = append(d.ByType, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return d, err
	}
	rows, err = tx.Query(ctx, "SELECT "+adminColumns+" FROM reports r JOIN report_types rt ON rt.id=r.report_type_id ORDER BY r.created_at DESC,r.id DESC LIMIT $1", limit)
	if err != nil {
		return d, err
	}
	for rows.Next() {
		var v AdminItem
		if err = rows.Scan(itemTargets(&v)...); err != nil {
			rows.Close()
			return d, err
		}
		d.Recent = append(d.Recent, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return d, err
	}
	return d, tx.Commit(ctx)
}

// Identity is read without a row lock, so contact lock always precedes SELECT FOR UPDATE.
func (r *Repository) Identity(ctx context.Context, id string) (string, string, error) {
	var kind, contact string
	err := r.pool.QueryRow(ctx, `SELECT contact_type,contact FROM reports WHERE id=$1`, id).Scan(&kind, &contact)
	return kind, contact, err
}

type WorkflowTx struct {
	tx          pgx.Tx
	maxAttempts int
}
type LockedReport struct {
	Status, Kind, Contact string
	UpdatedAt             time.Time
}

func (r *Repository) BeginWorkflow(ctx context.Context, kind, contact string) (*WorkflowTx, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, QuotaLockKey(kind, contact)); err != nil {
		rollback(ctx, tx)
		return nil, err
	}
	return &WorkflowTx{tx, r.maxAttempts}, nil
}
func (w *WorkflowTx) Rollback(ctx context.Context) { rollback(ctx, w.tx) }
func (w *WorkflowTx) Actor(ctx context.Context, id string) (auth.Principal, error) {
	return auth.Resolve(ctx, w.tx, id)
}
func (w *WorkflowTx) Lock(ctx context.Context, id string) (LockedReport, error) {
	var result LockedReport
	err := w.tx.QueryRow(ctx, `SELECT status,contact_type,contact,updated_at FROM reports WHERE id=$1 FOR UPDATE`, id).Scan(&result.Status, &result.Kind, &result.Contact, &result.UpdatedAt)
	return result, err
}
func (w *WorkflowTx) ActiveCount(ctx context.Context, kind, contact string) (int, error) {
	var n int
	err := w.tx.QueryRow(ctx, `SELECT count(*) FROM reports WHERE contact_type=$1 AND contact=$2 AND status NOT IN ('Done','Rejected')`, kind, contact).Scan(&n)
	return n, err
}
func (w *WorkflowTx) Apply(ctx context.Context, id, actorID, oldStatus, newStatus, internal, external string) (MutationResult, error) {
	result := MutationResult{ID: id, Status: newStatus}
	if err := w.tx.QueryRow(ctx, `UPDATE reports SET status=$2,updated_at=GREATEST(clock_timestamp(),updated_at+interval '1 microsecond') WHERE id=$1 RETURNING updated_at`, id, newStatus).Scan(&result.UpdatedAt); err != nil {
		return result, err
	}
	if _, err := w.tx.Exec(ctx, `INSERT INTO report_histories(report_id,admin_id,old_status,new_status,internal_note,external_note,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, actorID, oldStatus, newStatus, internal, external, result.UpdatedAt); err != nil {
		return result, err
	}
	if oldStatus != newStatus {
		if err := notifications.Record(ctx, w.tx, id, notifications.EventKey(newStatus), "report.status_changed", newStatus, external, w.maxAttempts); err != nil {
			return result, err
		}
	}
	return result, nil
}
func (w *WorkflowTx) Commit(ctx context.Context) error { return w.tx.Commit(ctx) }
