# ADR 0001: Phase 7 master data enhancements

- Status: Accepted for Phase 7 implementation
- Date: 2026-11-16

## Context

The business blueprint defines flat report types and its existing CRUD schema. Phase 7 explicitly authorizes two-level hierarchy, parent-scoped FAQ, XLSX bulk import/export, and rich email-template editing. Existing records and workflows must remain intact.

## Decisions

1. Use an adjacency-list `report_types.parent_id` with a strict maximum depth of two. Existing rows stay unchanged and serve as roots. New reports store the selected child in the current report type FK; parent is derived. Legacy report references to roots remain readable.
2. FAQs remain attached to NULL/general or a root category only. The backend and database reject child-linked FAQs.
3. XLSX natural upsert keys are normalized parent/name for report types, email for users, and normalized question within general/parent scope for FAQs. Stable UUIDs take precedence; imports cannot silently reparent records. Preview is immutable, actor-bound, expiring, stale-checked, and commit is atomic/idempotent.
4. Import-created admin accounts are inactive, receive a cryptographically generated hashed temporary credential, and require password change within 24 hours. No password is exposed. Since no safe one-time retrieval or step-up verification exists, secure handoff remains blocked and activation requires explicit administrator password setup.
5. Preserve existing visual tokens and CRUD behavior; use Dialog for master CRUD and dedicated Tiptap pages for email-template create/edit.

## Consequences

Migration 000006 is additive. Its down migration refuses while children exist because flattening would be lossy. PostgreSQL migration verification must occur on a disposable QA database. Existing authentication and notification renderer remain authoritative.
