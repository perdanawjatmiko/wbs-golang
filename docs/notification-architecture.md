# Notification architecture

Phase4 uses the existing modular monolith, pgxpool, cookie authentication, database RBAC and response contract. No added dependency, Redis, network call in a business transaction, or independent Next.js business logic.

## Transaction boundaries and contracts

`reports.Repository.Insert` appends REPORT_CREATED after insert, inside the same contact-quota/counter transaction. `WorkflowTx.Apply` appends REPORT_ON_PROGRESS, REPORT_DONE or REPORT_REJECTED after the report version and history change. Notes-only and rejected same-status requests do not emit notification events. Reopening to Open has a technical realtime event but no invented business email event. Failed validation, stale version, history insertion, attachment publication or outbox insertion rolls back the complete transaction.

The append function materializes external deliveries, durable realtime records and recipient inbox rows in that transaction. This small SQL fanout avoids a second dispatcher race between SSE arrival and inbox creation. No SMTP/Telegram action occurs until a worker claims a committed delivery.

Payload v1: `version`, `report_code`, `report_type`, `status`, `external_note`, `timestamp`. Only immutable notification snapshots are stored; no contact, report detail, attachment, internal note, admin identity or secret. `aggregate_id` references the report. Name is loaded for rendering from the immutable reporter identity on the report. A canonical email recipient is stored only in the private delivery row, necessary for stable routing/deduplication; never returned by monitoring. Public external notes are snapshotted so queued rejection reasons cannot be replaced by a later status.

| Business event | Trigger | Email | Telegram |
| --- | --- | --- | --- |
| REPORT_CREATED | Successful new Open report | EMAIL contacts | Configured group |
| REPORT_ON_PROGRESS | Actual transition to On Progress | EMAIL contacts | None |
| REPORT_DONE | Actual transition to Done | EMAIL contacts | None |
| REPORT_REJECTED | Actual transition to Rejected, mandatory external note | EMAIL contacts | None |

`EMAIL` recipient identity is canonical contact. `TELEGRAM` identity is the single logical `configured-group`; the trusted current worker configuration resolves its destination. Changing the group configuration changes the target for pending alerts. Existing reports are not retroactively notified by migration.

## Ordering and consistency

Existing order remains contact advisory transaction lock -> report/type/counter locks -> report/history writes -> notification append transaction lock (761983224) -> event/fanout -> commit. No notification code obtains a contact lock after the append lock. The short global append lock serializes durable sequence allocation through commit, so an SSE cursor cannot skip an earlier allocated sequence that commits late. It is a simple throughput tradeoff, not a business rule. Existing file publication still occurs before submission commit; existing crash/ambiguous-commit limitations remain.

A separate `notification_events` infrastructure table supports technical `report.created` and `report.status_changed`, including Open reactivation without inventing a supported email event. Its UUID is the stable event ID; monotonic sequence is the SSE cursor. Events/inbox are available immediately after business commit and independent of provider availability or worker uptime.

## Inbox semantics

At event creation, one inbox row is inserted for every active admin whose current role is Super Admin or has report.view. Unique(event_id,admin_id) prevents duplicates. Inactive/unauthorized users are not recipients. New grants receive future notifications; no retrospective inbox backfill. The normal report APIs remain available to newly authorized admins. Every inbox API and every SSE batch revalidates current session/permission; losing permission blocks access to old inbox rows as well. Read operations always constrain admin_id to the current session. No assignment or per-report ownership model is introduced.

## Operational boundaries

Delivery state is durable; workers may scale independently of API instances. Outbox processed_at means all eligible deliveries are terminal (sent/failed), or no external delivery is eligible. Pending disabled channels keep processed_at null. SQL finalization serializes on the outbox row before delivery updates to prevent concurrent completion races. Retention/purge is deliberately manual and not implemented; tables accumulate durable audit/event data. No deletion job or speculative production infrastructure is added.
