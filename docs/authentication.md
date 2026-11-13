# Admin authentication

Go adalah sumber autentikasi/otorisasi tunggal. Next.js tidak menyimpan user/session database sendiri. Password bcrypt cost 12 dengan panjang 12–72 byte, tidak dipotong diam-diam. Unknown email menjalankan bcrypt terhadap dummy hash dengan cost sama; invalid password, unknown email dan inactive user mengembalikan 401 `INVALID_CREDENTIALS` dengan pesan identik.

## Bootstrap

Jalankan migrations dan seeder sebelum login. Untuk membuat initial Super Admin, pasang environment melalui shell/secret manager tepercaya:

```dotenv
ADMIN_SEED_EMAIL=your-admin@example.com
ADMIN_SEED_NAME=Your Admin
ADMIN_SEED_PASSWORD=<unique secret, 12..72 bytes>
```

```sh
cd backend
# Export configuration securely; do not paste passwords into shell history.
go run ./cmd/seed
```

Tanpa ketiga nilai tersebut tidak ada initial admin dibuat. Konfigurasi parsial ditolak. Seeder tidak mencetak password dan tidak mengubah password/name/role/active milik email existing; ia juga tidak otomatis mempromosikan akun existing. Nilai bootstrap hanya dibaca CLI seed, tidak API. Unset ADMIN_SEED_* setelah bootstrap. Password tidak boleh di-commit.

## Session lifecycle

1. POST `/api/v1/auth/login` menerima JSON email/password dan Origin dari allowlist.
2. Password diverifikasi, lalu status aktif dan hash diverifikasi ulang di transaksi untuk mencegah race password change. Session cookie lama yang dipresentasikan direvoke.
3. Token acak CSPRNG 256-bit dikirim hanya sebagai `wbs_session` cookie. Database menyimpan SHA-256 token, bukan token mentah. Respons JSON hanya membawa user dan csrf_token.
4. Cookie host-only, Path=/, HttpOnly, SameSite=Lax; Secure wajib pada APP_ENV=production. Default umur absolut 8h, SESSION_TTL menerima 1m..24h. Tidak ada sliding expiration atau localStorage token.
5. Setiap request memeriksa hash, expiry, revoked_at, is_active dan permission DB terkini. Tidak ada cache permission pada session.
6. POST logout merevoke row dan menghapus cookie. Perubahan password atau nonaktifkan user merevoke seluruh session user dalam transaksi. Permission/role berubah berlaku pada request berikutnya.

Session kadaluwarsa/revoked/inactive menghasilkan 401 `UNAUTHENTICATED`. Frontend mengarahkan render server ke login; request browser 401 mengosongkan tabel/form dan menampilkan state sesi berakhir. Logout mengarahkan ke login dan merefresh router.

## Cookie and CSRF flow

Admin browser menggunakan `/api/backend/...` di Next.js same-origin. Adapter hanya mengizinkan route auth, enam resource admin dan public lookup yang eksplisit. URL upstream selalu berasal dari API_BASE_URL server-only; tidak ada arbitrary URL/header forwarding. Hanya cookie wbs_session diteruskan, dan hanya Set-Cookie dengan nama itu dikembalikan. Adapter tidak menentukan identitas atau hak akses.

APP_ORIGIN harus sama persis dengan origin browser (tanpa trailing slash) dan masuk CORS_ALLOWED_ORIGINS Go. Contoh localhost:3000 harus dipakai konsisten; 127.0.0.1 adalah origin lain. Login diwajibkan JSON + exact allowed Origin untuk mencegah login CSRF. Mutation terautentikasi memerlukan Origin allowlist dan header `X-CSRF-Token`. Go menurunkan token CSRF dengan SHA-256(domain separator + secret session token), memverifikasinya constant-time, dan mengembalikannya lewat login/me. Token CSRF boleh dibaca JavaScript; token session tetap HttpOnly.

SSR protected routes memanggil auth/me langsung ke Go dengan cookie session saja, cache:no-store, timeout dan redirect:error. Tidak pernah menserialisasi cookie ke props. Client CRUD memakai adapter dengan CSRF header. CORS credentials hanya pada explicit origin; tidak ada wildcard. SameSite bukan satu-satunya proteksi CSRF.

## Login rate limit

In-memory mutex-protected fixed window: maksimal 10 login attempts per client IP per 15 menit, map maksimal 10.000 key, expired bucket direklamasi dan kapasitas penuh default deny. Berlaku untuk sukses maupun gagal; 429 dan Retry-After. Trusted proxies tetap nonaktif. Melalui adapter Next.js, Go melihat IP adapter sehingga limit dibagi pengguna pada instance tersebut; ini disengaja lebih konservatif. Belum ada distributed limiter; evaluasi trusted proxy/address handling sebelum scale-out, tanpa menambah Redis di fase ini.

## Security boundaries and limits

Tidak ada forgot-password, verification, email delivery, auth JWT atau secret di NEXT_PUBLIC_*. Tidak log body/password/session/cookie/Authorization. Error internal generik dan log hanya kelas error. Production perlu HTTPS browser dan transport privat/HTTPS Next->Go sesuai infrastruktur; deployment tidak dilakukan di Phase 1. Password recovery darurat belum tersedia; jangan menonaktifkan akses Super Admin yang valid. Row session expired/revoked dipertahankan; retention/purge operasional belum diimplementasikan.
