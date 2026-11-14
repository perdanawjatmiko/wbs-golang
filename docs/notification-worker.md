# Notification worker

Run migrations and idempotent seed, then run `go run ./cmd/worker` from backend with the same database/schema environment as API. The API does not launch a worker. `NOTIFICATION_WORKER_ENABLED=false` exits cleanly without opening a pool.

Worker polls every2s, at most25 jobs per cycle and5 concurrent sends by default. Each wave claims at most available concurrency so jobs never queue while spending their lease. Claim uses a short atomic CTE with SELECT FOR UPDATE SKIP LOCKED and UPDATE RETURNING. External I/O begins only after that statement commits and releases database row locks. Multiple workers can safely share PostgreSQL.

Claim increments attempts, assigns random UUID claim_token, worker identity and60s lease. Completion compares id/token/processing state and requires an unexpired lease. A stale owner cannot finalize even if its provider call returns later. Expired processing jobs below the attempt limit can be reclaimed; exhausted expired jobs become failed with LEASE_ATTEMPTS_EXHAUSTED. Finalization, expiry cleanup and manual retry use outbox-before-delivery locking. No session advisory lock is used by workers.

Config validation requires lease >= max provider timeout + two DB_QUERY_TIMEOUT windows +5s, and at least10s provider margin. Default SMTP15s/Telegram10s, DB5s and lease60s. A stalled database or scheduler beyond the margin can still lose ownership; its result is rejected and normal lease recovery handles it. Providers respect cancellation/deadlines. SIGINT/SIGTERM stops polling, cancels sends, joins all in-flight goroutines, persists safe outcomes with fresh bounded contexts and closes the pool.

## Retry

Default intervals30s,2m,10m,30m,2h; plus uniform positive jitter0–20%. Attempt n uses entry n, saturating at the last configured interval. Five total attempts means only the first four delays are normally used. Configure intervals and total attempts1..20. Telegram retry_after (capped24h) overrides the computed delay when greater. All retryable failures remain bounded by max_attempts. No immediate tight retry loop.

Transient: network/timeouts, SMTP4xx, Telegram5xx/429. Permanent: invalid recipient, unavailable/invalid DB template, unsupported payload, SMTP authentication or missing TLS, Telegram permanent4xx. Persisted error codes contain no remote response text, email, token or stack trace. Fix configuration before manual retry.

Provider-disabled policy: all eligible delivery identities are committed normally; workers exclude disabled channels from claims. New delivery status remains pending, attempts stay0 and no external connection is attempted. Enabling a channel later processes its backlog; review that backlog before enabling a provider. Disabling a channel does not rewrite the state of already-attempted deliveries; their pending/retrying/abandoned state remains durable until enabled or exhausted. Disabled never means sent. API/inbox/SSE continue working.

## Monitoring and administrative retry

notifications.manage is seeded for Super Admin, not Investigator. It can be explicitly delegated using existing RBAC. Monitoring returns global status counts and paginated status/channel filters; no recipient_ref/payload/secrets. Manual retry requires failed state, remaining attempt budget, and no previous manual retry. One audited manual retry per delivery prevents unbounded loops and does not reset attempts or create another delivery identity. Pending/processing/sent/exhausted deliveries return409. Fix the provider/template first. There is no bulk resend.
