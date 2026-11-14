# Phase 4 — Multi-Channel Notifications

## Audit and implementation

Blueprint read completely and unchanged. Audited Go/Next dependencies, auth/master/reports modules, migrations000001–000003, actual report/history/template schema, seed/RBAC/session resolution, pgxpool, quota/hash/counter/submission/status/notes transaction boundaries, tracking privacy DTO, response/error/logging, frontend layout/adapter and Phase0–3 docs/tests. Baseline Go passed; baseline frontend33 passed with3 HTTP suites explicitly skipped until QA was configured. Existing architecture, schema migrations and functionality retained.

Implementation order: additive schema -> same-transaction event writer/fanout -> delivery queries/leases/worker -> verified SMTP/allowlisted templates/Telegram -> durable SSE/inbox -> monitoring -> Next notification center/stream adapter -> PostgreSQL/concurrency/failure/HTTP tests -> docs. No new Go/npm dependency was required; standard net/smtp, crypto/tls, net/http, pgx, existing sanitizer, native EventSource and existing Node test setup suffice.

## Files and architecture

- backend/migrations/000004_notifications.{up,down}.sql: five infrastructure tables, explicit constraints/FKs/indexes.
- backend/internal/modules/notifications/: event contracts and transactional writer, DTOs, repository/queries, service/handler, worker, retry/provider failures, safe templates, SSE and unit tests.
- backend/internal/platform/{mailer,telegram}/: bounded verified providers and local-only tests.
- backend/cmd/worker/main.go: dedicated signal-aware worker composition root.
- backend/internal/config/notifications*: typed validated groups/tests; existing config integrates them.
- reports/repository.go/admin_repository.go: atomic event hooks, configurable attempt snapshot. Existing contact/row/counter lock order, business quota/version rules and public tracking unchanged.
- cmd/api/main.go: explicit auth/notification dependencies and shutdown context; seed/seed.go adds notifications.manage idempotently.
- backend/tests/phase4_test.go: disposable-schema PostgreSQL integration/failure/security/streaming tests.
- frontend/features/notifications/{contracts,bell,deliveries} and protected /admin/notification-deliveries: bell/inbox/read actions, live states, deduplication and delivery filters/retry.
- Existing admin layout, finite bridge and Go adapter extended for notification routes and cancellable incremental SSE. Public adapter unchanged.
- frontend/tests/notifications{,-http}.test.mjs: rendering, effect lifecycle, dedup/reconnect/revocation, allowlist, live Go–Next SSE/inbox/CSRF/monitoring/logout.
- README/CHANGELOG/.env example and architecture/schema/API/development/RBAC docs updated; seven Phase4 documents added.

See notification-architecture.md for contracts/transaction/recipient/lock decisions, notification-worker.md for claims/retries, smtp-integration.md, telegram-integration.md, realtime-sse.md and notification-recovery.md for operation and limitations.

## API and frontend

| Method / /api/v1/admin path | Permission |
| --- | --- |
| GET /notifications | report.view |
| GET /notifications/unread-count | report.view |
| POST /notifications/:id/read | report.view + CSRF + own inbox |
| POST /notifications/read-all | report.view + CSRF + own inbox |
| GET /notifications/stream | report.view, periodically revalidated |
| GET /notification-deliveries | notifications.manage |
| GET /notification-deliveries/:id | notifications.manage |
| POST /notification-deliveries/:id/retry | notifications.manage + CSRF |

Default page1/per_page20; max100. Monitoring channel EMAIL/TELEGRAM and pending/processing/sent/retrying/failed filters; global metrics use those five statuses. Details use the same safe delivery DTO list with one item. No raw provider payload/recipient exposed. Read/retry success uses normal response envelope; errors use401/403/404/409/422 as applicable. SSE is the deliberate event-stream response exception.

Frontend notification bell sits in existing protected admin header only with report.view. /admin/notification-deliveries requires notifications.manage and provides channel/status filters, pagination, safe error/attempt details and explicit retry confirmation. No session token enters JavaScript/localStorage, no new frontend business logic, chart library or global state.

## Run and environment

```sh
cd backend
# Export backend environment from a private local file, then:
go run ./cmd/migrate up
go run ./cmd/seed
go run ./cmd/api
# Separate terminal, same exported configuration:
go run ./cmd/worker
# Separate frontend terminal, existing API_BASE_URL / APP_ORIGIN:
cd frontend
npm run dev
```

Migration never runs during API startup. Migration down is destructive and only used by automated disposable-schema tests, never the normal development database. Existing migrations000001–000003 stay byte-for-byte unchanged. Existing bootstrap command still requires all three ADMIN_SEED_* variables only when creating an initial account; no default credential or overwrite.

Additional environment defaults are fully listed in backend/.env.example: SMTP_*, TELEGRAM_*, NOTIFICATION_WORKER_*, NOTIFICATION_MAX_ATTEMPTS/RETRY_INTERVALS, SSE_MAX_CONNECTIONS/REPLAY_LIMIT/POLL_INTERVAL/HEARTBEAT_INTERVAL. No new frontend environment or secret is required. SMTP/Telegram disabled defaults keep API/inbox available and queued deliveries pending.

## Verification and limitations

QA uses a new loopback-only PostgreSQL instance and random disposable schemas. TEST_DATABASE_URL, TEST_PHASE1_DATABASE_URL, TEST_PHASE2_DATABASE_URL, TEST_PHASE3_DATABASE_URL and TEST_PHASE4_DATABASE_URL explicitly opt in. Run go fmt ./..., go vet ./..., go test ./..., go test -race ./.... Automated migration up/down is confined to generated test schemas. No production resource, development-main down migration or real SMTP/Telegram delivery is used.

Frontend: npm test, npm run lint, npx tsc --noEmit, npm run build. Configure TEST_WEB_BASE_URL, TEST_PUBLIC_WEB_BASE_URL, TEST_ADMIN_WORKFLOW_WEB_BASE_URL, TEST_NOTIFICATIONS_WEB_BASE_URL and private TEST_ADMIN_EMAIL/PASSWORD for the four real HTTP suites. Seed a standing active QA report type before running HTTP tests in parallel; CRUD suites create/delete their own types. Default tests explicitly skip HTTP suites without this configuration. See final verification below for executed results.

Reliability is at-least-once processing. Duplicate risk after ambiguous provider acceptance is documented, not hidden. Local-only TLS/HTTP providers verify behavior without contacting the internet. No browser visual automation, production load/proxy behavior or live provider account acceptance is claimed. No retention automation or full provider circuit breaker is introduced. Bounded attempts and permanent credential failure prevent aggressive retries per delivery; a large newly enabled backlog can still use configured concurrency. No Phase5 CAPTCHA, deployment or unrelated feature is included.

Phase5 can build on durable delivery/audit state, permission-aware streaming and explicit failure recovery. Before any production rollout, review retention/privacy, scale/append-lock throughput, provider account settings and proxy SSE buffering/timeouts as part of that separately authorized phase.

## Verification — 2026-11-14

| Check | Result |
| --- | --- |
| go fmt / go vet | PASS |
| go test ./... with actual Phase0–4 PostgreSQL | PASS |
| go test -race ./... with actual Phase0–4 PostgreSQL | PASS |
| Phase4 targeted PostgreSQL/race suite | PASS: atomic rollback, PHONE eligibility, status snapshots, notes/stale isolation, multiworker claims/concurrent finalization, lease recovery, real SKIP LOCKED, exhausted claims, isolated poison-payload failure, in-flight cancellation, inbox/CSRF/monitoring/replay across separately composed API instances/revocation |
| Existing Phase2/3 quota concurrency | PASS: at boundary only one additional active report; same-report stale guard remains correct |
| Dedicated worker CLI | PASS: isolated startup, providers disabled, SIGTERM returns0 |
| Local SMTP tests | PASS: verified STARTTLS and implicit TLS, no downgrade, validation and safe classification |
| Local Telegram TLS tests | PASS: response validation,429 retry_after,5xx/permanent4xx, disabled channel |
| npm test with all four real Go–Next HTTP suites | PASS42, fail0, skip0 |
| npm run lint / npx tsc --noEmit / npm run build | PASS |
| Blueprint and migrations000001–000003 | Unchanged |
| Live external providers / browser visual automation | Not run, not claimed |

Initial HTTP public fixture race was corrected by adding a standing active report type only in QA PostgreSQL; existing tests were not weakened. Default frontend run is38 pass plus4 explicit HTTP skips without QA variables. Default Go suites skip DB tests when opt-in URLs are absent; results above used real isolated PostgreSQL, not a mock DB.

## Acceptance criteria — 50/50 PASS

| # | Criterion | Result |
| --- | --- | --- |
| 1 | Phase 0–3 regression tests | PASS |
| 2 | Existing migrations unchanged | PASS |
| 3 | Additive Phase4 migration | PASS |
| 4 | Transactional outbox | PASS |
| 5 | Durable report-created event | PASS |
| 6 | Correct status events | PASS |
| 7 | Rollback prevents events | PASS |
| 8 | Notes-only preserved | PASS |
| 9 | Worker entrypoint | PASS |
| 10 | Atomic claiming | PASS |
| 11 | Multi-worker safety | PASS |
| 12 | Worker graceful shutdown | PASS |
| 13 | Abandoned lease recovery | PASS |
| 14 | Stale ownership rejection | PASS |
| 15 | Retry/backoff | PASS |
| 16 | Permanent failures | PASS |
| 17 | Delivery deduplication | PASS |
| 18 | SMTP client | PASS |
| 19 | Safe DB templates | PASS |
| 20 | EMAIL-only recipients | PASS |
| 21 | Internal notes excluded from email | PASS |
| 22 | Telegram client | PASS |
| 23 | New Open group alert | PASS |
| 24 | Telegram excludes sensitive data | PASS |
| 25 | Telegram429 retry | PASS |
| 26 | SSE endpoint | PASS |
| 27 | SSE authentication | PASS |
| 28 | SSE report.view | PASS |
| 29 | SSE reconnect/replay | PASS |
| 30 | SSE cross-instance durable source | PASS |
| 31 | Admin inbox | PASS |
| 32 | Unread count | PASS |
| 33 | Mark read | PASS |
| 34 | Cross-admin inbox blocked | PASS |
| 35 | Delivery monitoring | PASS |
| 36 | Manual retry permission | PASS |
| 37 | Frontend notification center | PASS |
| 38 | Frontend SSE | PASS |
| 39 | Frontend delivery monitoring | PASS |
| 40 | Provider outage isolation | PASS |
| 41 | Secret exclusion | PASS |
| 42 | Backend unit tests | PASS |
| 43 | PostgreSQL integration tests | PASS |
| 44 | Backend race tests | PASS |
| 45 | Frontend tests | PASS |
| 46 | Frontend lint | PASS |
| 47 | Frontend typecheck | PASS |
| 48 | Frontend production build | PASS |
| 49 | Documentation | PASS |
| 50 | No Phase5 features | PASS |

QA API/Next listeners and owned PostgreSQL instance were stopped after verification. Temporary plaintext QA credential environment was removed; disposable private QA data remains outside the repository/public roots.
