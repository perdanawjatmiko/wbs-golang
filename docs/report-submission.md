# Public report submission

Business rules come only from project_blueprint.md. This phase exposes POST `/api/v1/public/reports` (multipart/form-data) with name, contact, contact_type, report_type_id, detail, and optional attachment. Status is always Open. No initial history row, notification, report administration or workflow is created.

## Validation and normalization

Name: trimmed, 1..150 Unicode characters. Detail: trimmed plain text, 1..20,000 characters; HTML is never executed. Both reject invalid UTF-8 and NUL. report_type_id is UUID, must exist and remain active during insertion (`FOR SHARE` row lock prevents racing deactivation/deletion). Public error 422 does not expose internal data.

EMAIL: trim, parse exact email address without display name, lowercase entire address for storage/comparison. Dots and plus aliases remain unchanged. No provider-specific transformation or DNS lookup.

PHONE: explicit international + prefix required; remove ASCII space, hyphen, parentheses and dot; resulting value must be + followed by 8..15 ASCII digits, first digit nonzero. No assumed country, national-number conversion, extension, fuzzy matching or subscriber-ownership verification. The digit minimum is a technical validation choice. The same Go NormalizeContact function is used before persistence/quota and by tracking after inferring EMAIL from @, otherwise PHONE. Frontend checks only improve feedback; Go remains authoritative.

## Transaction boundary and quota

1. Streaming multipart parser stages and validates any file, without loading the full request into application memory.
2. Service validates and canonicalizes input.
3. Repository opens a READ COMMITTED transaction.
4. Obtain `pg_advisory_xact_lock` on first signed 64 bits of SHA-256(namespace + contact_type + NUL + canonical contact).
5. Count exact contact_type/contact records whose status NOT IN ('Done','Rejected'). If count>=5, rollback and return 409 REPORT_ACTIVE_LIMIT_REACHED with `Anda telah mencapai batas maksimal 5 laporan aktif.`
6. Verify active report type with shared row lock.
7. Generate next date sequence and code, insert report, then atomically publish staged file before commit.
8. Commit. Known failure removes staged/final file; ambiguous commit retains file for reconciliation rather than risking deletion of a committed attachment.

Same-contact transactions lock before count and insert before commit. Hash collisions between different contacts merely serialize them; the SQL count still uses exact normalized contact and type. The lock is transaction-scoped, released automatically by commit/rollback. No in-memory quota or session-level lock. Future status workflows must consider this locking contract when adding transitions that increase active count.

## Report code algorithm

Created timestamp comes from the injected clock after acquiring quota/type locks, persisted to both timestamps. Date uses explicit APP_TIMEZONE (default Asia/Jakarta), while stored timestamps are UTC instants. Counter key is date `YYYY-MM-DD`.

```sql
INSERT INTO report_code_counters(counter_date,last_sequence)
VALUES($1::date,1)
ON CONFLICT(counter_date) DO UPDATE
SET last_sequence=report_code_counters.last_sequence+1,updated_at=now()
RETURNING last_sequence;
```

CSPRNG chooses two unbiased characters from A-Z0-9. Format: `WBS-{DDMMYY}-{2 chars}{sequence padded to >=3 digits}`; example WBS-141126-X9001. Sequence 1000 stays 1000; no modulo, MAX(code), timestamp-only ID or process counter. New date starts at 1. Counter update and report insertion are in the same transaction; rollback restores counter.

Unique reports.code remains the final guard. INSERT uses ON CONFLICT(code) DO NOTHING RETURNING, then obtains a new sequence/prefix on collision, at most 3 attempts. Exhaustion rolls back all changes and returns 503 REPORT_CODE_UNAVAILABLE. A daily counter serializes sequence allocation across contacts until commit; file validation happens before this lock to keep the transaction short.

## Example

```sh
curl -X POST http://127.0.0.1:8080/api/v1/public/reports   -F 'name=Example Reporter' -F 'contact=reporter@example.com'   -F 'contact_type=EMAIL' -F 'report_type_id=<active-type-uuid>'   -F 'detail=Plain text details' -F 'attachment=@proof.pdf;type=application/pdf'
```

```json
{"success":true,"data":{"report_code":"WBS-141126-X9001","status":"Open","created_at":"2026-11-14T00:00:00Z","tracking_instructions":"Simpan kode laporan. Gunakan kode dan contact yang sama pada halaman tracking."},"meta":{"request_id":"example"}}
```

## Rate limits and timeout

Default submission: 10 attempts/IP/15m; tracking: 30/IP/15m; separate bounded in-memory maps, maximum 10,000 keys each, expired entries reclaimed, full map fails closed. Upload parser concurrency default2; excess 429 UPLOAD_BUSY. Limiters run before multipart parsing. No Redis. Through Next.js Go sees adapter IP because trusted proxies remain disabled, so limits conservatively apply to the adapter's users together. Rate windows reset on restart and do not coordinate across replicas; evaluate trusted proxy/distributed strategy before scaling.

REPORT_REQUEST_TIMEOUT defaults10s and must be less than HTTP_WRITE_TIMEOUT. DB_QUERY_TIMEOUT defaults5s, bounded by parent context. Read timeout also bounds network input. PDF parser honors context; image size/dimensions and decoder resource use are bounded. No automatic retry on an uncertain submission. Network/timeout or uncertain commit may mean a report was committed; frontend disables repeat submission in that state and advises tracking if code was received or contacting the operator. No idempotency receipt/recovery endpoint is added; never automatically replay multipart requests.
