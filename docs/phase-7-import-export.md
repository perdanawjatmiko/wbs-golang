# Phase 7 XLSX import/export

## Shared safeguards

Supported: `.xlsx`, a single sheet named `Import`, exact stable headers, maximum 5 MiB compressed upload, 1,000 data rows, 20,000 characters per cell, and a 15-minute preview. Formula cells, unexpected columns/sheets, malformed workbooks, duplicate business keys, and invalid references are rejected. Formula values are never evaluated. Preview does not mutate master data; commit revalidates under a transaction and uses the existing shared master-data advisory lock. A repeated commit returns the stored result. A changed row snapshot invalidates stale update previews.

Expired preview records are pruned opportunistically on a new preview: uncommitted rows after 1 day; committed result rows after 30 days. This preserves a retry window while bounding inactive preview storage.

Preview and commit are rate-limited through the existing PostgreSQL-backed bounded bucket mechanism. Templates and exports use `no-store`, `nosniff`, and no secrets. Exports are bounded to 10,000 rows.

## Templates and upsert identity

- Report types: `id,parent_id,parent_name,name,description,is_active`. Roots precede children. New natural identity uses normalized root/child name. Existing records cannot be silently reparented.
- Admin users: `id,name,email,role_name,is_active`. Identity is UUID when supplied, otherwise normalized email. Existing passwords are never reset. New records get a random hash but remain inactive and require credential setup before activation.
- FAQs: `id,parent_id,parent_name,question,answer,sort_order,is_active`. General rows have empty parent. Identity without ID is normalized question within the general/parent scope. Child categories are invalid.

Exported identifiers permit safe round trips. Conflicting UUID/email identities fail the whole transaction. No partial commit is intended.

## Credential provisioning limitation

No password appears in preview, result, logs, or export. New import-created users remain inactive. Because the current application does not provide secure one-time credential retrieval or step-up verification, this phase does not activate them automatically and does not send passwords by email. An authorized administrator must set a password through the existing secured user-management process, then activate the account. Do not manually extract hashes or add passwords to spreadsheets.
