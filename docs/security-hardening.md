# Phase5 security hardening

These are source implementations with independent source review. New runtime behavior is NOT certified: OS sandbox/image builds are blocked by Docker storage exhaustion. Existing Phase0–4 test results are historical, not rerun evidence.

## Distributed request admission

API composition always injects PostgreSQL Gate into login, public submit and tracking. Legacy in-memory constructors remain only for compatible standalone callers/unit fixtures; there is no runtime fallback when PostgreSQL fails. Missing migration/table, timeout or DB failure returns503 RATE_LIMIT_UNAVAILABLE before password hashing/upload/tracking. A denied request returns429 RATE_LIMITED with remaining-window Retry-After; Next adapters preserve that header. No active-report quota rule changes.

Migration000005 adds infrastructure-only rate_limit_buckets. Scope is one of admin_login/report_submit/report_track, plus a fixed integer bucket0..65535. No IP, email, contact, password or secret is stored. Canonical IP handles IPv4-mapped IPv6 and equivalent IPv6 representations, then SHA-256 maps it to RATE_LIMIT_BUCKETS slots(default16384). Collisions conservatively share a quota; they never grant extra requests. All replicas must share slots/threshold/window config. The hard schema bound is3*65536 rows; counters saturate at threshold+1(max10001). Expired buckets are reused, so unique-IP rotation cannot create unbounded key rows. Ordinary PostgreSQL MVCC/WAL/autovacuum capacity remains a DBA responsibility; logical row cardinality is not a claim of fixed physical disk bytes.

One parameterized INSERT ON CONFLICT UPDATE RETURNING atomically initializes/resets/increments a fixed window under the row lock. statement_timestamp is consistent across count, expiry and retry calculation. There is no count-before-insert, full-table request scan or session-level advisory lock. RATE_LIMIT_TIMEOUT(default1s) bounds DB admission. Login defaults10/15m; report defaults10 submit and30 tracking/15m remain unchanged and configurable. Migration000001–000004 unchanged;000005 is never automatic on startup.

## Client identity and proxy trust

Default ignores all forwarded-IP headers. HTTP_TRUSTED_PROXY_CIDRS is empty unless an owner explicitly configures controlled private/loopback CIDRs; global/public prefixes are rejected. Gin only interprets X-Forwarded-For from those peers. Never trust a subnet merely because it is private or configure frontend peer trust while passing attacker-supplied headers unchecked. The Next adapter deliberately does not invent a client IP from incoming headers: without a controlled ingress that overwrites them, their provenance is unknown. Docker frontend users therefore share its backend-facing admission bucket, an existing documented conservative aggregate policy. New PostgreSQL state makes that policy consistent across API replicas; it does not create authenticated anonymous identities. Shared-peer false positives/availability tradeoffs remain explicit. Deployment-specific ingress identity is unverified and no extra proxy service is introduced.

## Sessions, browser policy and response bounds

COOKIE_SECURE defaults true in production and cannot be disabled there. Plain HTTP localhost defaults false; explicitly enable true for HTTPS staging. HttpOnly/SameSite=Lax, token rotation/hash storage, session revocation, live RBAC, CSRF and Origin architecture remain intact.

Backend JSON/download responses add restrictive default-src none/frame-ancestors none CSP and Permissions-Policy, retaining existing nosniff/referrer/no-store. Frontend adds frame-ancestors none, object-src none, base-uri/form-action self, DENY, nosniff and restricted capabilities. No HSTS on plain localhost. No fabricated nonce/unsafe-inline script policy is introduced; full script CSP with Next nonces remains a separate browser-validated task. Existing opaque template-preview iframe remains sandboxed. Public adapter now bounds upstream JSON16MiB rather than unbounded text buffering; admin adapter keeps bounds and forwards valid Retry-After.

## Worker and containers

Worker DB startup has configurable attempts1..10 and exponential delay1s..30s capped30s, signal-cancellable waits and bounded connect attempts. Failed pools are closed by existing database.Open. This does not retry business transactions or reset delivery attempts. Provider I/O remains outside transactions; lease fencing/dedup/retry/inbox/SSE checks unchanged. Local Compose forces providers disabled. See docker-local.md for non-root/read-only/capability/mount/port rules.

## CAPTCHA decision

No provider was forced into local development. Existing server validation, atomic five-active quota, distributed admission and private bounded uploads provide initial controls without introducing an external privacy/availability dependency or mandatory real QA requests. This is not a claim that CAPTCHA is unnecessary in every deployment. Bots can rotate IPs/contacts, consume a shared gateway budget or create queued notifications. Owner should assess actual abuse/UX/privacy requirements before selecting an optional server-verified adapter with strict timeout and replay control. Local isolated tests must keep external CAPTCHA disabled. No provider tokens, UI challenge or fake verification was added.

## Dependency decisions

Metadata-only npm audit reports7 moderate affected development-package entries through minimatch advisory GHSA-g3hv-75wg-m249 (affected>=7,<11.0.1; fixed11.0.1). This is not7 independently exploitable application flaws. No production dependency advisory was reported by that response. Compatible TypeScript-ESLint updates require isolated validation; no blind override/major npm update was made.

OSV manifest query covers48 Go modules/toolchain entries. Matches: x/crypto0.58.0 openpgp advisory, quic-go0.59.0 HTTP3 advisory(fixed0.59.1), x/net0.60.0 html parser/renderer advisories(fixed0.62.0), and Go1.26.2 standard-library advisories. App uses bcrypt rather than openpgp and does not start HTTP3, so module matches alone are not reachable-exploit proof. Go minimum/build image moved to1.26.11: patch releases address the reported standard-library family including GO-2026-6859. Node base moved within22 to official22.23.5 security release. Docker registry manifests were queried, not downloaded successfully or scanned. Library-version upgrades and callgraph reachability are unresolved validation tasks; no Critical/High application finding is claimed from version strings alone. No new library or major version was introduced.
