export type NotificationItem = {
    id: string;
    event_id: string;
    report_id: string;
    title: string;
    message: string;
    read_at: string | null;
    created_at: string;
};
export type Page = {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
};
export type Inbox = {
    items: NotificationItem[];
    pagination: Page;
};
export type Delivery = {
    id: string;
    event_key: string;
    channel: 'EMAIL' | 'TELEGRAM';
    status: 'pending' | 'processing' | 'sent' | 'retrying' | 'failed';
    attempts: number;
    max_attempts: number;
    next_attempt_at: string;
    last_error: string | null;
    created_at: string;
    sent_at: string | null;
    manual_retry_used: boolean;
};
export type Deliveries = {
    items: Delivery[];
    pagination: Page;
    metrics: Record<string, number>;
};
export type LiveEvent = {
    event_id: string;
    report_id: string;
    report_code: string;
    status: string;
    timestamp: string;
};
export function parseLiveEvent(raw: string): LiveEvent | null { try {
    const e = JSON.parse(raw);
    if (!e || !['event_id', 'report_id', 'report_code', 'status', 'timestamp'].every(k => typeof e[k] === 'string') || !['Open', 'On Progress', 'Done', 'Rejected'].includes(e.status))
        return null;
    return e as LiveEvent;
}
catch {
    return null;
} }
// Bounded deduplication only; never stores credentials or report contact.
export function acceptEvent(seen: Set<string>, id: string): boolean { if (seen.has(id))
    return false; seen.add(id); if (seen.size > 200) {
    const oldest = seen.values().next().value;
    if (oldest)
        seen.delete(oldest);
} return true; }
export function canRetry(d: Delivery) { return d.status === 'failed' && d.attempts < d.max_attempts && !d.manual_retry_used; }
