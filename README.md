# WBS & Ticketing Application

Phase 2: pelaporan publik, tracking, dan administrasi untuk Whistleblowing System & Ticketing Application. Spesifikasi bisnis hanya berasal dari [project_blueprint.md](project_blueprint.md). Sudah tersedia admin authentication, RBAC, master CRUD, dan complete database schema. Submission dan tracking publik sudah tersedia. Workflow status admin serta notifikasi belum diimplementasikan.

Stack: Go 1.26.2, Gin 1.12.0, PostgreSQL dengan pgx/v5; Next.js 16.4.6 App Router, React 19.3.0, TypeScript strict, Tailwind 4.3.3. Konfigurasi frontend existing dipertahankan.

```text
backend/
  cmd/{api,migrate,seed}/main.go
  internal/
    config/
    middleware/
    modules/{auth,health,master,reports}/
    platform/{database,httpserver,logger,migration,problem,response,security}/
    seed/
  migrations/
  tests/
  .env.example
frontend/
  src/
    app/
    components/ui/
    features/{admin,reports,system}/
    lib/{api,config}/
    types/
  .env.example
docs/
  architecture.md
  development.md
  api-conventions.md
  verification.md
```

Prasyarat: Go sesuai go.mod, Node.js >=20.9 (disarankan LTS yang mendukung Next.js), npm, PostgreSQL yang sudah diprovisikan, serta akses internet untuk dependency dan font Google existing saat build. Tidak ada database yang dibuat otomatis oleh aplikasi.

## Backend setup

Dari root:

```sh
cd backend
go mod download
cp .env.example .env
# Edit .env untuk database development milik Anda.
set -a
source .env
set +a
go run ./cmd/migrate up
go run ./cmd/seed
# Bootstrap Super Admin: export ADMIN_SEED_EMAIL/PASSWORD/NAME secara aman, lalu ulangi seed.
go run ./cmd/api
```

Backend membaca environment proses, tidak otomatis membaca `.env`. Gunakan shell tepercaya untuk `source`; kutip nilai yang mengandung karakter shell. Jangan commit secret. Port default 8080 dan host loopback 127.0.0.1. Startup memvalidasi config dan ping database dengan timeout; jika gagal, proses berhenti dengan pesan aman.

## Frontend setup

Jalankan terminal lain:

```sh
cd frontend
npm install
cp .env.example .env.local
npm run dev
```

Buka http://localhost:3000. Landing publik tersedia di `/`; halaman `/development` mempertahankan loading, status koneksi API, metadata aman, atau error jika backend tidak tersedia. Status koneksi API bukan indikator kesiapan database.

## PostgreSQL configuration

Gunakan database dan user lokal yang sudah disediakan administrator. Contoh development:

```dotenv
DATABASE_URL=postgres://wbs_user:password@localhost:5432/wbs_db?sslmode=disable
```

Untuk remote, gunakan URL dari administrator dan sertifikat CA yang sesuai; jangan membuat database remote otomatis. Contoh format (bukan credential): `postgres://USER:PASSWORD@DB_HOST:5432/DB_NAME?sslmode=verify-full&sslrootcert=/path/to/ca.pem`. Percent-encode karakter khusus pada credential URL. Jangan menyalin URL ke log/tiket. Production mewajibkan `sslmode=verify-full` dan origin CORS HTTPS. Terminasi HTTPS HTTP server dilakukan infrastruktur tepercaya pada fase deployment; fase ini tidak melakukan deployment.

Pool: maksimum total 20, target idle hangat 5, lifetime 30m, idle timeout 5m; koneksi awal 5s dan readiness 2s. Detail semantik pgxpool ada di [architecture](docs/architecture.md). Semua variabel dan default development tersedia pada [backend/.env.example](backend/.env.example).

## Health checks

```sh
curl -i http://127.0.0.1:8080/health/live
curl -i http://127.0.0.1:8080/health/ready
curl -i http://127.0.0.1:8080/api/v1/system/info
```

Liveness 200 tanpa query; readiness ping dengan timeout, 200 atau 503. `/health` mempertahankan payload legacy. Jika database gagal saat startup, HTTP server belum berjalan; 503 readiness berlaku ketika database gagal setelah startup.

## Testing

```sh
cd backend
gofmt -w cmd internal
go vet ./...
go test ./...
go test -race ./...
# Hanya dedicated local test DB; tidak menjalankan DDL/DML.
TEST_DATABASE_URL='postgres://USER:PASSWORD@127.0.0.1:5432/wbs_test?sslmode=disable' go test -v ./internal/platform/database
```

```sh
cd frontend
node --test tests/*.test.mjs
npm run lint
npx tsc --noEmit
npm run build
npm start
```

Tanpa `TEST_DATABASE_URL`, integration test database secara eksplisit SKIP. Unit test memakai dependency stub hanya untuk perilaku health, bukan pengganti integration test PostgreSQL. Hasil verifikasi tersedia di [docs/verification.md](docs/verification.md).

## Troubleshooting

- PostgreSQL unavailable: periksa service, host, port, firewall, role/database, password dan TLS; backend sengaja tidak start tanpa database.
- Frontend unavailable state: cek `API_BASE_URL` dari jaringan proses Next.js, backend aktif, dan timeout 5s. Reload halaman setelah memperbaiki koneksi.
- CORS 403: masukkan origin browser lengkap ke allowlist, tanpa trailing slash; `localhost` berbeda dari `127.0.0.1`.
- Build gagal mengunduh Geist: izinkan akses fonts.googleapis.com/fonts.gstatic.com atau rencanakan font lokal; konfigurasi font existing dipertahankan.
- Bind port gagal di sandbox: jalankan verifikasi lifecycle pada environment yang mengizinkan loopback.

Lihat [architecture](docs/architecture.md), [development](docs/development.md), dan [API conventions](docs/api-conventions.md). Changelog mencatat scope perubahan.


## Phase 1 administration

Buka `/admin/login` setelah bootstrap. Routes: /admin, /admin/users, /admin/roles, /admin/permissions, /admin/report-types, /admin/faqs, /admin/email-templates. Tidak ada report management UI.

Tambahan environment backend: DB_QUERY_TIMEOUT=5s, SESSION_TTL=8h, ADMIN_SEED_EMAIL/PASSWORD/NAME (CLI bootstrap saja). Frontend: APP_ORIGIN=http://localhost:3000 harus sama dengan browser origin dan terdaftar pada CORS_ALLOWED_ORIGINS backend. API_BASE_URL tetap server-only. Cookie HttpOnly dikelola Go dan diteruskan melalui adapter Next.js terbatas; tidak ada token localStorage.

Initial roles: Super Admin dan Investigator. Initial permissions dan aksesnya tersedia pada [RBAC matrix](docs/rbac.md). Investigator default deny sampai diassign permission. Seeder idempotent; tidak overwrite password/email existing dan tidak membuat default credential publik. Password 12–72 byte. Jangan commit bootstrap secrets.

Schema sembilan tabel blueprint ditambah admin_sessions infrastruktur. Migration/seeder tidak otomatis dijalankan API. Lihat [database schema and migration commands](docs/database-schema.md), [bootstrap and auth flow](docs/authentication.md), serta [Phase 1 report](docs/phase-1.md) untuk file, keputusan teknis, hasil pengujian, acceptance criteria dan keterbatasan.

Database integration Phase 1 opt-in: `TEST_PHASE1_DATABASE_URL=... go test -v ./tests` dari backend, hanya dedicated test DB. Suite membuat schema disposable dan menguji up/down di schema itu. Tidak menggunakan database development utama/production. HTTP smoke opt-in memerlukan TEST_WEB_BASE_URL dan test admin credentials; lihat [development](docs/development.md).


## Phase 2 public reporting

Routes: `/` landing/FAQ, `/lapor` submission dengan satu attachment opsional, `/tracking` code+contact; connectivity existing sekarang `/development`. Semua route admin tetap berjalan. Jalankan migrasi up terbaru dan buat jenis laporan aktif melalui admin sebelum mencoba form publik.

Submission menggunakan advisory transaction lock per canonical contact, quota5 active reports dan daily atomic code counter. Kode WBS-DDMMYY-2RANDOM+SEQUENCE, timezone eksplisit Asia/Jakarta default. Attachment private (5MiB default JPEG/PNG/PDF), tidak tersedia melalui static/download URL. Tracking hanya mengembalikan summary dan external notes, tanpa internal/admin/contact/storage metadata.

Tambahan env tersedia di contoh backend/frontend: APP_TIMEZONE, private UPLOAD_DIR, UPLOAD_MAX_FILE_BYTES/REQUEST_BYTES, UPLOAD_ALLOWED_TYPES/MAX_FILES, REPORT_UPLOAD_CONCURRENCY/REQUEST_TIMEOUT/SUBMIT_RATE_LIMIT/TRACK_RATE_LIMIT/RATE_WINDOW. UPLOAD_MAX_REQUEST_BYTES harus selaras di kedua server. Backend/var di-ignore.

Dokumentasi lengkap: [Phase 2](docs/phase-2.md), [submission](docs/report-submission.md), [tracking](docs/report-tracking.md), [private storage](docs/file-storage.md). Hasil tests, acceptance criteria, concurrency evidence dan batasan operasional tercantum di Phase 2. Tidak ada fitur workflow Phase 3–5 atau deployment.
