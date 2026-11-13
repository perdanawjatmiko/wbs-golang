BEGIN;
DROP INDEX report_histories_timeline_idx;
DROP INDEX reports_active_contact_idx;
DROP TABLE report_code_counters;
COMMIT;
