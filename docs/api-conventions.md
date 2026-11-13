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

Error codes: INTERNAL_ERROR (500), NOT_READY (503), NOT_FOUND (404), METHOD_NOT_ALLOWED (405), ORIGIN_FORBIDDEN (403), HEADER_FORBIDDEN (403), BODY_TOO_LARGE (413), INVALID_BODY (400). Preflight sukses 204 tanpa body. Business errors belum ada. Tidak ada HTTP 200 untuk error API.

Tidak ada secret, database host, internal IP, stack trace atau config lengkap pada payload. Structured logs memuat timestamp, level, environment, request_id, method, route, status dan duration. Route tidak menyertakan query string. Request body, Authorization, cookies dan panic value tidak dicatat. Readiness failure mencatat kelas error aman; detail driver tidak dicetak. Request ID adalah correlation ID, bukan bukti identitas.

Security headers: nosniff, frame DENY, referrer no-referrer, no-store; production menambah HSTS. HTTPS endpoint HTTP disiapkan pada fase deployment. CORS hanya allowlist eksplisit dan tidak mengaktifkan credentials.
