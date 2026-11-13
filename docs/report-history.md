# Report histories and public projection

Existing report_histories is the immutable API audit log. Each status transition writes one row in the same transaction as reports.status/updated_at. old_status is the locked previous state, new_status target, admin_id authenticated actor, notes trimmed plain text, created_at server-generated and identical to the new report updated_at. No update/delete history endpoint exists. Phase 3 does not change the schema.

Notes-only endpoint requires one nonempty note and writes old_status=new_status=current status; report status is unchanged and updated_at advances for concurrency protection. Internal-only notes are allowed even on Rejected without repeating a rejection reason because no rejection transition occurs. Every transition *to* Rejected requires a public external reason. Request body cannot choose actor, old status or server timestamp.

History ordering: created_at ascending, UUID id as stable secondary order. Monotonic updated_at prevents same-report timestamp ties during application mutations; imported/fixture timestamp ties remain deterministically sorted by id. Detail history pagination defaults50/max100. Actor name is the current admin display name, not an immutable historical name snapshot; actor ID remains the audit identity.

## Public tracking regression

The same public POST code+canonical-contact verification, rate limiting and no-store contract remain. SQL filters the matched report before projection; no admin-only fields enter the public DTO.

Public timeline contains only new_status/external_note/created_at. It includes status changes and histories with nonblank external_note. Internal-only notes with unchanged status are omitted entirely, so no empty internal-note event appears. Report updated_at still reflects any report mutation, including internal notes, as the existing public summary contract requires.

A virtual Open event at reports.created_at is prepended unless the first retained Open event is at that exact creation instant. An external note posted later while status remains Open is not mistaken for a creation event. No duplicate initial creation event is introduced. Raw note/detail text is escaped by React, never rendered with dangerouslySetInnerHTML.

This public query adjustment is the necessary Phase 3 privacy extension to Phase 2: notes-only records did not previously exist. Public credential model and Next public adapter are unchanged. Internal notes, admin identity, history IDs, contact, storage key and authentication metadata never enter public tracking responses.

## Integrity and limitations

Status, version and history commit together; injected history-insert failure test confirms rollback of all three. Same-status requests create no history. Stale retries are rejected before mutation. New requests with a fresh version can intentionally append identical note text; no semantic idempotency ledger. Direct database operators remain outside API immutability guarantees. Public timeline remains complete/unpaginated under the prior API contract; future high-history-volume work should evaluate bounded pagination without leaking private fields.
