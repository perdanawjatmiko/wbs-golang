BEGIN;

ALTER TABLE rate_limit_buckets DROP CONSTRAINT rate_limit_buckets_scope_check;
ALTER TABLE rate_limit_buckets ALTER COLUMN scope TYPE varchar(32);
ALTER TABLE rate_limit_buckets ADD CONSTRAINT rate_limit_buckets_scope_check
    CHECK (scope IN ('admin_login','report_submit','report_track','master_import_preview','master_import_commit','master_user_admin'));

-- Existing report types remain unchanged and therefore become hierarchy roots.
ALTER TABLE report_types
    ADD COLUMN parent_id uuid NULL REFERENCES report_types(id) ON DELETE RESTRICT;
ALTER TABLE report_types
    ADD CONSTRAINT report_types_not_own_parent CHECK (parent_id IS NULL OR parent_id <> id);
CREATE INDEX report_types_parent_idx ON report_types(parent_id, is_active, name);

-- Names are unique within the same level. Fail safely if legacy duplicates exist;
-- operators can resolve those records explicitly before retrying the migration.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM report_types
        WHERE parent_id IS NULL
        GROUP BY lower(btrim(name)) HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'duplicate parent report type names; resolve duplicates before migration 000006';
    END IF;
END $$;
CREATE UNIQUE INDEX report_types_parent_name_unique
    ON report_types (lower(btrim(name))) WHERE parent_id IS NULL;
CREATE UNIQUE INDEX report_types_child_name_unique
    ON report_types (parent_id, lower(btrim(name))) WHERE parent_id IS NOT NULL;

-- Existing accounts are not forced into a password-change flow by this migration.
ALTER TABLE admin_users
    ADD COLUMN must_change_password boolean NOT NULL DEFAULT false,
    ADD COLUMN temporary_credential_expires_at timestamptz NULL;

CREATE TABLE master_import_previews (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id uuid NOT NULL REFERENCES admin_users(id) ON DELETE RESTRICT,
    module varchar(32) NOT NULL CHECK (module IN ('admin-users','report-types','faqs')),
    token_hash bytea NOT NULL UNIQUE,
    payload jsonb NOT NULL,
    result jsonb NULL,
    expires_at timestamptz NOT NULL,
    committed_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX master_import_previews_expiry_idx ON master_import_previews(expires_at);

CREATE OR REPLACE FUNCTION wbs_guard_report_type_hierarchy() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    parent_parent_id uuid;
BEGIN
    IF NEW.parent_id IS NOT NULL THEN
        IF NEW.parent_id = NEW.id THEN
            RAISE EXCEPTION 'report type cannot be its own parent' USING ERRCODE = '23514';
        END IF;
        SELECT parent_id INTO parent_parent_id FROM report_types WHERE id = NEW.parent_id FOR KEY SHARE;
        IF NOT FOUND OR parent_parent_id IS NOT NULL THEN
            RAISE EXCEPTION 'report type parent must be a root category' USING ERRCODE = '23514';
        END IF;
        IF EXISTS (SELECT 1 FROM report_types WHERE parent_id = NEW.id) THEN
            RAISE EXCEPTION 'category with children cannot become a child' USING ERRCODE = '23514';
        END IF;
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER report_types_hierarchy_guard
    BEFORE INSERT OR UPDATE OF parent_id ON report_types
    FOR EACH ROW EXECUTE FUNCTION wbs_guard_report_type_hierarchy();

CREATE OR REPLACE FUNCTION wbs_guard_faq_parent_type() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.report_type_id IS NOT NULL AND EXISTS (
        SELECT 1 FROM report_types WHERE id = NEW.report_type_id AND parent_id IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'FAQ report type must be a parent category' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER faqs_parent_type_guard
    BEFORE INSERT OR UPDATE OF report_type_id ON faqs
    FOR EACH ROW EXECUTE FUNCTION wbs_guard_faq_parent_type();

COMMIT;
