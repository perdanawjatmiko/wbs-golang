# Phase 7 architecture decisions

## Approved changes from the original business baseline

The original blueprint defines flat report types. Phase 7 explicitly approves a maximum two-level parent/child hierarchy, parent-scoped FAQ, Excel bulk operations, and rich email-template editing. This file records those changes without editing the blueprint.

## Hierarchical report types and legacy compatibility

`report_types.parent_id` is an adjacency-list reference. Existing rows are not rewritten and therefore become roots. New report submissions require an active root and an active child belonging to it; the existing `reports.report_type_id` stores the child ID. Historical reports referencing a root remain valid and render with the parent name only. No second parent column is added to reports.

Database trigger and service validation enforce two levels. Roots with existing children cannot be converted into children. FAQ association remains `faqs.report_type_id`; new or edited FAQs may refer to NULL (general) or a root only.

## CRUD and API compatibility

Existing `/api/v1/admin/{users,roles,report-types,faqs,email-templates}` resources remain. Report type JSON adds `parent_id` and `parent_name`; public reporting multipart requests add `report_parent_id` while retaining `report_type_id` as child. Backend validates the pair. Existing report tracking is unchanged.

## Excel identity and preview

Templates include stable UUID identity where available. Without IDs, report types use normalized parent/name scope, admin users use normalized email, and FAQs use normalized question within General/parent scope. Ambiguous duplicate keys are rejected. Reparenting an existing report type or FAQ through an import is rejected; use the explicit CRUD flow. A preview token is random, stored only as a hash, bound to the administrator/module, expires after 15 minutes, and commit is idempotent. Stale updates are detected using a row snapshot hash under the shared master-data advisory lock.

## Temporary credential lifecycle

Passwords are generated with cryptographic randomness and stored only as the existing password hash. Imported accounts are created inactive and must change password within 24 hours; exports and preview responses never contain credentials. The current product has no secure one-time credential handoff or step-up verification. Consequently activation/provisioning is intentionally blocked until an administrator performs an approved password setup through the existing protected account management process. Existing users are not forced into password change by migration.

## Email template editor

Create/edit are dedicated pages rather than dialogs. Tiptap emits HTML; the backend sanitizer and allowlisted renderer remain the final boundary. Preview uses a sandboxed iframe with restrictive CSP and performs no delivery. Supported placeholders remain those registered in the Go notification renderer.
