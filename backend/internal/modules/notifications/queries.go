package notifications

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) Inbox(ctx context.Context, adminID string, p, size int) (Inbox, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Inbox{}, err
	}
	defer tx.Rollback(ctx)
	result := Inbox{Items: []InboxItem{}}
	var total int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM admin_notifications WHERE admin_id=$1`, adminID).Scan(&total); err != nil {
		return result, err
	}
	result.Pagination = page(p, size, total)
	rows, err := tx.Query(ctx, `SELECT id::text,event_id::text,report_id::text,title,message,read_at,created_at FROM admin_notifications WHERE admin_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, adminID, size, (p-1)*size)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var v InboxItem
		if err = rows.Scan(&v.ID, &v.EventID, &v.ReportID, &v.Title, &v.Message, &v.ReadAt, &v.CreatedAt); err != nil {
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
func (r *Repository) Unread(ctx context.Context, id string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM admin_notifications WHERE admin_id=$1 AND read_at IS NULL`, id).Scan(&n)
	return n, err
}
func (r *Repository) Read(ctx context.Context, adminID, id string) error {
	if id == "" {
		_, err := r.pool.Exec(ctx, `UPDATE admin_notifications SET read_at=now() WHERE admin_id=$1 AND read_at IS NULL`, adminID)
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE admin_notifications SET read_at=COALESCE(read_at,now()) WHERE id=$1 AND admin_id=$2`, id, adminID)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *Repository) Deliveries(ctx context.Context, p, size int, channel, status, id string) (Deliveries, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Deliveries{}, err
	}
	defer tx.Rollback(ctx)
	result := Deliveries{Items: []Delivery{}, Metrics: map[string]int{"pending": 0, "processing": 0, "sent": 0, "retrying": 0, "failed": 0}}
	where := `($1='' OR d.channel=$1) AND ($2='' OR d.status=$2) AND ($3='' OR d.id::text=$3)`
	var total int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM notification_deliveries d WHERE `+where, channel, status, id).Scan(&total); err != nil {
		return result, err
	}
	if id != "" && total == 0 {
		return result, pgx.ErrNoRows
	}
	result.Pagination = page(p, size, total)
	rows, err := tx.Query(ctx, `SELECT status,count(*) FROM notification_deliveries GROUP BY status`)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var status string
		var n int
		if err = rows.Scan(&status, &n); err != nil {
			rows.Close()
			return result, err
		}
		result.Metrics[status] = n
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	rows, err = tx.Query(ctx, `SELECT d.id::text,o.event_key,d.channel,d.status,d.attempts,d.max_attempts,d.next_attempt_at,d.last_error,d.created_at,d.sent_at,EXISTS(SELECT 1 FROM notification_retry_audits a WHERE a.delivery_id=d.id) FROM notification_deliveries d JOIN notification_outbox o ON o.id=d.outbox_id WHERE `+where+` ORDER BY d.created_at DESC,d.id DESC LIMIT $4 OFFSET $5`, channel, status, id, size, (p-1)*size)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var v Delivery
		if err = rows.Scan(&v.ID, &v.EventKey, &v.Channel, &v.Status, &v.Attempts, &v.MaxAttempts, &v.NextAttempt, &v.LastError, &v.CreatedAt, &v.SentAt, &v.ManualRetryUsed); err != nil {
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
func (r *Repository) Cursor(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(max(sequence),0) FROM notification_events`).Scan(&n)
	return n, err
}
func (r *Repository) Replay(ctx context.Context, adminID string, after int64, limit int) ([]RealtimeEvent, error) {
	rows, err := r.pool.Query(ctx, `SELECT e.sequence,e.event_name,e.id::text,e.report_id::text,e.report_code,e.status,e.created_at FROM notification_events e JOIN admin_notifications n ON n.event_id=e.id WHERE n.admin_id=$1 AND e.sequence>$2 ORDER BY e.sequence LIMIT $3`, adminID, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []RealtimeEvent{}
	for rows.Next() {
		var e RealtimeEvent
		if err = rows.Scan(&e.Sequence, &e.Name, &e.ID, &e.ReportID, &e.Code, &e.Status, &e.Timestamp); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
func Serialize(e RealtimeEvent) string { return fmt.Sprint(e.Sequence) }
