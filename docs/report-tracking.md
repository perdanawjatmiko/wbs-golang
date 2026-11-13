# Secure public tracking

POST `/api/v1/public/reports/track` accepts JSON:

```json
{"report_code":"WBS-141126-X9001","contact":"reporter@example.com"}
```

Exact uppercase code format (outer whitespace trimmed) and canonical contact must both match. Contact type is inferred from @ versus international phone, then normalized with the same Go function as submission. A single parameterized query filters report code + contact_type + contact before returning any data. Unique code index already makes this lookup efficient; no redundant code index is added.

Unknown code, wrong contact, malformed code/contact and mismatched pair all return 404 REPORT_NOT_FOUND with `Laporan tidak ditemukan. Periksa kode dan contact.` Invalid JSON/content type uses ordinary validation errors independent of report existence. Do not expose whether the code exists. No OTP, public login or contact-only lookup.

Response data: report_code, name, report_type (display name), detail (plain text), status, created_at, updated_at, timeline. It omits contact, internal IDs, attachment storage key/path, internal_note, admin_id, admin name and session metadata. The matched report's type is returned even if later deactivated; public active master lists remain filtered separately.

Timeline is built from report_histories using only new_status, external_note and created_at, ordered created_at/id ascending. If no history begins with Open creation event, a virtual Open event at reports.created_at is prepended. If the first stored history is already Open, no duplicate initial event is added. Phase 2 does not insert histories or provide status changes.

Frontend `/tracking` uses client POST with credentials omitted, no-store and timeout. Code/contact stay in memory and JSON body; never query string or persistent storage. Server responses use no-store; Next adapter adds no-store, private. Page HTML is a static form without report data, while API results are private uncached responses. React renders detail/external notes as escaped text, with whitespace preserved. Changing credentials or submitting another lookup clears the previous result. Browser history does not contain report credentials.

Tracking rate limit default30/IP/15m is independent from submission, without Redis. Public failure messages are generic; application logs record method, registered route, status, duration and request ID rather than contact, code, body or attachment. JSON errors never include driver details or stack traces.

No attachment download is implemented in Phase 2. Attachment paths are not returned by tracking. A future download must independently verify code+contact (or authorized admin), stream bounded data with attachment disposition/nosniff, and never accept raw filesystem paths. See file-storage.md.
