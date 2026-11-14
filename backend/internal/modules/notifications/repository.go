package notifications

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	"wbs-golang/backend/internal/platform/problem"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(p *pgxpool.Pool) *Repository { return &Repository{p} }

type Job struct {
	ID, Channel, Recipient, EventKey, Token string
	Attempts, MaxAttempts                   int
	Payload                                 Payload
}

func (r *Repository) Claim(ctx context.Context, owner string, batch int, lease time.Duration, email, telegram bool) ([]Job, error) {
	if err := r.expire(ctx, batch); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `WITH candidates AS (
 SELECT id FROM notification_deliveries WHERE attempts<max_attempts AND ((channel='EMAIL' AND $4) OR (channel='TELEGRAM' AND $5)) AND
 ((status IN ('pending','retrying') AND next_attempt_at<=now()) OR (status='processing' AND locked_until<now()))
 ORDER BY next_attempt_at,id FOR UPDATE SKIP LOCKED LIMIT $2
 ), claimed AS (UPDATE notification_deliveries d SET status='processing',attempts=attempts+1,locked_by=$1,claim_token=gen_random_uuid(),locked_until=now()+$3::interval,updated_at=now() FROM candidates c WHERE d.id=c.id RETURNING d.*)
 SELECT c.id::text,c.channel,c.recipient_ref,o.event_key,c.claim_token::text,c.attempts,c.max_attempts,o.payload FROM claimed c JOIN notification_outbox o ON o.id=c.outbox_id`, owner, batch, lease.String(), email, telegram)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []Job{}
	for rows.Next() {
		var j Job
		var raw []byte
		if err = rows.Scan(&j.ID, &j.Channel, &j.Recipient, &j.EventKey, &j.Token, &j.Attempts, &j.MaxAttempts, &raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &j.Payload); err != nil {
			// A poison payload must fail only its own delivery, not abandon the rest of this claim wave.
			j.Payload = Payload{}
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}
func (r *Repository) EmailContent(ctx context.Context, j Job) (Message, error) {
	var subject, body, name string
	err := r.pool.QueryRow(ctx, `SELECT t.subject,t.body_html,r.name FROM notification_outbox o JOIN reports r ON r.id=o.aggregate_id JOIN email_templates t ON t.event_key=o.event_key WHERE o.id=(SELECT outbox_id FROM notification_deliveries WHERE id=$1) AND r.contact_type='EMAIL'`, j.ID).Scan(&subject, &body, &name)
	if err != nil {
		return Message{}, &Failure{Code: "TEMPLATE_UNAVAILABLE"}
	}
	subject, body, err = Render(j.EventKey, subject, body, name, j.Payload)
	if err != nil {
		return Message{}, &Failure{Code: "TEMPLATE_INVALID"}
	}
	return Message{Recipient: j.Recipient, Subject: subject, HTML: body, DeliveryID: j.ID}, nil
}
func (r *Repository) Finish(ctx context.Context, j Job, providerID string, f *Failure, delay time.Duration) (bool, error) {
	status, code := "sent", ""
	var sent *time.Time
	if f == nil {
		v := time.Now()
		sent = &v
	} else {
		code = f.Code
		status = "failed"
		if f.Retryable && j.Attempts < j.MaxAttempts {
			status = "retrying"
		}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var outboxID string
	if err = tx.QueryRow(ctx, `SELECT o.id::text FROM notification_outbox o JOIN notification_deliveries d ON d.outbox_id=o.id WHERE d.id=$1 FOR UPDATE OF o`, j.ID).Scan(&outboxID); err != nil {
		return false, err
	}
	tag, err := tx.Exec(ctx, `UPDATE notification_deliveries SET status=$3,last_error=NULLIF($4,''),provider_message_id=NULLIF($5,''),sent_at=$6,next_attempt_at=now()+$7::interval,locked_until=NULL,locked_by=NULL,claim_token=NULL,updated_at=now() WHERE id=$1 AND claim_token=$2::uuid AND status='processing' AND locked_until>clock_timestamp()`, j.ID, j.Token, status, code, providerID, sent, delay.String())
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	_, err = tx.Exec(ctx, `UPDATE notification_outbox o SET processed_at=now() WHERE id=(SELECT outbox_id FROM notification_deliveries WHERE id=$1) AND NOT EXISTS(SELECT 1 FROM notification_deliveries d WHERE d.outbox_id=o.id AND d.status NOT IN ('sent','failed'))`, j.ID)
	if err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}
func (r *Repository) Retry(ctx context.Context, id, adminID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var outboxID string
	if err = tx.QueryRow(ctx, `SELECT o.id::text FROM notification_outbox o JOIN notification_deliveries d ON d.outbox_id=o.id WHERE d.id=$1 FOR UPDATE OF o`, id).Scan(&outboxID); err != nil {
		return err
	}
	// One manual retry per delivery, retaining the global attempt bound and an immutable audit.
	tag, err := tx.Exec(ctx, `UPDATE notification_deliveries d SET status='retrying',next_attempt_at=now(),last_error=NULL,updated_at=now() WHERE id=$1 AND status='failed' AND attempts<max_attempts AND NOT EXISTS(SELECT 1 FROM notification_retry_audits a WHERE a.delivery_id=d.id)`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return problem.New(409, "DELIVERY_RETRY_CONFLICT", "Delivery tidak dapat diulang atau batas retry tercapai.")
	}
	if _, err = tx.Exec(ctx, `INSERT INTO notification_retry_audits(delivery_id,admin_id) VALUES($1,$2)`, id, adminID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE notification_outbox SET processed_at=NULL WHERE id=(SELECT outbox_id FROM notification_deliveries WHERE id=$1)`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// expire uses the same outbox-before-delivery ordering as finalization and manual retry.
func (r *Repository) expire(ctx context.Context, batch int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT o.id::text FROM notification_outbox o WHERE EXISTS(SELECT 1 FROM notification_deliveries d WHERE d.outbox_id=o.id AND d.status='processing' AND d.locked_until<now() AND d.attempts>=d.max_attempts) ORDER BY o.id FOR UPDATE SKIP LOCKED LIMIT $1`, batch)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err = tx.Exec(ctx, `UPDATE notification_deliveries SET status='failed',last_error='LEASE_ATTEMPTS_EXHAUSTED',locked_by=NULL,locked_until=NULL,claim_token=NULL,updated_at=now() WHERE outbox_id=$1 AND status='processing' AND locked_until<now() AND attempts>=max_attempts`, id); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE notification_outbox o SET processed_at=now() WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM notification_deliveries d WHERE d.outbox_id=o.id AND d.status NOT IN ('sent','failed'))`, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
