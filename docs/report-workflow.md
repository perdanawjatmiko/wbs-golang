# Admin report workflow

Business rules come only from project_blueprint.md. Phase 3 uses existing reports/report_histories, contact canonicalization, active limit5 and PostgreSQL transaction lock key. No migration, report assignment, approval stage, extra status, SLA or notification has been added.

## API and permissions

| Endpoint | Permission |
| --- | --- |
| GET /api/v1/admin/reports | report.view |
| GET /api/v1/admin/reports/options | report.view |
| GET /api/v1/admin/reports/:id | report.view |
| GET /api/v1/admin/reports/:id/attachment | report.view |
| GET /api/v1/admin/dashboard | report.view |
| PATCH /api/v1/admin/reports/:id/status | report.update + CSRF |
| POST /api/v1/admin/reports/:id/notes | report.update + CSRF |

All require active, non-revoked, unexpired admin session. Super Admin has effective access to both permissions. report.update does not imply report.view; write-only calls return id/updated_at acknowledgment without report fields or status. Users with both permissions also receive status in acknowledgment. There is no role/type ownership filtering beyond blueprint RBAC.

## Status policy

Valid statuses: Open, On Progress, Done, Rejected. Any change between different valid statuses is allowed; no invented approval matrix. Valid same-status requests return409 STATUS_UNCHANGED without modifying history or updated_at, even if notes were supplied; use the notes endpoint. Target Rejected requires trimmed nonempty external_note (422 otherwise). Internal/external notes each allow up to10,000 Unicode characters, plain text; NUL/invalid UTF-8 rejected. No user-supplied admin_id/old_status/timestamp accepted.

Both mutation endpoints require expected_updated_at copied exactly from admin detail/list, preserving full fractional-second precision. Do not round via JavaScript Date.toISOString(): PostgreSQL microseconds may be lost. Missing/zero version422; mismatch against locked state409 STALE_REPORT. No schema version column is needed because existing updated_at is made monotonically increasing with GREATEST(clock_timestamp(),old+1 microsecond).

```json
{"status":"Rejected","internal_note":"Internal review","external_note":"Reason visible to reporter","expected_updated_at":"2026-11-13T10:00:00.123456Z"}
```

```json
{"success":true,"data":{"id":"<uuid>","status":"Rejected","updated_at":"2026-11-13T10:01:00.123456Z"},"meta":{"request_id":"example"}}
```

For notes, omit status and provide internal_note and/or external_note plus expected_updated_at.

## Lock ordering and transaction

1. Read report identity/contact without holding a row lock.
2. Verify stored contact is already canonical using the existing Go normalizer. Legacy/noncanonical records return409 CONTACT_REVIEW_REQUIRED rather than using an incompatible quota key; no automatic rewriting.
3. Begin READ COMMITTED transaction and acquire existing transaction advisory lock for contact_type + canonical contact (same namespace/hash as public submission).
4. SELECT report FOR UPDATE, re-read authoritative status/version/contact. If identity changed unexpectedly, return409 REPORT_IDENTITY_CHANGED; never acquire a new contact lock while holding the report row lock.
5. Re-resolve actor is_active/permissions in the transaction. No session permission cache. Validate expected version and transition.
6. Terminal->active (Done/Rejected -> Open/On Progress): count exact canonical active reports under contact lock; if >=5 return409 REPORT_ACTIVE_LIMIT_REACHED, no mutation/history. Active->terminal releases capacity automatically through query predicate; no quota counter.
7. Update status/updated_at and insert history with locked old_status, authenticated admin_id and server timestamp in one transaction.
8. Commit. Failure inserting history rolls back status and version. Contact and row locks release automatically.

All mutations, including notes, acquire contact lock before report row lock. Submission also acquires contact lock first; it never needs an existing report row lock. This avoids mixed lock-order inversion. Hash collisions only serialize unrelated contacts; count remains exact. Bounded DB_QUERY_TIMEOUT controls all database paths.

## Conflicts and retry

Stale request is409, never silent overwrite. Frontend reloads current detail after conflicts/errors and resets action forms for review; quota conflict explains the5 active limit. No automatic replay. Repeating identical request with old version cannot append a second history. Repeating the same note text with a fresh version is a new action; there is no persistent idempotency-key ledger or semantic deduplication. A network timeout during commit can be ambiguous: reload report/history before deciding whether to submit a new action.

## Queries and dashboard

List accepts page1, per_page20 (max100), search<=200 characters across code/name/contact, status, report_type_id UUID, date_from/date_to YYYY-MM-DD. Dates use APP_TIMEZONE; from inclusive midnight, to exclusive next calendar day. sort allowlist created_at/updated_at/status and direction asc/desc; default newest first, UUID secondary tie order. Invalid/malformed/duplicate query parameters rejected; raw query cap4096 bytes. Identifiers are checked again at persistence boundary, all values parameterized. Response data={items,pagination:{page,per_page,total,total_pages}}; list never includes notes.

Detail includes filtered report fields, safe attachment metadata and admin histories with actor id/current display name, no email/password/session/storage path. Histories are ascending created_at/id with history_page1 and history_per_page50 (max100), with history_pagination. Reads use read-only repeatable-read snapshots so count/items or detail/history agree within each response. No query per row.

Report type options are separately paginated, include active/inactive types, and expose only id/name/is_active under report.view, without requiring master.manage. UI options loader caps10,000 entries. Dashboard recent_limit defaults10/max50. Aggregates count all reports and each of four statuses, group by report type and return recent reports in a single repeatable-read snapshot using a fixed number of queries. Status distribution includes zeros; type distribution includes types with reports. Empty DB returns zeros/empty lists. No advanced analytics.

## Attachment access

Admin report UUID is the only client identifier. Backend checks report.view, obtains the report's key from DB, then opens via filestore.Reader under os.Root. Key format is validated; traversal/symlink/nonregular/oversized/missing files fail safely. Original file names were not stored in Phase 2; download name is report-<uuid>.<extension>. DTO has available/name/content_type/size only, no absolute path or key. Missing physical file yields available=false in detail and404 ATTACHMENT_NOT_FOUND on download.

Go streams at most stat size, checks context between reads and HTTP write timeout, sets attachment disposition/nosniff/no-store private. Interruptions log only class/request ID. Next adapter forwards session cookie only, uses an explicit UUID route allowlist, reconstructs safe disposition, validates MIME/length, bounds streaming bytes to configured upload ceiling, and forwards no raw path. There is no public download route or static server. No resume/range support was added.
