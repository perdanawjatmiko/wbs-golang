# Phase5 security audit

Official installed skill provenance: local skill lock identifies cloudflare/security-audit-skill, skillPath skills/security-audit/SKILL.md. Its current instructions and companion rules were inspected. No new audit tooling installed and no remote script executed. Profile standard, entire first-party repository, source-only initial assessment followed by independent validation, source remediation review and coverage critics. No compatible prior ledger existed.

Artifacts are private/gitignored under .security-audit/phase-5/run-1/: metadata, architecture, coverage ledger, candidates/findings JSON, source-review results, advisory responses and derived reports. No credentials, reporter content or real provider responses are collected. Do not commit raw artifacts. Coverage validators check structural consistency, not absence of vulnerabilities.

## Method and execution limits

Map entrypoints/principals/transforms/sinks, follow source controls, separate source facts from deployment assumptions, independently validate each candidate, review patches, run coverage critics and validate structured reports. Covered domains: auth/session/CSRF, database RBAC/master administration, reporting/tracking/quota/concurrency, private files/decoding, admin workflow/history, notification providers/outbox/leases/inbox, SSE/replay/revocation, Next adapters/browser rendering, CLI/bootstrap/migration lifecycle, Docker/config/build contexts and dependency metadata.

The skill requires target-controlled tests/builds to use an OS-enforced sandbox: no external network, isolated loopback only, empty allowlisted environment, read-only target/toolchain, scratch-only writes and CPU/memory/pid/file/disk/time bounds. Docker is available, but images cannot be extracted due to no space left on device. Equivalent host isolation could not be established. No unrestricted host execution of app tests was substituted. Therefore no dynamic exploit, runtime regression or comprehensive security assurance claim is made. External database identity, privileges, pg_hba/firewall/TLS and observed client source are unknown and were not probed.

## Finding dispositions

| ID | Classification | Result |
| --- | --- | --- |
| P5-RATE-001 / reports/public-adapter/shared-peer-rate-budget | Rejected / false positive | Independent reviewer confirmed peer sharing is an explicitly documented aggregate admission policy, not a demonstrated violated per-user security boundary. Retained as operational limitation, not vulnerability. |

No confirmed Critical/High application vulnerability was established by this source pass. That is not proof of their absence, especially with dynamic and external-environment checks blocked. No vulnerability is labelled confirmed without a demonstrated boundary outcome. No separate needs-validation vulnerability record survived independent review; environment and advisory reachability gaps remain in the validation backlog below, not invented findings.

The rejected candidate traced frontend public adapter forward -> no client-IP assertion -> backend untrusted-proxy handling -> IP admission. Proposed impact was unrelated users sharing the budget after invalid attempts. Counterevidence: docs/report-submission.md and docs/phase-2.md already define conservative pre-parser shared-peer admission. No per-user entitlement or stronger isolation invariant was established. A two-client429 reproduction would demonstrate the policy, not prove a new boundary bypass. Existing policy remains explicit after distributed storage.

## Independent validation and re-audit

Initial source hunters used separate auth/workflow, public/files, notification and runtime/browser units. Critics identified missing bootstrap/migration and admin report rendering units, which received fresh source review. A second critic identified changed-source rate/config/worker/header paths after implementation; archived earlier evidence is preserved and current-source reassignment is reviewed separately. Final result and counts are recorded in phase-5.md after report validation. No unresolved unit may be silently reported covered.

No speculative architecture rewrite, arbitrary forwarded-IP trust, production probe, external provider delivery, dangerous fixture processing or availability stress was performed. Generic best-practice improvements were kept distinct from findings: framing policy, bounded public response, secure-cookie configurability and patched toolchain/images.

## Validation backlog

- Docker build/image contents/non-root effective state, private volume persistence, logs and app smoke: blocked by storage.
- Go/Next compile, lint, typecheck, unit/race/security/HTTP tests: blocked by unavailable required execution sandbox.
- PostgreSQL rate SQL/concurrency/expiry/schema regression: additionally needs an explicitly approved disposable test database; no managed database DDL was executed.
- External PostgreSQL source address/access grants/listen/pg_hba/TLS: needs owner-provided nonsecret topology and approval; no automatic configuration edits.
- Go symbol-level advisory reachability and actual image dependency scan: tools/images unavailable; metadata inventory alone is partial.
- Browser visual/header/stream behavior: not executed or claimed.

See security-hardening.md for exact mechanisms/advisory decisions and security-test-results.md for executed evidence versus blocked checks. No load/performance, production validation/deployment, Kubernetes, CI/CD or real SMTP/Telegram testing is in scope.
