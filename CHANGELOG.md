# Changelog

## 2026-11-16 — Phase 7 master data enhancements

- Additive migration 000006 adds two-level report type hierarchy, preserves flat legacy rows as roots, guards FAQ parent-only references, and adds bounded import preview storage plus temporary-password lifecycle metadata.
- Extend public report submission to validate an active parent/child pair while retaining the existing stored child FK and report/tracking contracts.
- Add permission-protected XLSX template/export, non-mutating preview and transactional idempotent upsert for users, report types, and FAQs; passwords/credentials are excluded and new imported users remain inactive pending administrator provisioning.
- Add CRUD dialogs, import preview UI, dedicated Tiptap email-template pages, and cascading public report type selection while retaining WBS visual tokens.
- Add Phase 7 architecture, migration, import/export, security, and test documentation. Database migration and browser visual verification remain blocked pending disposable QA infrastructure.

## 2026-11-14 — Phase 4 multi-channel notifications

- Add transactional outbox, durable realtime events, per-admin inbox, deduplicated deliveries and audited bounded manual retry through additive migration000004.
- Integrate report/status commits without external calls or changes to quota/version/history/privacy invariants.
- Add dedicated PostgreSQL worker with SKIP LOCKED, expiring token-owned claims, recovery, bounded retry/jitter and graceful shutdown.
- Add verified SMTP with safe DB templates and fixed HTTPS Telegram client with429 handling; disabled channels remain pending.
- Add report.view-protected durable SSE/inbox, notifications.manage monitoring, Next notification bell/monitoring and secure incremental adapter.
- Add local-provider, PostgreSQL failure/concurrency/security and actual Go–Next HTTP tests; no real delivery or Phase5 deployment.


## 2026-11-13 — Phase 3 admin workflow and ticket management

- Add report.view-protected list/detail/dashboard/options and authorized bounded private attachment streaming.
- Add report.update-protected transactional status/notes workflow, mandatory rejection reason, precise optimistic version and shared quota-lock reactivation safety.
- Add atomic immutable API histories with admin/internal/external projections; hide internal-only events publicly and preserve creation event for Open notes.
- Upgrade admin dashboard/navigation, list/filter/pagination/detail and status/notes forms with conflicts/refetch; extend finite cookie adapter with response caps.
- Add isolated PostgreSQL workflow/RBAC/rollback/concurrency tests, frontend markup/logic and real HTTP smoke coverage, and Phase3 docs.
- Reuse migrations/schema/dependencies and Phase0–2 behavior; no Phase4–5 features or deployment.

## 2026-11-13 — Phase 2 public reporting and tracking

- Additive counter/quota/timeline indexes; reuse reports/history tables without rewriting data.
- Add canonical email/international-phone normalization, transactional advisory quota lock and atomic daily ticket codes with bounded collision retry.
- Add bounded streaming multipart, private staged JPEG/PNG/PDF validation/storage and definite-failure cleanup/ambiguous-commit retention.
- Add public code+contact tracking and external-only timeline, independent public rate limits and upload concurrency cap.
- Add public landing/reporting/tracking/interactive FAQ and credential-free streaming Next adapter; retain connectivity at /development and admin routes.
- Add PostgreSQL concurrency/security/rollback tests, frontend logic/rendering/HTTP smoke tests and Phase 2 documentation.
- No admin workflow, status mutation UI, notifications, CAPTCHA or deployment. Blueprint unchanged.

## 2026-11-13 — Phase 1 administration

- Tambah versioned SQL migrations untuk 9 tabel blueprint dan admin_sessions, serta CLI migration/seeding terpisah.
- Tambah idempotent role/permission/template seeds dan environment-only Super Admin bootstrap tanpa overwrite akun existing.
- Tambah bcrypt, hash-only cookie sessions, expiry/revocation/rotation, CSRF, rate limit login, dan DB-driven RBAC.
- Tambah transactional CRUD admin users/roles/permissions/report types/FAQs/email templates, public active master read APIs, validation/pagination/search, safe HTML dan FK conflicts.
- Tambah protected Next.js admin shell, login/logout, permission navigation, CRUD forms/tables serta sandbox email preview dan explicit cookie adapter.
- Tambah unit/database/concurrency/frontend/HTTP integration tests dan dokumentasi schema/auth/RBAC.
- Blueprint dan health compatibility dipertahankan; tanpa workflow Phase 2–5 atau deployment.

## 2026-11-13 — Phase 0 foundation

- Audit blueprint dan stack existing; pertahankan Go/Gin dan Next.js App Router/TypeScript/Tailwind.
- Tambah config typed tervalidasi, pgxpool, structured logging dan graceful HTTP lifecycle.
- Tambah request ID, recovery, CORS allowlist, security headers, body limit dan response envelope.
- Tambah liveness/readiness/system info; pertahankan payload /health legacy.
- Tambah server-side typed API client, timeout, shape validation dan UI connectivity dengan loading/success/error.
- Tambah unit/integration tests serta panduan environment, architecture, API dan development.
- Tidak mengubah blueprint, tidak menambah business tables/features, tidak melakukan deployment.

## 2026-11-16 — Phase 5 local environment and security hardening (verification incomplete)

- Add three-service local Docker Compose (frontend/backend/worker), multi-stage non-root images, private attachment volume, health CLI, sample config, and read-only smoke helper. PostgreSQL remains external; no database container, auto-migration, or auto-seed.
- Add additive migration 000005 and bounded PostgreSQL-backed distributed rate admission for login, submission, and tracking; validate trusted proxy configuration and cookie policy; bound worker startup retries and public adapter responses.
- Add frontend security headers/health route, regression tests, Docker/security documentation, and private audit artifacts.
- Patch Go toolchain and Node image base to reviewed security releases; application library dependencies unchanged.
- Verification is incomplete: Docker image extraction failed for lack of storage; target tests/build and runtime/database checks were blocked by audit sandbox/database prerequisites. No production resources or real providers used. See `docs/security-test-results.md`.

## 2026-11-16 — Phase 6 UI/UX overhaul

- Add semantic WBS Design System v1 tokens, responsive portal/admin shell, inline icon and status primitives without adding UI dependencies.
- Redesign public landing, report/tracking presentation and success states; admin dashboard, report table/detail, master data styling, notification surfaces and login.
- Preserve all API, permission, authentication, CSRF, tracking privacy, report and notification contracts.
- Add UI design/navigation tests and design/component/QA documentation. Lint, strict typecheck, frontend tests and production build pass; 4 environment-gated HTTP tests skipped. Visual spot-check limited to 375px public landing/login; authenticated/admin and populated backend states need QA.

## 2026-11-16 — Phase 6F incremental shadcn/ui integration

- Configure official shadcn CLI for current Next.js 16/Tailwind 4 CSS variables and existing WBS aliases without replacing the current theme.
- Add Radix-backed AlertDialog and WBS-styled Button primitives; use AlertDialog for the existing master-data delete confirmation while preserving request/CSRF behavior.
- Add shadcn NativeSelect for all application select fields, retaining native HTML semantics and existing form contracts without another UI dependency.
- Map shadcn semantic utility tokens to existing WBS values; retain local form controls, StatusBadge, layouts, tables, notifications and business components.
- Add focused primitive/token/select contract tests and Phase 6F documentation. Screenshot comparison and keyboard/WCAG verification remain blocked.
