# Durable admin SSE

GET /api/v1/admin/notifications/stream requires a valid active cookie session and report.view before streaming. Native EventSource connects through the finite same-origin Next adapter; the HttpOnly token is forwarded server-side only. Stream responses are no-cache, no-store, private, text/event-stream and X-Accel-Buffering:no. Backend per-write5s deadlines allow SSE without removing ordinary HTTP server write timeouts. Default heartbeat15s, poll2s, max100 simultaneous connections per API process.

Each poll revalidates session expiration, revocation, admin activity and current database permissions before reading event data. Failure sends `access.revoked` without report data, then closes. Shutdown/disconnect cancels polling and releases the connection slot. No dedicated DB connection is held for the stream lifetime. Each query is DB_QUERY_TIMEOUT bounded. Multi-instance operation reads the same durable PostgreSQL records, not an in-memory subscription hub. LISTEN/NOTIFY is unnecessary for this simple bounded polling implementation.

Example:

```text
id: 42
event: report.status_changed
data: {"event_id":"UUID","report_id":"UUID","report_code":"WBS-141126-X9001","status":"Done","timestamp":"2026-11-14T08:00:00Z"}

```

Event UUID identifies the event; numeric sequence is Last-Event-ID. Native reconnect preserves this cursor. Fresh connections start at the committed current cursor and refresh the inbox; reconnect replays only rows assigned to that authenticated admin. Replay default100. If backlog exceeds the bound, server sends inbox.reset at the current cursor and UI refreshes the durable inbox. The bell shows latest20 inbox entries; all read APIs paginate up to100. No contact, note, payload, provider credentials or admin identity appears in SSE.

The bell deduplicates the last200 event UUIDs, refreshes inbox/unread after events, and reports connection errors. Native retry3s; after ten errors the client pauses30s before a fresh stream/inbox refresh. Session/permission errors clear private client state and stop reconnecting. Healthy SSE does not trigger interval-based client polling. All effects close EventSource/timers on unmount. Realtime is an invalidation signal, not a guaranteed complete client-side event archive; durable inbox remains authoritative.

Next adapter connects with10s header timeout, streams incrementally with backpressure, bounds each upstream chunk64KiB, forwards only session/validated cursor, aborts upstream on browser disconnect, and never buffers a whole stream. A large coalesced upstream chunk can cause reconnect/refresh rather than unbounded buffering. Deployment proxies must preserve streaming and not cache it; deployment itself remains out of scope.
