# Phase 7 — Master data enhancement

Phase 7 adds two-level report type categories, parent-scoped FAQs, master-data XLSX import/export, and an email-template rich editor. It is an approved enhancement to the baseline in `project_blueprint.md`; the blueprint remains unchanged. Existing report identifiers, histories, users, role assignments, FAQ references, and templates are preserved by additive migration `000006`.

## Implementation status

- Backend hierarchy validation and report-submission parent/child validation are implemented.
- Admin master CRUD continues to use existing RBAC and API envelopes. Report type responses now include `parent_id`; report projections display a child under its parent while legacy parent references remain readable.
- Import preview is non-mutating, actor-bound, expires after 15 minutes, validates strict XLSX templates, and commit revalidates/upserts in one transaction. Preview/commit use shared PostgreSQL rate-limit buckets.
- New imported users are inactive and have a one-time-password-change flag with a 24-hour expiry. No plaintext credential is returned. Secure one-time credential delivery/step-up provisioning is not available in the current authentication architecture; imported users require an administrator-operated credential setup before activation.
- Email templates have dedicated create/edit routes using Tiptap. Backend sanitization remains authoritative.

## Database

Migration `000006_master_enhancements` adds `report_types.parent_id`, hierarchy/FAQ guards, case-insensitive sibling-name indexes, temporary-password metadata, import-preview storage, and import rate-limit scopes. Run only after selecting an explicitly disposable or approved development database. Existing flat report types naturally remain root categories. The migration fails safely if normalized parent names conflict.

The down migration refuses to run while child categories exist because flattening would lose valid hierarchy. Back up and reconcile/export new data before a separately reviewed rollback.

## Local verification

```sh
cd backend && go test ./... && go vet ./...
cd ../frontend && npm run lint && npx tsc --noEmit && npm test && npm run build
```

Backend vet/unit/race checks, frontend lint/typecheck/tests, and production build pass locally. Four frontend HTTP integration tests remain environment-gated. PostgreSQL migration/integration verification requires a disposable QA database and was not run.

See [architecture](phase-7-architecture.md), [database migration](phase-7-database-migration.md), [import/export](phase-7-import-export.md), [security](phase-7-security.md), and [test report](phase-7-test-report.md).
