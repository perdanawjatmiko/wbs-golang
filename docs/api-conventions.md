# API conventions

REST prefix `/api/v1`. JSON menggunakan success/data/meta atau success/error/meta. Meta selalu membawa request_id; X-Request-ID diterima hanya jika 1–64 karakter ASCII alfanumerik, underscore atau hyphen; lainnya diganti random 128-bit hex. ID dipantulkan pada response header dan log.

```json
{"success":true,"data":{"status":"ok"},"meta":{"request_id":"example-id"}}
```

```json
{"success":false,"error":{"code":"INTERNAL_ERROR","message":"Terjadi kesalahan internal."},"meta":{"request_id":"example-id"}}
```

| Endpoint | Status | Data |
| --- | --- | --- |
| GET /health/live | 200 | status=ok, service; tanpa DB query |
| GET /health/ready | 200 / 503 | status=ready atau NOT_READY; bounded DB ping |
| GET /api/v1/system/info | 200 | app_name, environment, version |
| GET /health | 200 | Compatibility legacy: `{ "status":"ok", "service":"wbs-backend" }`, tanpa envelope |

Error codes: INTERNAL_ERROR (500), NOT_READY (503), NOT_FOUND (404), METHOD_NOT_ALLOWED (405), ORIGIN_FORBIDDEN (403), HEADER_FORBIDDEN (403), BODY_TOO_LARGE (413), INVALID_BODY (400). Preflight sukses 204 tanpa body. Phase 1 codes are documented below. Tidak ada HTTP 200 untuk error API.

Tidak ada secret, database host, internal IP, stack trace atau config lengkap pada payload. Structured logs memuat timestamp, level, environment, request_id, method, route, status dan duration. Route tidak menyertakan query string. Request body, Authorization, cookies dan panic value tidak dicatat. Readiness failure mencatat kelas error aman; detail driver tidak dicetak. Request ID adalah correlation ID, bukan bukti identitas.

Security headers: nosniff, frame DENY, referrer no-referrer, no-store; production menambah HSTS. HTTPS endpoint HTTP disiapkan pada fase deployment. Phase 1 CORS mengaktifkan credentials hanya bagi allowlist eksplisit; tidak ada wildcard.


## Phase 1 REST contract

| Route | Methods | Access |
| --- | --- | --- |
| /api/v1/auth/login | POST | JSON + allowed Origin, rate limited |
| /api/v1/auth/me | GET | Active valid session |
| /api/v1/auth/logout | POST | Session + CSRF + Origin |
| /api/v1/admin/users | GET, POST | users.manage + Super Admin |
| /api/v1/admin/roles | GET, POST | roles.manage + Super Admin |
| /api/v1/admin/permissions | GET, POST | permissions.manage + Super Admin |
| /api/v1/admin/report-types | GET, POST | report-types.manage OR master.manage |
| /api/v1/admin/faqs | GET, POST | faqs.manage OR master.manage |
| /api/v1/admin/email-templates | GET, POST | email-templates.manage OR master.manage |
| Each admin resource /:id | GET, PATCH, DELETE | Same resource authorization |
| /api/v1/public/report-types | GET | No auth, active only |
| /api/v1/public/faqs | GET | No auth, active FAQ and active linked type only |

All mutations except login also require X-CSRF-Token + allowed Origin. JSON Content-Type required for POST/PATCH. Unknown fields, null on nonnullable field, invalid references/UUID/length and invalid placeholder are rejected. Create 201; read/update/delete 200 with envelope (delete data={deleted:true}); no password hash/session hash/raw session token in JSON.

List query: page=1, page_size=20 (1..100), search<=200 chars. page max100000. Relevant is_active=true|false filter for users/report-types/faqs. FAQ report_type_id=UUID filters a specific type, report_type_id=general filters NULL/general; omitted includes both. Public API ignores attempts to request inactive records. FAQ order sort_order,id; other lists stable id order.

List data: `{items: [...], total: N, page: 1, page_size: 20}` inside standard envelope. Public report types expose id/name/description; public FAQs expose id/report_type_id/question/answer/sort_order. Internal active flags/timestamps are omitted publicly.

DTO fields match schema except: users accepts password (never password_hash), roles accepts permission_ids. POST requires required fields; PATCH uses partial fields and rejects empty payload. is_active defaults true, sort_order defaults0, descriptions default empty, report_type_id defaults NULL. PATCH report_type_id=null selects general FAQ. No automatic missing-field reset.

```json
{"name":"Investigator custom","description":"Read-only reports","permission_ids":["<permission-uuid>"]}
```

Auth login/me data is `{user:{id,name,email,role_id,role_name,permissions},csrf_token}`. Logout returns status logged_out. Cookie carries secret token; csrf_token is safe to send as mutation header.

Phase 1 errors: INVALID_CREDENTIALS/UNAUTHENTICATED 401; FORBIDDEN/SELF_ROLE_CHANGE/CSRF_REJECTED 403; DUPLICATE/REFERENCE_CONFLICT/LAST_SUPER_ADMIN/PROTECTED_ROLE 409; VALIDATION_ERROR 422; JSON_REQUIRED 415; RATE_LIMITED 429. PostgreSQL FK 23503 and RESTRICT 23001 both map409. Internal details never included.

Next.js browser adapter `/api/backend/...` only forwards approved Phase 1 paths/methods to Go. It does not expose reports workflows, accept arbitrary upstream URLs, forward arbitrary cookies/headers, or replace backend authorization. Adapter error uses the same envelope with BRIDGE_ERROR and a safe message.
