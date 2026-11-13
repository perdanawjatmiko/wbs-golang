BEGIN;
CREATE TABLE report_code_counters (
 counter_date date PRIMARY KEY,
 last_sequence bigint NOT NULL CHECK (last_sequence > 0),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX reports_active_contact_idx ON reports(contact_type,contact) WHERE status NOT IN ('Done','Rejected');
CREATE INDEX report_histories_timeline_idx ON report_histories(report_id,created_at,id);
COMMIT;
