# Changelog

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
