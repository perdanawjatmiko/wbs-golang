# Phase 1 — Master Data, RBAC & Database Setup

## Audit and architecture

Blueprint dibaca ulang lengkap, tidak diubah. Working tree baseline bersih. Fondasi Phase 0 sudah menyediakan config typed, pgxpool, response envelope, request ID/logging/recovery/CORS/security/body limit, graceful lifecycle, health checks serta server-side API client. Baseline backend dan 6 API client tests lulus; lint/typecheck baseline lulus. Versi Go/Gin/Next/React/TypeScript/Tailwind dipertahankan.

Tambahan: modules/auth dan modules/master dengan handler-service-repository, platform/security/problem/migration, seed CLI dan migrations CLI. Master CRUD memakai DTO schema catalog yang statis untuk field/type validation dan identifier SQL whitelist. Semua nilai user menggunakan SQL parameter. Service menangani authorization/invariants; repository memegang query dan transaksi. Auth punya SessionStore interface konkret untuk unit tests. Tidak ada business logic Go dipindahkan ke Next.

## Important files

```text
backend/
  cmd/{api,migrate,seed}/main.go
  migrations/{000001_blueprint,000002_admin_sessions}.{up,down}.sql
  migrations/embed.go
  internal/modules/auth/{repository,service,handler,auth_test}.go
  internal/modules/master/{dto,repository,service,handler,dto_test}.go
  internal/platform/{security,problem,migration}/
  internal/seed/seed.go
  internal/config/config.go
  internal/middleware/middleware.go
  tests/phase1_test.go
frontend/
  src/features/admin/{contracts,client,server,bridge,login-form,logout-button,management}.*
  src/app/admin/login/page.tsx
  src/app/admin/(protected)/{layout,page,loading}.tsx
  src/app/admin/(protected)/[resource]/page.tsx
  src/app/admin/error.tsx
  src/app/api/backend/[...path]/route.ts
  tests/{admin,admin-http,api-client}.test.mjs
docs/{database-schema,authentication,rbac,phase-1}.md
```

Updated: README, CHANGELOG, architecture/development/API conventions, environment examples, config validation tests, middleware CORS test, frontend styling/config. Blueprint unchanged. Phase 0 public landing/connectivity and health endpoints retained.

## Dependencies

- golang-migrate/v4: embedded explicit SQL up/down, version/dirty tracking and locking, pgx/v5 driver.
- bluemonday: maintained allowlist HTML sanitizer. Formatting-only tags; no script, event attributes, images, style or URLs.
- golang.org/x/crypto/bcrypt: password hashing; existing transitive module now directly used.
- Transitive updates recorded in go.mod/go.sum. No frontend/test framework dependency added: node:test + TypeScript existing.

## Routes and endpoints

Frontend: /admin/login; protected /admin; /admin/users, /admin/roles, /admin/permissions, /admin/report-types, /admin/faqs, /admin/email-templates. Protected server layout resolves current user. Each management page has search, pagination, loading/empty/error states, create/edit form, delete confirmation, feedback, and accessible labels. No reports UI/metrics.

API details: [api-conventions](api-conventions.md). Database and commands: [database-schema](database-schema.md). Bootstrap/session flow: [authentication](authentication.md). Permission matrix: [rbac](rbac.md).

## Additional environment

Backend: DB_QUERY_TIMEOUT=5s (<HTTP_WRITE_TIMEOUT), SESSION_TTL=8h (1m..24h), ADMIN_SEED_EMAIL/PASSWORD/NAME optional bootstrap trio. Existing DATABASE_URL and CORS_ALLOWED_ORIGINS remain required for deployment connectivity. Production session cookie secure follows APP_ENV=production.

Frontend: APP_ORIGIN=http://localhost:3000, exact browser origin also listed in Go CORS allowlist. API_BASE_URL remains private server-side only. No NEXT_PUBLIC session or credentials.

Tests only: TEST_PHASE1_DATABASE_URL dedicated PostgreSQL database supporting disposable schema creation; TEST_DATABASE_URL optional Phase 0 ping; TEST_WEB_BASE_URL, TEST_ADMIN_EMAIL, TEST_ADMIN_PASSWORD for opt-in real HTTP smoke test. ALLOW_DISPOSABLE_DB_DOWN=yes and APP_ENV=test are safeguards for manual disposable down.

## Verification

See verification results appended below. Migration up/down tested only on disposable schema in newly initialized local PostgreSQL test cluster. CLI seeding twice and real HTTP Next->Go login/CRUD/logout also exercised. No production resources or main development database accessed.

## Known limitations

- Login limiter is process-local and sees adapter IP with trusted proxies disabled; it is conservative and not distributed.
- Active/revoked session rows are retained; user DELETE may return 409 due to session references. Deactivate instead; no automatic history purge.
- Lookup selectors load up to 10,000 rows; role assignment supports 200 permission IDs. Lists independently paginate at <=100/page. Evaluate searchable remote selectors if administrative catalogs grow beyond these limits.
- HTML policy deliberately allows basic formatting only. Preview sandbox has scripts/forms/network disabled by CSP; it shows editor content without sending email. Placeholder rendering with real report values is deferred; future email rendering must escape substituted values.
- Automated frontend validation uses pure form/navigation logic, server boundary tests and real HTTP smoke checks; no browser DOM/visual automation suite or production load testing was run.
- No account recovery, session management UI, distributed limiter, production deployment or TLS termination added.
- Migrations provision schema only and require an already-provisioned database. Upgrade an existing non-empty conflicting schema only after an explicit migration review; never force migration version blindly.

## Phase 2 readiness

Schema reports/report_histories and read-only active public master data endpoints are available. Before Phase 2, define API payloads and tests for public submission/tracking directly from blueprint, atomic report-code generation and quota transaction behavior. None of those workflows are implemented here.

## Test results — 2026-11-13

| Verification | Result |
| --- | --- |
| go fmt ./... / go vet ./... | PASS |
| go test ./... | PASS (Phase 1 DB integration enabled) |
| go test -race ./... | PASS (Phase 1 DB integration enabled) |
| Migration up twice / down | PASS, disposable randomly named schema only |
| Seeder idempotency / password preservation | PASS, DB assertions and CLI run twice |
| Authentication lifecycle | PASS: success, wrong/unknown/inactive, expiry/revocation/logout/rotation |
| Cookie / CSRF / rate limit / password hashing | PASS |
| RBAC assignment/re-resolution / escalation / last Super Admin | PASS, including concurrent deactivate test |
| All six resource CRUD / duplicates / FK / SQL injection / HTML | PASS |
| Frontend unit/boundary tests | PASS: 13 tests |
| Real Next.js -> Go HTTP smoke | PASS: 1 integration test; combined run 14 passed, 0 skipped |
| npm run lint / npx tsc --noEmit / npm run build | PASS |
| Blueprint diff / whitespace / ignored env secrets | PASS |

Default `npm test` runs 13 tests and explicitly skips the opt-in HTTP test without its environment. The real configured HTTP run executed all 14 with zero skips. Go Phase 1 integration similarly skips without TEST_PHASE1_DATABASE_URL; the reported configured run used a new local PostgreSQL instance, not mocks. Test instance and local application servers are stopped after QA.

## Acceptance criteria

| Criterion | Status |
| --- | --- |
| Phase 0 tetap berjalan | PASS |
| PostgreSQL migrations berhasil | PASS |
| Seluruh 9 tabel blueprint tersedia | PASS |
| Seeder berjalan idempotent | PASS |
| Initial Super Admin dapat dibuat secara aman | PASS |
| Admin dapat login dan logout | PASS |
| Session cookie aman | PASS |
| Inactive admin ditolak | PASS |
| RBAC backend berfungsi | PASS |
| Privilege escalation dicegah | PASS |
| CRUD admin users | PASS |
| CRUD roles | PASS |
| CRUD permissions | PASS |
| CRUD report types | PASS |
| CRUD FAQs | PASS |
| CRUD email templates | PASS |
| Public read-only master data API | PASS |
| Frontend login terintegrasi Go | PASS |
| Protected admin routes | PASS |
| Admin management UI tersedia dan HTTP routes berfungsi | PASS |
| Email template preview terisolasi/safe policy | PASS |
| Backend unit tests | PASS |
| Backend integration tests | PASS |
| Frontend lint | PASS |
| Frontend typecheck | PASS |
| Frontend build | PASS |
| Dokumentasi diperbarui | PASS |
| Tidak ada fitur Phase 2–5 | PASS |

UI accessibility/rendering checked through implementation and HTTP output; no browser visual automation is claimed. See known limitations above.
