# Changelog

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
