BEGIN;
CREATE TABLE notification_outbox (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), event_key varchar(40) NOT NULL CHECK(event_key IN ('REPORT_CREATED','REPORT_ON_PROGRESS','REPORT_DONE','REPORT_REJECTED')),
 aggregate_type varchar(20) NOT NULL DEFAULT 'report' CHECK(aggregate_type='report'), aggregate_id uuid NOT NULL REFERENCES reports(id) ON DELETE RESTRICT,
 payload jsonb NOT NULL CHECK(jsonb_typeof(payload)='object' AND payload ? 'version' AND payload->>'version'='1'), created_at timestamptz NOT NULL DEFAULT now(), processed_at timestamptz
);
CREATE INDEX notification_outbox_aggregate_idx ON notification_outbox(aggregate_id);
CREATE TABLE notification_deliveries (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), outbox_id uuid NOT NULL REFERENCES notification_outbox(id) ON DELETE RESTRICT,
 channel varchar(10) NOT NULL CHECK(channel IN ('EMAIL','TELEGRAM')), recipient_ref varchar(254) NOT NULL,
 status varchar(20) NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','processing','sent','retrying','failed')),
 attempts integer NOT NULL DEFAULT 0 CHECK(attempts>=0), max_attempts integer NOT NULL DEFAULT 5 CHECK(max_attempts BETWEEN 1 AND 20),
 next_attempt_at timestamptz NOT NULL DEFAULT now(), locked_until timestamptz, locked_by varchar(64), claim_token uuid,
 last_error varchar(100), provider_message_id varchar(200), sent_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(outbox_id,channel,recipient_ref), CHECK(attempts<=max_attempts),
 CHECK ((status='processing' AND locked_until IS NOT NULL AND locked_by IS NOT NULL AND claim_token IS NOT NULL) OR (status<>'processing' AND locked_until IS NULL AND locked_by IS NULL AND claim_token IS NULL)),
 CHECK((status='sent')=(sent_at IS NOT NULL)), CHECK(updated_at>=created_at)
);
CREATE INDEX notification_delivery_pending_idx ON notification_deliveries(next_attempt_at,id) WHERE status IN ('pending','retrying');
CREATE INDEX notification_delivery_lease_idx ON notification_deliveries(locked_until) WHERE status='processing';
CREATE INDEX notification_delivery_monitor_idx ON notification_deliveries(status,channel,created_at DESC);
CREATE TABLE notification_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), sequence bigserial NOT NULL UNIQUE,
 event_name varchar(40) NOT NULL CHECK(event_name IN ('report.created','report.status_changed')),
 report_id uuid NOT NULL REFERENCES reports(id) ON DELETE RESTRICT, report_code varchar(100) NOT NULL,
 status varchar(20) NOT NULL CHECK(status IN ('Open','On Progress','Done','Rejected')), created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE admin_notifications (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), event_id uuid NOT NULL REFERENCES notification_events(id) ON DELETE RESTRICT,
 admin_id uuid NOT NULL REFERENCES admin_users(id) ON DELETE RESTRICT, report_id uuid NOT NULL REFERENCES reports(id) ON DELETE RESTRICT,
 title varchar(150) NOT NULL, message text NOT NULL, read_at timestamptz, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(event_id,admin_id)
);
CREATE INDEX admin_notifications_inbox_idx ON admin_notifications(admin_id,created_at DESC,id);
CREATE INDEX admin_notifications_unread_idx ON admin_notifications(admin_id) WHERE read_at IS NULL;
CREATE TABLE notification_retry_audits (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), delivery_id uuid NOT NULL REFERENCES notification_deliveries(id) ON DELETE RESTRICT,
 admin_id uuid NOT NULL REFERENCES admin_users(id) ON DELETE RESTRICT, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(delivery_id)
);
COMMIT;
