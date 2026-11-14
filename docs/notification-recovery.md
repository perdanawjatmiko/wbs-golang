# Notification recovery

| Failure | Durable outcome / recovery |
| --- | --- |
| Business validation/version/history/outbox failure | Whole business transaction rolls back; no external notification |
| SMTP unavailable | Business commit succeeds; retrying delivery with bounded backoff |
| Telegram429 | Retry scheduling respects provider delay; no tight loop |
| Worker exits after claim | Processing lease expires; another worker claims with a new token |
| Provider accepts, worker exits before persistence | Lease recovery can send a duplicate; no exactly-once promise |
| Old worker returns after reclaim | Compare-and-update rejects stale token/expired lease |
| Attempts exhausted | failed; abandoned processing finalized with safe code |
| Invalid template/authentication | failed; fix trusted config/template, then one audited bounded manual retry if budget remains |
| Providers disabled | pending, no attempt/network; review backlog before enabling |
| Admin loses access/session | Inbox403/401; open SSE closes without further report payload |
| SSE/backend/network restart | Native reconnect and bounded durable replay; inbox reset if backlog exceeds bound |

Operational sequence: inspect status counts and safe error codes through authorized monitoring; check worker process/database/migrations, trusted provider settings, and DB templates without logging credentials; correct root cause; allow scheduled retries or perform a permitted manual retry. Never repeatedly reset attempts, change identities or mark jobs sent merely to clear a queue. No destructive reset tooling is included.

Delivery identity is unique(event ID,channel,recipient identity). Claim token protects ownership but cannot atomically commit an external SMTP/Telegram side effect with PostgreSQL. Stable Message-ID and durable delivery status suppress known duplicates only. This is at-least-once processing with best-effort duplicate suppression, not exactly-once delivery. Provider idempotency is not assumed.

Known limits: polling latency, single append lock throughput, SQL fanout proportional to authorized admin count, durable table growth without retention automation, current configured Telegram group/current DB template at dispatch, and possible duplicate sends after ambiguous provider acceptance. No browser visual automation, live provider test, production proxy or production load verification is claimed. A future retention/archival plan must respect audit/privacy needs and be explicitly approved before deleting data.
