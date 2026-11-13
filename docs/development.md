# Development

Ikuti setup di README. File `.env` backend harus diekspor manual; Next.js memuat `.env.local` dari root frontend otomatis. `.env.example` hanya dokumentasi development dan aman di-commit. Jangan menaruh secret pada variabel NEXT_PUBLIC_*; nilainya dapat masuk bundle browser dan dibekukan saat build.

Frontend config: NEXT_PUBLIC_APP_NAME=WBS, API_BASE_URL=http://127.0.0.1:8080. API_BASE_URL harus HTTP(S) origin tanpa credential/path/query/fragment. Config dibaca saat server render dan divalidasi. Default local tersedia untuk development/build. Timeout client API 5000ms. Pada production server-to-server, pilih HTTPS bila melewati jaringan yang tidak tepercaya.

Server-side fetch terjadi dari proses Next.js ke Go, sehingga browser tidak perlu mengetahui alamat Go dan CORS browser tidak berlaku pada koneksi ini. Browser-side fetch baru memerlukan `NEXT_PUBLIC_API_BASE_URL` eksplisit (belum digunakan), dapat mengekspos alamat API publik, dan harus sesuai CORS allowlist. CORS tidak menggantikan authentication/CSRF. Phase 0 connectivity tetap langsung server-side. Phase 1 menambahkan adapter cookie terbatas untuk browser admin; lihat authentication.md.

Verifikasi manual integrasi:

1. Jalankan PostgreSQL development dan Go API; akses /health/live, /health/ready dan /api/v1/system/info.
2. Jalankan Next.js dengan API_BASE_URL yang menunjuk Go; halaman harus menampilkan Terhubung dan metadata.
3. Hentikan Go; reload halaman harus tetap 200 dan menampilkan Backend belum tersedia tanpa detail jaringan.
4. Hidupkan Go, lalu hentikan hanya database development setelah startup; /health/ready menjadi 503, /health/live dan system info tetap 200. Hidupkan kembali database development dan readiness pulih.
5. Uji OPTIONS dengan Origin yang dikonfigurasi dan X-Request-ID; origin lain harus 403.

Jangan menghentikan database bersama/production untuk pengujian. Gunakan instance terisolasi. Integration test membaca TEST_DATABASE_URL dan hanya ping; tidak menyiapkan schema, membuat database, migrasi atau menghapus data. Unit test tanpa PostgreSQL tetap dapat berjalan. Test lifecycle membuka loopback ephemeral dan membutuhkan environment yang mendukung bind. Race detector memerlukan toolchain/C compiler yang sesuai.

Rekomendasi sebelum Phase 1: siapkan dedicated local/test DB dan CI dengan PostgreSQL service terisolasi, simpan secret melalui mekanisme environment CI, tetapkan strategi migrasi dan kontrak auth/RBAC berdasarkan blueprint. Tidak ada deployment di Phase 0.


## Phase 1 local workflow

Setelah export backend .env dan siapkan database lokal yang sudah diprovisikan:

```sh
cd backend
go run ./cmd/migrate up
go run ./cmd/seed
# Untuk initial Super Admin, export ADMIN_SEED_EMAIL/PASSWORD/NAME secara aman lalu ulangi seed.
go run ./cmd/api
```

Frontend .env.local: API_BASE_URL=http://127.0.0.1:8080, APP_ORIGIN=http://localhost:3000. Backend CORS_ALLOWED_ORIGINS harus memuat http://localhost:3000. Akses browser melalui localhost:3000 konsisten, lalu /admin/login. Seed tidak membuat admin jika bootstrap environment kosong. Jangan hardcode/password default.

```sh
cd backend
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
# Dedicated test DB only; suite creates and drops a generated disposable schema.
TEST_PHASE1_DATABASE_URL='postgres://TEST_USER:TEST_PASSWORD@127.0.0.1:5432/wbs_test?sslmode=disable' go test -v ./tests
```

```sh
cd frontend
node --test tests/*.test.mjs
npm run lint
npx tsc --noEmit
npm run build
```

HTTP smoke test opt-in requires running Go/Next against dedicated test DB with a test Super Admin: set TEST_WEB_BASE_URL, TEST_ADMIN_EMAIL, TEST_ADMIN_PASSWORD, then `node --test tests/admin-http.test.mjs`. Test creates/updates/deletes one disposable report type via admin API. Without these variables it explicitly skips. Regular tests use node:test + existing TypeScript, no added frontend dependency.

Troubleshooting: 401 berarti login/session/inactive state; 403 berarti permission/Origin/CSRF mismatch; 409 berarti duplicate/reference/protected Super Admin. Missing schema: run migrations explicitly. Duplicate/referenced user: deactivate, do not purge sessions/historical rows merely to force deletion. Origin includes scheme/host/port and excludes trailing slash. Logout or password change invalidates current session, requiring login again. Login 429: wait fixed 15-minute window; do not weaken protections to bypass tests.


## Phase 2 public workflow

Apply migration up, start Go, use existing admin UI to create active report types/FAQs, then open /lapor. Landing at / now shows public CTAs/general FAQ; original connectivity moved to /development. Tracking at /tracking uses code+same contact with no URL query or persistent storage.

Use explicit APP_TIMEZONE=Asia/Jakarta and private UPLOAD_DIR (default backend/var/private/attachments when running from backend). Directories must be0700 and files0600; never serve the upload directory. UPLOAD_MAX_REQUEST_BYTES must match between Go and Next server config. Default file5MiB/request6MiB; do not simply raise global HTTP_MAX_BODY_BYTES for attachment handling. See file-storage.md for validation and crash reconciliation.

DB integration: set TEST_PHASE2_DATABASE_URL to dedicated test PostgreSQL, then `go test -v ./tests -run Phase2`. For full regression also set TEST_PHASE1_DATABASE_URL. Suites generate disposable schemas; main dev/production data is never targeted. HTTP public smoke opts in with TEST_PUBLIC_WEB_BASE_URL and requires a seeded active QA report type; it creates real reports and must run only on QA. See phase-2.md.

Troubleshooting: 409 quota counts only non-Done/non-Rejected reports for exact canonical contact; do not delete/reset counters to bypass it. 422 attachment: verify true format, extension, MIME and valid structure; strict PDF validator may reject nonconforming files. 429: wait window/upload slots, remember adapter-IP sharing. 503 SUBMISSION_UNCERTAIN or network timeout: do not automatically resend; code may already have been committed. Stage/final orphans require conservative operator reconciliation, not blind deletion.
