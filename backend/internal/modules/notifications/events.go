package notifications

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
)

// Payload v1 contains only notification snapshots: never contact, detail or internal notes.
type Payload struct {
	Version      int       `json:"version"`
	ReportCode   string    `json:"report_code"`
	ReportType   string    `json:"report_type"`
	Status       string    `json:"status"`
	ExternalNote string    `json:"external_note"`
	Timestamp    time.Time `json:"timestamp"`
}

func EventKey(status string) string {
	return map[string]string{"On Progress": "REPORT_ON_PROGRESS", "Done": "REPORT_DONE", "Rejected": "REPORT_REJECTED"}[status]
}

// Record must run in the caller's business transaction after report/history writes.
// The append lock serializes sequence allocation through commit, avoiding replay gaps
// caused by transactions committing in a different order from bigserial allocation.
func Record(ctx context.Context, tx pgx.Tx, reportID, eventKey, eventName, status, external string, maxAttempts int) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(761983224)`); err != nil {
		return err
	}
	var p Payload
	var contact, kind string
	p.Version = 1
	p.Status = status
	p.ExternalNote = external
	if err := tx.QueryRow(ctx, `SELECT r.code,t.name,r.contact,r.contact_type,r.updated_at FROM reports r JOIN report_types t ON t.id=r.report_type_id WHERE r.id=$1`, reportID).Scan(&p.ReportCode, &p.ReportType, &contact, &kind, &p.Timestamp); err != nil {
		return err
	}
	var eventID string
	if err := tx.QueryRow(ctx, `INSERT INTO notification_events(event_name,report_id,report_code,status,created_at) VALUES($1,$2,$3,$4,$5) RETURNING id::text`, eventName, reportID, p.ReportCode, status, p.Timestamp).Scan(&eventID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO admin_notifications(event_id,admin_id,report_id,title,message,created_at)
 SELECT $1,u.id,$2,$3,$4,$5 FROM admin_users u JOIN roles r ON r.id=u.role_id WHERE u.is_active AND (r.name='Super Admin' OR EXISTS(SELECT 1 FROM role_permissions rp JOIN permissions p ON p.id=rp.permission_id WHERE rp.role_id=u.role_id AND p.slug='report.view'))`, eventID, reportID, p.ReportCode, status, p.Timestamp); err != nil {
		return err
	}
	if eventKey == "" {
		return nil
	} // Open reactivation is realtime only, not a new business email event.
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO notification_outbox(id,event_key,aggregate_id,payload) VALUES($1,$2,$3,$4)`, eventID, eventKey, reportID, data); err != nil {
		return err
	}
	if kind == "EMAIL" {
		if _, err = tx.Exec(ctx, `INSERT INTO notification_deliveries(outbox_id,channel,recipient_ref,max_attempts) VALUES($1,'EMAIL',$2,$3)`, eventID, contact, maxAttempts); err != nil {
			return err
		}
	}
	if eventKey == "REPORT_CREATED" {
		if _, err = tx.Exec(ctx, `INSERT INTO notification_deliveries(outbox_id,channel,recipient_ref,max_attempts) VALUES($1,'TELEGRAM','configured-group',$2)`, eventID, maxAttempts); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE notification_outbox o SET processed_at=now() WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM notification_deliveries d WHERE d.outbox_id=o.id)`, eventID)
	return err
}

func ValidPayload(key string, p Payload) bool {
	if p.Version != 1 || p.ReportCode == "" || len(p.ReportCode) > 100 || p.ReportType == "" || len(p.ReportType) > 600 || p.Timestamp.IsZero() || len(p.ExternalNote) > 40000 {
		return false
	}
	if key == "REPORT_CREATED" {
		return p.Status == "Open"
	}
	return EventKey(p.Status) == key && key != "" && (p.Status != "Rejected" || strings.TrimSpace(p.ExternalNote) != "")
}
