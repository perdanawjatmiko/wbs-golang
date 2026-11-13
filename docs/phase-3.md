# Phase 3 — Admin Workflow & Ticket Management

## Audit and plan

Blueprint read fully before source changes and preserved. Audited modules auth/master/reports, migrations000001–000003, actual report schema foundations, pgxpool/response/error handling, contact normalization/counter/quota transaction, tracking DTO/query, sessions/RBAC, filestore and admin/public Next adapters. Existing working changes from completed Phase 2 were retained; no reinitialization. Baseline unit/frontend checks passed; full Phase0–2 integration regression is included in final QA.

Existing schema/indexes already support Phase3. **No migration or new dependency added.** Existing migrations remain valid and tested on disposable schemas. Phase2 submission business logic is unchanged. Tracking changes only filter newly introduced internal-only note histories and preserve correct initial Open event when first external note is later.

Implementation order: admin queries/DTO -> safe private download -> locked workflow/history/notes -> quota reactivation -> dashboard -> protected UI/adapter -> concurrency/security/regression -> docs.

## Files

New backend: reports/admin_dto.go, admin_repository.go, admin_service.go, admin_handler.go, admin_test.go; platform/filestore/read.go; tests/phase3_test.go. Modified cmd/api/main.go composition root and reports/repository.go public history projection.

New frontend: features/admin-reports/{contracts,shared,dashboard,list,detail}.*, features/admin/stream.ts, app/admin/(protected)/reports/page.tsx and reports/[id]/page.tsx; tests/admin-reports.test.mjs and workflow-http.test.mjs. Modified existing admin layout/home, finite bridge allowlist, API error code mapping, admin adapter, permission helper, relevant old allowlist test, and Next development logging settings. Public adapter unchanged.

Updated README, CHANGELOG, architecture/schema/API/development/RBAC docs. Added phase-3.md/report-workflow.md/report-history.md. No blueprint changes, deployment, email/Telegram/realtime/assignment/SLA/escalation/status expansion/CAPTCHA.

## Architecture

Existing reports module owns admin DTOs, service policies and repository SQL. No duplicate report entity/business constants; IsActive/IsStatus share four statuses/ActiveLimit and QuotaLockKey. Service remains Gin-independent. Repository owns SQL, repeatable-read query snapshots and WorkflowTx persistence; service checks version/transition/rejection/quota/actor before atomic update+history.

Filestore.Reader is an additional minimal read interface, leaving Phase2 Stage/Publish/Remove contract intact. Explicit constructors wire pool/timezone/files/auth/logger/timeouts. Backend checks permissions and session for every endpoint; UI only helps UX. Full algorithm, matrix, examples and bounds: report-workflow.md and report-history.md.

## Frontend and integration

/admin now renders dashboard only with report.view; other authenticated users see authorized administrative fallback. /admin/reports and /admin/reports/[id] are protected server pages. List is server-filtered/paginated; detail renders plain text, authorized download and paginated histories; status/notes controls require report.update in addition to page's report.view. Writer-only API access does not make view routes accessible.

No chart dependency: responsive cards, accessible tables and simple CSS distribution bars. Loading skeleton, empty/error/session-expired/denied states, mutation feedback and409 refresh/review flow. UI echoes exact updated_at string, gates duplicate clicks and does not automatically replay uncertain writes. No history editing/deletion buttons.

Admin Next adapter extends finite route/method allowlist, forwards only session cookie plus CSRF/Origin, bounds request1MiB, JSON response16MiB, attachment stream to validated length/upload ceiling, total timeout15s. Public adapter behavior unchanged. App logs registered route rather than search query, contact or file path. Next incoming development request logging disabled because GET admin search may contain PII; deployment proxy/access logging must likewise avoid full query strings.

## Commands / test environment

No new application environment variables. Existing DB_QUERY_TIMEOUT, HTTP timeouts, APP_TIMEZONE and private upload configuration apply. Up-to-date migrations and Phase1 seeded report.view/report.update permissions suffice. Grant permissions via existing role editor.

```sh
cd backend
go run ./cmd/migrate up
go run ./cmd/api
# In another terminal:
cd frontend
npm run dev
```

Regression test flags TEST_DATABASE_URL, TEST_PHASE1_DATABASE_URL, TEST_PHASE2_DATABASE_URL and TEST_PHASE3_DATABASE_URL must point only to dedicated QA PostgreSQL. Phase suites create random independent schemas, apply existing migrations and run down only inside disposable schemas. No existing development/production resource is touched. Trigger/function fault injection exists only in the Phase3 test schema.

```sh
cd backend
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
# Enable real DB suite explicitly:
TEST_PHASE3_DATABASE_URL='postgres://TEST_USER:TEST_PASSWORD@127.0.0.1:5432/wbs_test?sslmode=disable' go test -v ./tests -run Phase3
```

```sh
cd frontend
npm test
npm run lint
npx tsc --noEmit
npm run build
```

HTTP opt-in: run QA Go/Next with active report type and test Super Admin; set TEST_WEB_BASE_URL, TEST_PUBLIC_WEB_BASE_URL, TEST_ADMIN_WORKFLOW_WEB_BASE_URL, TEST_ADMIN_EMAIL/PASSWORD. npm test then exercises all three real HTTP suites. Workflow smoke creates QA reports/accounts/roles and cannot run against main development or production. No automatic report/history deletion is provided.

## Known limits / Phase4 preparation

No browser visual automation/load testing claimed; tests cover form/navigation logic, actual component markup, API/service/DB behavior and HTTP streaming. Existing local file crash-reconciliation and process-local limiter limitations remain. Public timeline remains complete under prior contract; admin history is paginated. Type selector caps10,000 rows. Dashboard aggregates all permitted reports, no extra filtering model. Leading-wildcard search can scan rows; no speculative search extension/index dependency added.

updated_at is a precision-sensitive optimistic token, not a full idempotency ledger. On commit timeout reload current state/history before a new action. Identical notes with a fresh token are new actions. Current actor display name may change when account name changes; immutable audit identifier is admin_id. Privileged direct DB writes are outside API invariants. Attachment downloads do not support range/resume and use current configured size ceilings.

Ready for Phase4 notifications based on committed report/history data and existing templates. Design delivery reliability/outbox and worker behavior as a separate phase; no worker, SMTP/Telegram/SSE/WebSocket or notification side effect has been implemented here.

## Verification results — 2026-11-13

| Verification | Result |
| --- | --- |
| Blueprint and existing migrations unchanged | PASS; no Phase3 migration required |
| go fmt ./... / go vet ./... | PASS |
| go test ./... | PASS with real Phase0–3 PostgreSQL suites enabled |
| go test -race ./... | PASS with real Phase0–3 PostgreSQL suites enabled |
| Admin list/filter/pagination/sort/query rejection | PASS |
| Detail/history pagination/attachment metadata | PASS |
| Authorized/unauthenticated/no-view attachment | PASS; denied response contains no file bytes |
| Rejection validation/same-status/version checks | PASS |
| Atomic status/history and injected history failure rollback | PASS; status/version/history remain unchanged on failure |
| 5 active + terminal reactivation | PASS409; no status or history mutation |
| 4 active + terminal reactivation | PASS; active becomes5 |
| 4 active +10 mixed submissions/reactivations | PASS; exactly1 additional active,9 quota conflicts, total5 |
| Two terminal reactivations concurrent at boundary | PASS; exactly1 success,1 quota conflict |
| Two admins update same report/version | PASS;1 success,1 STALE_REPORT, exactly1 consistent history |
| Internal-only note and public tracking | PASS; admin sees content, no internal note or blank history event exposed publicly |
| Later Open external note / timestamp tie order | PASS; creation preserved, UUID secondary order stable |
| Revoked permissions/inactive user/view-update independence | PASS |
| Dashboard empty/counts/type totals/recent bounds | PASS |
| Frontend tests with QA HTTP enabled | PASS36, fail0, skip0 |
| npm run lint / npx tsc --noEmit / npm run build | PASS |
| HTTP admin/public/workflow regression | PASS; includes protected SSR fallback, streamed file, CSRF, stale guard and tracking privacy |

Default npm test runs33 non-HTTP tests and explicitly skips3 HTTP tests when QA environment is missing. Default Go tests likewise explicitly skip DB suites without test URL flags. Configured results above use actual isolated PostgreSQL and actual Next/Go servers; no mock DB success is claimed. Source-level/markup responsiveness and accessibility checks are not browser visual automation.

The six required concurrency/security scenarios are covered by named Phase3 integration subtests: five-active reactivation, mixed operations, same-report stale updates, private notes, rejection/external-reason/public state, and denied file bytes. Status history and public submission use the same contact lock throughout.

## Acceptance criteria — 42/42 PASS

| Criterion | Status |
| --- | --- |
| Phase 0–2 regression tests | PASS |
| Existing migrations remain valid | PASS |
| Admin report list API | PASS |
| Search/filter/pagination | PASS |
| Admin report detail API | PASS |
| Authorized attachment download | PASS |
| Dashboard metrics API | PASS |
| report.view enforcement | PASS |
| report.update enforcement | PASS |
| Open supported | PASS |
| On Progress supported | PASS |
| Done supported | PASS |
| Rejected supported | PASS |
| Rejected requires external_note | PASS |
| Transactional status update | PASS |
| Correct report histories | PASS |
| Internal notes stored | PASS |
| External notes stored | PASS |
| Notes-only API | PASS |
| Public tracking remains secure | PASS |
| Internal notes not exposed publicly | PASS |
| Concurrent status updates | PASS |
| Stale update conflict | PASS |
| Quota-safe reactivation | PASS |
| Concurrent submission + reactivation | PASS |
| Active quota never exceeds5 through application paths | PASS |
| Admin dashboard frontend | PASS |
| Report list frontend | PASS |
| Report detail frontend | PASS |
| Status update UI | PASS |
| Rejection UI validation | PASS |
| Notes management UI | PASS |
| Permission-aware navigation | PASS |
| Backend unit tests | PASS |
| Backend integration tests | PASS |
| Backend race tests | PASS |
| Frontend tests | PASS |
| Frontend lint | PASS |
| Frontend typecheck | PASS |
| Frontend production build | PASS |
| Phase3 documentation | PASS |
| No Phase4–5 features | PASS |

QA servers and dedicated PostgreSQL instance are stopped after verification; temporary plaintext QA credential file is removed. Historical QA data/files remain only in private temporary QA directories, never public web roots or repository artifacts. No deployment/production access performed.
