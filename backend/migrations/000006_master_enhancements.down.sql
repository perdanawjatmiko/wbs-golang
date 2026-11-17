BEGIN;

ALTER TABLE rate_limit_buckets DROP CONSTRAINT rate_limit_buckets_scope_check;
DELETE FROM rate_limit_buckets WHERE scope IN ('master_import_preview','master_import_commit','master_user_admin');
ALTER TABLE rate_limit_buckets ALTER COLUMN scope TYPE varchar(20);
ALTER TABLE rate_limit_buckets ADD CONSTRAINT rate_limit_buckets_scope_check
    CHECK (scope IN ('admin_login','report_submit','report_track'));

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM report_types WHERE parent_id IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot down migration 000006 while child report types exist; export/reconcile data before rollback';
    END IF;
END $$;

DROP TRIGGER IF EXISTS faqs_parent_type_guard ON faqs;
DROP FUNCTION IF EXISTS wbs_guard_faq_parent_type();
DROP TRIGGER IF EXISTS report_types_hierarchy_guard ON report_types;
DROP FUNCTION IF EXISTS wbs_guard_report_type_hierarchy();
DROP INDEX IF EXISTS report_types_child_name_unique;
DROP INDEX IF EXISTS report_types_parent_name_unique;
DROP INDEX IF EXISTS report_types_parent_idx;
ALTER TABLE report_types DROP CONSTRAINT IF EXISTS report_types_not_own_parent;
ALTER TABLE report_types DROP COLUMN IF EXISTS parent_id;
ALTER TABLE admin_users
    DROP COLUMN IF EXISTS temporary_credential_expires_at,
    DROP COLUMN IF EXISTS must_change_password;
DROP TABLE IF EXISTS master_import_previews;

COMMIT;
