# Architecture and technical decisions

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
