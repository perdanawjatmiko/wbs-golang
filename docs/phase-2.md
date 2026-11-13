# Phase 2 — Public Reporting & Tracking

## Audit

Blueprint read completely before changes; unchanged. Phase 0 config/pgxpool/middleware/response/lifecycle and Phase 1 auth/RBAC/master CRUD were inspected and retained. Baseline Go unit tests, Phase 1 PostgreSQL integration, frontend tests/lint/typecheck passed. Actual reports/report_histories schema was inspected on a new isolated PostgreSQL QA cluster and matched migrations, including FK/unique/check constraints. No application initialization was repeated and no production resource accessed.

Gaps addressed: no public report flow/counter/storage/tracking yet, global1MiB buffered body limit unsuitable for attachment, existing Next adapter JSON/admin oriented, root still development connectivity page. Development page retained at /development; public landing now /.

## Architecture and changed files

```text
backend/
  migrations/000003_report_foundation.{up,down}.sql
  internal/modules/reports/{domain,repository,service,handler}.go
  internal/modules/reports/{domain_test,service_test}.go
  internal/platform/filestore/{local,local_test}.go
  internal/platform/ratelimit/limiter.go
  internal/testutil/fixtures.go
  tests/phase2_test.go
frontend/
  src/app/(public)/{layout,page,loading}.tsx
  src/app/(public)/{lapor,tracking}/page.tsx
  src/app/api/public/[...path]/route.ts
  src/app/development/page.tsx
  src/features/reports/{contracts,client,bridge,faq,report-form,tracking-form}.*
  tests/{public,public-http}.test.mjs
docs/{phase-2,report-submission,report-tracking,file-storage}.md
```

Updated composition root, config + validation tests, body-limit middleware, auth limiter compatibility wrapper, go.mod/go.sum, frontend server config, environment examples, .gitignore, README/changelog/architecture/schema/API/development docs. Existing migration files, admin routes and blueprint remain unchanged.

Reports is handler-service-repository; service is Gin-independent. Single Go normalizer and transaction repository are business authority. Explicit constructor injection supplies clock, random prefix, timezone, pool, storage and logger. Storage interface is minimal; proven in-memory limiter moved to platform while auth API remains compatible. New direct dependency pdfcpu performs strict PDF parsing; transitives recorded in Go modules. No frontend dependency, Redis or ORM added.

## Migrations and APIs

000003 adds report_code_counters(date PK, last_sequence BIGINT>0, updated_at), partial active quota index(contact_type,contact), history chronology index(report_id,created_at,id). No existing table recreated or data rewritten. Tracking uses unique code index; no duplicate code index.

POST /api/v1/public/reports (multipart), POST /api/v1/public/reports/track (JSON), GET /api/v1/public/reports/options (technical policy). Existing public report-types/faqs and admin APIs retained. Details and examples: report-submission.md, report-tracking.md, api-conventions.md.

Frontend: / public landing/general FAQ; /lapor form + relevant general/specific FAQ; /tracking private POST lookup; /development retained connectivity. Existing /admin routes unchanged. Browser integration uses /api/public finite allowlist, never forwards admin cookie/Authorization, streams upload under hard byte cap/backpressure and timeout, preserves request naming, and applies no-store/private to results. No business logic duplicated in Next.

## Configuration and commands

See backend/.env.example and frontend/.env.example. Additional backend: APP_TIMEZONE, UPLOAD_DIR, UPLOAD_MAX_FILE_BYTES, UPLOAD_MAX_REQUEST_BYTES, UPLOAD_ALLOWED_TYPES, UPLOAD_MAX_FILES, REPORT_UPLOAD_CONCURRENCY, REPORT_REQUEST_TIMEOUT, REPORT_SUBMIT_RATE_LIMIT, REPORT_TRACK_RATE_LIMIT, REPORT_RATE_WINDOW. Frontend: server-only UPLOAD_MAX_REQUEST_BYTES. Existing APP_ORIGIN must match browser origin/CORS allowlist. No credentials in NEXT_PUBLIC_*.

```sh
cd backend
# Export trusted .env configuration first; database must already exist.
go run ./cmd/migrate up
go run ./cmd/api
```

Admin creates an active report type and optional FAQs using Phase 1 UI before public reporting is usable; no business master data is silently seeded. No automatic database creation or migration at HTTP startup.

## Testing

```sh
cd backend
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
# Dedicated isolated test PostgreSQL only.
TEST_PHASE1_DATABASE_URL='postgres://TEST_USER:TEST_PASSWORD@127.0.0.1:5432/wbs_test?sslmode=disable' \
TEST_PHASE2_DATABASE_URL='postgres://TEST_USER:TEST_PASSWORD@127.0.0.1:5432/wbs_test?sslmode=disable' go test -v ./tests
```

Each database suite creates its own random schema, applies migrations twice, exercises fixtures, then tests down only within that disposable schema and removes it. No primary development/production database is modified. Without explicit URL variables, integration tests SKIP, never pretend success.

```sh
cd frontend
npm test
npm run lint
npx tsc --noEmit
npm run build
```

HTTP smoke opt-in: run Go/Next on a QA database with one active report type and test Super Admin; set TEST_WEB_BASE_URL, TEST_PUBLIC_WEB_BASE_URL, TEST_ADMIN_EMAIL, TEST_ADMIN_PASSWORD, then npm test. The public smoke test inserts a disposable report with attachment and checks tracking/body caps; admin smoke runs existing CRUD/login/logout. Use a dedicated QA database, since there is deliberately no public report delete endpoint.

## Limits and readiness

No browser visual automation or production load test is claimed. Forms are responsive via existing Tailwind grid/breakpoints, accessible labels/status/errors, with native single file selector. Automated coverage includes logic, actual timeline text rendering and real HTTP adapters, not a full browser interaction suite.

Local private storage has documented crash/ambiguous-commit reconciliation limits; no attachment download in this phase. No automatic idempotency receipt recovery; uncertain submissions are not retried automatically. Limiters are process-local and conservatively share adapter IP; no cross-replica coordination. Catalog loading caps at10,000 rows. Phone format validation is not carrier allocation verification. Preexisting manually imported noncanonical report contacts require a reviewed normalization/backfill before enabling submissions; migrations deliberately do not rewrite such data.

Ready for Phase 3 to add authorized admin report queries and transactional status/history workflow from blueprint, including required external note for rejection. That phase must preserve quota/contact locking if it adds active-state transitions, public timeline field exclusion, private file authorization and current RBAC. No status mutation UI/API, rejection processing, metrics, notifications, CAPTCHA or deployment implemented here.

## Verification results — 2026-11-13

| Check | Result / evidence |
| --- | --- |
| Blueprint / existing migration preservation | PASS; diffs empty |
| Baseline Phase 0–1 | PASS; auth/RBAC/master/health preserved |
| Migration 000003 up/repeat/down | PASS on disposable schema; additive public QA migration also applied |
| go fmt ./... / go vet ./... | PASS |
| go test ./... | PASS with actual PostgreSQL integration enabled |
| go test -race ./... | PASS with actual PostgreSQL integration enabled |
| Quota boundary: 4 active +10 concurrent | PASS: exactly1 success,9 REPORT_ACTIVE_LIMIT_REACHED, total5 |
| 20 concurrent independent contacts | PASS:20 unique report codes, no failure |
| Email/phone canonical quota/tracking | PASS, plus aliases/dots preserved and national-only phone rejected |
| Counter rollover/overflow/rollback/collision | PASS: new date001, sequence1000, rollback restores counter, retry collision advances,3-attempt exhaustion bounded |
| Upload security | PASS: actual valid JPEG/PNG/PDF, invalid MIME/extension/signature/content/empty/size/traversal/duplicates, private mode and storage failure |
| Cleanup / ambiguous commit | PASS: real file publish then injected failure rolls back DB/counter and deletes file; closed pgxpool failure cleans staging; uncertain commit intentionally retains file |
| Tracking security | PASS: exact pair, generic mismatch/unknown, injection attempts rejected, no internal/admin/storage/contact metadata, no-store |
| Timeline and FAQ | PASS: synthetic Open/no duplicate creation, external-only fields; general/specific/active/sorting |
| Frontend configured test run | PASS:26 tests,0 failures,0 skips (includes2 HTTP smoke tests) |
| Frontend default npm test |24 non-HTTP tests;2 HTTP suites explicitly skip unless configured |
| Frontend lint/typecheck/build | PASS, Next.js16.4.6 production build |
| Actual HTTP upload/track | PASS via Next->Go on QA, valid PNG attachment, code/summary response, wrong contact404, declared-size and chunked oversized body413 |
| Regression HTTP admin | PASS login/cookie/protected routes/CSRF/CRUD/logout |

Strictly no browser visual tests are claimed. No direct production database, upload directory, deployment or destructive schema operation on existing development data was used. Disposable test schemas alone run migration down. QA servers/database are stopped and temporary plaintext credential file removed after verification.

## Acceptance criteria

| Criterion | Status |
| --- | --- |
| Phase 0–1 regression tests | PASS |
| Database migrations | PASS |
| Report code counter available | PASS |
| Blueprint report code format | PASS |
| Report code concurrency safety | PASS |
| Maximum5 active reports | PASS |
| Quota concurrency safety | PASS |
| Consistent email/phone normalization | PASS |
| Public report submission | PASS |
| Active valid report type validation | PASS |
| Secure attachment validation | PASS |
| Private attachment storage | PASS |
| Cleanup on definite transaction failure | PASS |
| Public FAQ | PASS |
| FAQ filtering by report type | PASS |
| Responsive reporting UI implementation | PASS |
| Success displays ticket code | PASS |
| Secure code+contact tracking | PASS |
| Invalid tracking privacy | PASS |
| External-only public timeline | PASS |
| Internal notes not exposed | PASS |
| Responsive tracking UI implementation | PASS |
| Consistent API error handling | PASS |
| Public endpoint rate limiting | PASS |
| Backend unit tests | PASS |
| Backend integration tests | PASS |
| Backend race tests | PASS |
| Frontend tests | PASS |
| Frontend lint | PASS |
| Frontend typecheck | PASS |
| Frontend production build | PASS |
| Phase 2 documentation | PASS |
| No Phase 3–5 features | PASS |

For crash or ambiguous commit, cleanup semantics are necessarily different from a confirmed rollback; file-storage.md documents retained files and operator reconciliation. Responsive/accessibility checks are implementation and rendering tests, not browser visual certification.
