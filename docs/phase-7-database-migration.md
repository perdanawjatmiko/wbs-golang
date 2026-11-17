# Phase 7 database migration

Migration `000006_master_enhancements` is additive and must be applied only to an explicitly approved development database or disposable QA database.

## Changes

- Adds nullable self-FK `report_types.parent_id`; all existing type rows remain untouched and are roots.
- Adds parent lookup and normalized unique indexes: root names unique among roots, child names unique among siblings.
- Adds trigger guards for maximum depth and parent-only FAQ references.
- Adds `admin_users.must_change_password` (default false) and `temporary_credential_expires_at`; existing accounts remain usable.
- Adds `master_import_previews` with actor/module/token hash/payload/expiry/idempotent result.
- Extends the existing rate-limit scope check for import preview and commit.

## Preflight and apply

1. Take a database backup using the approved local database procedure.
2. Check duplicates that would prevent the new normalized root-name index:

```sql
SELECT lower(btrim(name)) AS normalized_name, count(*)
FROM report_types WHERE parent_id IS NULL
GROUP BY lower(btrim(name)) HAVING count(*) > 1;
```

3. Verify target database identity, environment, and migration status. Never infer safety from a private address.
4. From `backend/`, configure environment variables for the approved disposable QA database (do not reuse a normal developer/production DSN), then run:

```sh
set -a; . ./.env; set +a
go run ./cmd/migrate status
go run ./cmd/migrate up
```

The `.env` file must be an operator-created, gitignored copy with the QA `DATABASE_URL` and `APP_ENV=test`. The app does not load `.env` automatically. Do not run the `down` command for this migration while children exist; its safe guard will refuse.

5. Verify
   existing report and FAQ references still resolve and legacy roots remain present.

Migration execution was not performed here because no disposable QA database was provided. Do not run against production or an unidentified external PostgreSQL server.

## Rollback

The down migration aborts if child types exist. This avoids silently flattening hierarchy and losing associations. Reconcile/export hierarchical data and review a dedicated rollback plan before proceeding. It does not drop reports, histories, users, FAQ rows, roles, or permissions.
