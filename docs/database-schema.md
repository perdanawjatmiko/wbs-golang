# Database schema — Phase 1

PostgreSQL >=14. UUID native dengan `gen_random_uuid()` (tersedia tanpa extension pada versi ini) untuk semua PK; timestamp `timestamptz` menyimpan waktu absolut. Identitas session tetap UUID; cookie token berbeda dan tidak menjadi primary key publik. Tidak ada ORM.

## Versioned migrations

- `000001_blueprint.up.sql` / `.down.sql`: sembilan tabel blueprint, index dan check constraints.
- `000002_admin_sessions.up.sql` / `.down.sql`: tabel session infrastruktur.
- `schema_migrations`: metadata version/dirty milik golang-migrate.

| Table | Fields / constraints |
| --- | --- |
| roles | id UUID PK, name VARCHAR(100) NOT NULL UNIQUE nonblank, description TEXT, created_at/updated_at |
| permissions | id UUID PK, slug VARCHAR(100) UNIQUE NOT NULL dengan pola module.action, module_name VARCHAR(100) nonblank, description TEXT |
| role_permissions | role_id FK roles, permission_id FK permissions, composite PK(role_id,permission_id) |
| admin_users | id UUID PK, role_id FK roles wajib, name VARCHAR(150), email VARCHAR(254), password_hash VARCHAR(255), is_active default true, timestamps |
| report_types | id UUID PK, name VARCHAR(150), description TEXT, is_active default true, timestamps |
| faqs | id UUID PK, report_type_id FK nullable, question/answer TEXT nonblank, sort_order INTEGER >=0 default 0, is_active default true |
| email_templates | id UUID PK, event_key VARCHAR(100) UNIQUE uppercase event identifier, subject VARCHAR(255), body_html TEXT nonblank, updated_at |
| reports | id UUID PK, code VARCHAR(100) UNIQUE, name VARCHAR(150), contact VARCHAR(254), contact_type EMAIL/PHONE, report_type_id FK wajib, detail TEXT, attachment_url VARCHAR(2048) nullable, status Open/On Progress/Done/Rejected default Open, timestamps |
| report_histories | id UUID PK, report_id FK, admin_id FK, old_status/new_status dibatasi empat status, internal_note/external_note TEXT, created_at |
| admin_sessions | id UUID PK, admin_id FK, token_hash BYTEA UNIQUE tepat 32 byte, expires_at > created_at, revoked_at nullable, created_at |

Email unik case-insensitive menggunakan unique expression index `lower(email)`; API juga menormalisasi lowercase. Semua foreign key `ON DELETE RESTRICT`; tidak ada cascade delete. Nama role unik sesuai string PostgreSQL; API trim whitespace. Report type tidak memiliki hierarchy atau tambahan unique-name rule bisnis.

Index unique bawaan mencakup reports.code, permissions.slug, roles.name, email_templates.event_key dan session hash. Index tambahan: admin_users(lower(email)) unik, admin_users.role_id, reports.contact/status/report_type_id, report_histories.report_id/admin_id, faqs.report_type_id dan (sort_order,id), role_permissions.permission_id, admin_sessions.admin_id/expires_at. Tidak membuat index duplikat pada unique key.

## Commands

Dari `backend/` setelah environment diekspor:

```sh
go run ./cmd/migrate up
go run ./cmd/seed
```

Migration tidak otomatis berjalan saat API startup. golang-migrate mencatat versi dan mencegah pengulangan SQL yang sudah applied. Menjalankan up lagi tanpa perubahan aman. SQL tiap versi eksplisit dan transactional. Jangan memakai force pada dirty schema tanpa investigasi dan backup.

Down menghapus tabel dan hanya disediakan untuk database/schema disposable. CLI memerlukan `APP_ENV=test` dan `ALLOW_DISPOSABLE_DB_DOWN=yes`:

```sh
# ONLY dedicated disposable test database; never main development or production.
APP_ENV=test ALLOW_DISPOSABLE_DB_DOWN=yes go run ./cmd/migrate down
```

Tidak ada down terhadap database existing pada pekerjaan ini. Automated tests membuat schema unik `wbs_phase1_test_<random>` di PostgreSQL test, menerapkan up dua kali, menguji data, menjalankan down hanya di schema itu, lalu menghapus schema kosong. URL test harus khusus database test; jangan menunjuk production.

## Deletion and timestamps

PATCH memperbarui updated_at untuk tabel yang memilikinya. Data yang masih direferensikan ditolak 409; gunakan is_active=false pada users/report-types/faqs. Role assignment harus dihapus eksplisit melalui PATCH permission_ids sebelum role tanpa users dapat dihapus. Permission yang masih diassign juga 409. Session historis dapat membuat DELETE user konflik; deactivate adalah pilihan aman. Penghapusan/purge session bukan fitur admin Phase 1.

Reports dan report_histories hanya schema persiapan. Tidak ada endpoint, generator code, quota check, submission atau status workflow.


## Phase 2 additive migration

000003_report_foundation adds report_code_counters(counter_date DATE PRIMARY KEY, last_sequence BIGINT NOT NULL CHECK>0, updated_at TIMESTAMPTZ DEFAULT now()), reports_active_contact_idx on (contact_type,contact) WHERE status NOT IN ('Done','Rejected'), and report_histories_timeline_idx on (report_id,created_at,id). Reports/report_histories schemas from Phase 1 are reused unchanged. Existing reports.code unique index handles tracking and collision guard; no redundant code index. attachment_url now holds an opaque private storage key, not a public/absolute URL. No binary/extra attachment table.

Apply via existing `go run ./cmd/migrate up`. Down exists for disposable test schemas only and removes the added indexes/counter; never reset counters on a database retaining reports. There is no automatic rewriting of imported contact data. See report-submission.md for transaction, timezone, normalization and quota invariants.


## Phase 3 reuse

No migration added or existing migration modified. reports.updated_at is reused as an optimistic concurrency token, advancing by at least one microsecond on workflow/notes writes. report_histories.created_at equals that new server timestamp; status and history insert commit atomically. Existing active-contact partial index and history(report_id,created_at,id) index support shared quota locks and chronological pagination. No history update/delete API, extra status, assignment or infrastructure table. API guards rejection reason inside the workflow transaction; direct DB writes are outside API guarantees.
