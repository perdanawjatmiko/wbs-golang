# Architecture and technical decisions

Current Phase 1 additions: [Phase 1](phase-1.md), [authentication](authentication.md), [RBAC](rbac.md), [schema](database-schema.md). The sections below record Phase 0 decisions; Phase 1 extends CORS to credentialed explicit origins and adds a narrowly scoped Next.js cookie adapter.

## Audit baseline

Blueprint root dibaca lengkap sebelum perubahan dan tidak diubah. Existing repository bersih; backend memiliki Go module 1.26.2, Gin 1.12.0, composition root `cmd/api/main.go` dan satu `/health` tanpa lifecycle/config/tests. Frontend memiliki Next 16.4.6, React 19.3.0, App Router, TypeScript strict, ESLint 9, Tailwind 4.3.3 melalui loader Turbopack, font Geist dan halaman starter. Dependency/config valid dipertahankan; halaman starter diganti pemeriksaan konektivitas. Tidak ada fitur bisnis existing yang dihapus.

## Backend boundaries

Modular monolith: composition root memuat config, logger, pool, service, handler dan server. Transport Gin hanya ada di handler/middleware/router; health service hanya menggunakan context dan interface `Pinger`. Interface ini konkret untuk readiness dan unit testing. Tidak ada repository generik, ORM, global mutable database, atau package kosong.

Alur: HTTP -> request ID -> request log -> recovery -> security -> CORS -> body limit -> handler -> health service -> pgxpool. Database pool dimiliki composition root dan ditutup setelah HTTP selesai shutdown. SIGINT/SIGTERM memicu drain dengan context baru dan timeout; timeout memaksa server close.

## Decisions beyond the business blueprint

- PostgreSQL/Gin dipilih mengikuti instruksi Phase 0; pgx/v5 adalah satu-satunya tambahan dependency langsung backend. log/slog dan HTTP lifecycle menggunakan standard library.
- Environment dibaca dari proses tanpa dependency dotenv. Default development kecuali DATABASE_URL yang wajib tersedia. APP_ENV menerima local/test/staging/production. Validation error tidak mencetak nilai connection string.
- Startup fail-fast setelah ping; tidak ada mode database palsu atau degraded startup. Setelah startup, readiness 503 ketika database gagal; liveness tetap 200.
- `DB_MAX_OPEN_CONNS` memetakan `pgxpool.MaxConns`. pgxpool tidak menyediakan maksimum idle seperti sql.DB. Nama `DB_MAX_IDLE_CONNS` dipertahankan dari kontrak konfigurasi yang diminta, tetapi memetakan **MinIdleConns (target idle hangat)**; bukan hard idle ceiling. MaxConnIdleTime mengurangi koneksi idle berlebih, dan total tetap dibatasi MaxConns. Target idle harus <= total.
- Tambahan config: DB_CONNECT_TIMEOUT=5s, DB_READY_TIMEOUT=2s, HTTP_MAX_BODY_BYTES=1048576. Readiness timeout harus lebih kecil daripada write timeout. Pool connect dan readiness memakai context bounded serta cancellation.
- Production: Gin release mode, JSON logging, HSTS, explicit HTTPS CORS, PostgreSQL verify-full. HTTP TLS termination adalah tanggung jawab infrastruktur fase deployment. Credential/header/body/query tidak dicatat; log path menggunakan registered route untuk menghindari data sensitif dari URL liar. Panic value dan stack tidak diekspos/log karena dapat berisi secret; event recovery dan request ID tetap dicatat. Driver error disimpan melalui Unwrap tetapi string error aman.
- CORS tanpa credentials, wildcard dilarang, metode Phase 0 GET/HEAD/OPTIONS dan request header Content-Type/X-Request-ID. CORS bukan authentication/CSRF protection. Trusted proxies dinonaktifkan sampai ada konfigurasi proxy tepercaya yang nyata.
- Semua endpoint baru memakai envelope. `/health` mempertahankan JSON lama `{status, service}` sebagai pengecualian compatibility yang didokumentasikan.
- Request body dibaca maksimal limit sebelum handler, termasuk chunked body; ini menjamin 413 tanpa decoder bisnis. Batas default 1MiB; jangan menaikkan tanpa menilai concurrency dan memori.
- Frontend memakai Server Components, fetch no-store, AbortSignal timeout 5s dan runtime shape validation. Config API server-only; tidak ada proxy, state library, session atau client component tambahan. Suspense menyediakan loading, error fetch dirender sebagai state aman. Kesalahan konfigurasi environment ditolak secara jelas pada server.
- Cache Components tidak diaktifkan. Tailwind, loader dan font existing dipertahankan. UI primitive hanya StatusCard; folder kosong yang belum dibutuhkan tidak dibuat.
- Tidak ada migrasi atau tabel bisnis di Phase 0. Versi API metadata default `dev`, dapat diinjeksi build Go dengan `-ldflags '-X main.version=...'`.

## Phase 1 preparation

Rancang migrasi/rollback dan strategi identifier dari blueprint, pilih mekanisme admin auth yang disebut blueprint dengan threat model cookie/JWT, lalu definisikan matriks RBAC serta test transaksi. Implementasi tersebut hanya dimulai pada Phase 1; belum dibuat di foundation ini.


## Phase 1 decisions

UUID/PostgreSQL-native IDs, explicit embedded golang-migrate SQL and separate idempotent seed CLI. DB schema changes never run from HTTP startup. Handler/service/repository modules preserve Phase 0 boundaries. Master catalog limits field names and SQL identifiers; input values remain parameterized. Mutation repository transactions serialize guards with a PostgreSQL advisory lock; no global application mutex for RBAC state. Last Super Admin is enforced in application service under this lock, while all FK/unique/check invariants also exist in PostgreSQL.

Server-side HttpOnly cookie sessions use bcrypt cost 12 and SHA-256 hashes of 256-bit random session tokens. DB-driven RBAC resolves every request and transaction. Security administration is Super Admin-only. Formatting-only email HTML sanitation uses bluemonday; preview runs in an opaque sandbox iframe with network/script-denying CSP. No messages are sent.

Next.js remains App Router/Server Components by default. Interactive forms are Client Components; protected pages/layout resolve auth from Go. A finite route allowlist adapter is necessary to keep local browser cookies same-origin and forward Go Set-Cookie securely. It has no independent identity store or authorization logic. No global state library, UI dependency, Cache Components or JWT/localStorage added.


## Phase 2 additions

Reports module adds public submission/tracking under existing modular monolith boundaries. Contact quota uses deterministic SHA-256-derived transaction-level PostgreSQL advisory lock, atomic daily counter UPSERT and insert in the same transaction. Report type shared row lock prevents concurrent deactivation races. DTOs explicitly select public fields and exclude contact/internal/admin/storage metadata. Virtual initial timeline event avoids Phase 2 history workflow.

Private staging and atomic publication precede DB commit; service distinguishes definite failure cleanup from ambiguous commit retention. Streaming multipart body handling only bypasses global buffered limit for exact public upload route; existing JSON/admin limits remain intact. PDF structural validator uses stateless/offline context and resource limits. Shared platform limiter preserves auth compatibility.

Public Next adapter is independent of admin cookie adapter, finite allowlist, streaming byte limit/backpressure, no credential forwarding, no-store and no automatic retry. Landing/reporting/tracking share public layout; original connectivity remains /development. See [Phase 2](phase-2.md), [submission](report-submission.md), [tracking](report-tracking.md), [storage](file-storage.md) for decisions and limits.
