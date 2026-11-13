# Development

Ikuti setup di README. File `.env` backend harus diekspor manual; Next.js memuat `.env.local` dari root frontend otomatis. `.env.example` hanya dokumentasi development dan aman di-commit. Jangan menaruh secret pada variabel NEXT_PUBLIC_*; nilainya dapat masuk bundle browser dan dibekukan saat build.

Frontend config: NEXT_PUBLIC_APP_NAME=WBS, API_BASE_URL=http://127.0.0.1:8080. API_BASE_URL harus HTTP(S) origin tanpa credential/path/query/fragment. Config dibaca saat server render dan divalidasi. Default local tersedia untuk development/build. Timeout client API 5000ms. Pada production server-to-server, pilih HTTPS bila melewati jaringan yang tidak tepercaya.

Server-side fetch terjadi dari proses Next.js ke Go, sehingga browser tidak perlu mengetahui alamat Go dan CORS browser tidak berlaku pada koneksi ini. Browser-side fetch baru memerlukan `NEXT_PUBLIC_API_BASE_URL` eksplisit (belum digunakan), dapat mengekspos alamat API publik, dan harus sesuai CORS allowlist. CORS tidak menggantikan authentication/CSRF. Tidak ada proxy API tambahan.

Verifikasi manual integrasi:

1. Jalankan PostgreSQL development dan Go API; akses /health/live, /health/ready dan /api/v1/system/info.
2. Jalankan Next.js dengan API_BASE_URL yang menunjuk Go; halaman harus menampilkan Terhubung dan metadata.
3. Hentikan Go; reload halaman harus tetap 200 dan menampilkan Backend belum tersedia tanpa detail jaringan.
4. Hidupkan Go, lalu hentikan hanya database development setelah startup; /health/ready menjadi 503, /health/live dan system info tetap 200. Hidupkan kembali database development dan readiness pulih.
5. Uji OPTIONS dengan Origin yang dikonfigurasi dan X-Request-ID; origin lain harus 403.

Jangan menghentikan database bersama/production untuk pengujian. Gunakan instance terisolasi. Integration test membaca TEST_DATABASE_URL dan hanya ping; tidak menyiapkan schema, membuat database, migrasi atau menghapus data. Unit test tanpa PostgreSQL tetap dapat berjalan. Test lifecycle membuka loopback ephemeral dan membutuhkan environment yang mendukung bind. Race detector memerlukan toolchain/C compiler yang sesuai.

Rekomendasi sebelum Phase 1: siapkan dedicated local/test DB dan CI dengan PostgreSQL service terisolasi, simpan secret melalui mekanisme environment CI, tetapkan strategi migrasi dan kontrak auth/RBAC berdasarkan blueprint. Tidak ada deployment di Phase 0.
