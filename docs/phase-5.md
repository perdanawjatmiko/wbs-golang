# Phase 5 — Local Docker dan security hardening

## Status

Phase 5 adalah **belum lengkap**. Infrastruktur Compose dan hardening source sudah disiapkan, tetapi build/runtime Docker, koneksi database, seluruh test target, dan audit re-run dinamis tidak dapat diverifikasi. Tidak ada deployment atau akses production.

## Arsitektur lokal

Compose mendefinisikan tepat tiga service aplikasi: `frontend`, `backend`, `worker`. Backend dan worker berbagi image Go multi-stage; frontend menggunakan Next standalone. PostgreSQL tetap server eksternal yang sudah ada. Tidak ada service/volume PostgreSQL atau Redis. Network Compose memakai bridge internal; port UI/API hanya bind ke loopback host. Volume bernama `attachments` menyimpan file laporan privat.

Container menjalankan user non-root, filesystem read-only, capabilities dropped, `no-new-privileges`, batas proses/memori/CPU lokal, dan `/tmp` terbatas. SMTP dan Telegram dipaksa nonaktif pada Compose lokal. Migration dan seed tidak berjalan otomatis saat startup.

## Menjalankan secara lokal

1. Salin `.env.docker.example` menjadi `.env.docker` dan isi `DATABASE_URL` dengan alamat PostgreSQL development yang disetujui. Dari container, `127.0.0.1` adalah container itu sendiri. Untuk DB di VM gunakan private IP/DNS; untuk DB pada host macOS Docker Desktop dapat memakai `host.docker.internal` jika listener/firewall mengizinkan.
2. Pastikan DB/user development memang telah diidentifikasi, backup sesuai prosedur tim, akses jaringan privat/TLS/role telah disiapkan oleh pemilik DB. Jangan gunakan URL yang belum diketahui lingkungannya.
3. Build: `docker compose --env-file /dev/null --env-file .env.docker.example build` (Compose interpolation harus menggunakan file yang benar-benar diisi; `.env.docker.example` hanya nilai dummy dan tidak dapat mengautentikasi DB).
4. Start setelah konfigurasi diperiksa: `docker compose up -d`.
5. Smoke helper baca-saja mensyaratkan `WBS_APPROVED_DEV_DATABASE=yes`, `.env.docker`, dan memeriksa health API/Next serta koneksi DB/migration status. Helper tidak menjalankan DDL, seed, atau bootstrap.
6. Migrasi eksplisit dari host: `cd backend && set -a && source ../.env.docker && set +a && go run ./cmd/migrate status`, kemudian `go run ./cmd/migrate up` hanya pada development DB teridentifikasi. Seeder/bootstrap hanya dengan perintah di `docs/development.md` dan secret yang dimasukkan aman. Container tidak menjalankan migration/seed sendiri.
7. Hentikan service dengan `docker compose down`; perintah ini tidak menghapus volume. Jangan gunakan `down -v` jika ingin mempertahankan attachment.

Perintah smoke/migration memerlukan binary, environment dan akses ke DB yang benar. Perintah tersebut **belum dijalankan** pada sesi ini. Tidak ada database server yang diprovisikan, diubah, atau disentuh.

## Batas koneksi PostgreSQL eksternal

Periksa pada jaringan development yang disetujui: alamat sumber yang benar-benar terlihat oleh server (tidak diasumsikan sama dengan IP container), `listen_addresses`, rule `pg_hba.conf` paling sempit, firewall, kredensial role least-privilege, TLS/CA dan timeout. Jangan membuka `0.0.0.0/0`. Perubahan server hanya dilakukan DBA/pemilik, bukan otomatis oleh repo. Koneksi aktual tidak diverifikasi karena endpoint/credential database tidak diberikan dan external DB tidak boleh diasumsikan aman untuk dimodifikasi.

## Health dan persistent data

Backend healthcheck `/app/check live` hanya mengukur liveness proses; `ready`/`db` tersedia sebagai mode CLI untuk pengecekan eksplisit tanpa schema change. Worker tidak mengekspos port. Worker membuka koneksi dengan retry startup bounded dan shutdown melalui signal. Volume attachment dipasang hanya pada backend yang menulis file; persistensi/restart belum dapat diverifikasi tanpa runtime Docker.

## Bukti dan keterbatasan

`docker compose config` divalidasi oleh pemeriksa kebijakan statis: tepat tiga service, tidak ada database/Redis/port DB, hanya volume attachment, port host loopback, dan hardening container yang dikonfigurasi. Ini bukan bukti image dapat dibangun atau container berjalan. Build Docker gagal pada ekstraksi layer dengan `no space left on device`; tidak dilakukan prune karena dapat menghapus resource Docker pengguna. Source test/lint/build aplikasi tidak dijalankan karena sandbox OS yang diwajibkan audit skill tidak tersedia. Rincian ada pada [hasil security](security-test-results.md) dan [docker-local](docker-local.md).

## Risiko tersisa

- Belum diketahui apakah versi image/toolchain yang dipilih kompatibel setelah build aktual.
- Database networking/authorization/TLS/privileges dan persisted attachment belum dites.
- Cookie, CSRF, SSE, API, worker, dan provider behavior belum diuji runtime.
- CAPTCHA tidak ditambahkan: scope ini tidak mensyaratkan integrasi vendor; CAPTCHA akan menambah pihak ketiga dan mengirim metadata publik. Rate limit bukan perlindungan penuh dari bot yang berganti alamat/kontak.
- Audit sumber dan review perubahan bukan pengganti final-clean reviewer dan verifikasi dinamis. Run audit ditandai `incomplete` secara eksplisit.

## Readiness

Konfigurasi source siap ditinjau dan dicoba pada mesin development setelah ruang Docker tersedia, lalu gunakan DB test/disposable yang disetujui. Jangan menyatakan Phase 5 selesai atau menerima perubahan deployment sampai semua BLOCKED pada checklist terverifikasi. Blueprint dan migration 000001–000004 dipertahankan; tambahan 000005 hanya mendukung rate limit.
