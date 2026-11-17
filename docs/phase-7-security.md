# Phase 7 security notes

- All import/export routes are registered under existing authenticated, CSRF-protected, permission-checked master API groups.
- Backend validation and DB constraints are authoritative; the frontend only improves usability.
- Import file bytes, row count, sheet count, headers, cell size, formulas, IDs, references, and duplicate identities are bounded/validated.
- Preview tokens are high-entropy, stored hashed, administrator-bound, module-bound, time-limited, one-use for mutation, and idempotent after commit.
- Commit re-resolves authorization and serializes against master CRUD; stale row hashes reject changed updates.
- XLSX export has no password hashes, temporary credentials, sessions, or secrets. Export is bounded and marked no-store/nosniff.
- New imported accounts remain inactive. Temporary passwords are cryptographically generated and hashed; plaintext is discarded. Secure delivery is a known blocker, so activation is withheld pending an approved administrator password setup.
- Email body sanitization is enforced by Go; Tiptap/iframe preview is not a security boundary. Preview cannot send mail and is sandboxed.
- Public report selection verifies active parent and child relationship server-side. Legacy reports remain readable.
- Existing session, CSRF, RBAC, rate limiting, quotas, attachments, tracking, notification, and status behavior remain in place.

Security tests use local unit fixtures only. PostgreSQL migration/integration verification and browser accessibility/visual QA require isolated QA infrastructure and remain blocked when unavailable.

The production dependency audit (`npm audit --omit=dev --audit-level=high`) found no advisories. Full npm audit reports seven moderate development-only transitive advisories involving `minimatch` through `typescript-eslint`; a broad toolchain upgrade was not applied. `govulncheck` was not installed, so Go dependency vulnerability scanning remains unverified.
