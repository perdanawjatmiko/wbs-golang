"use client";
import Link from 'next/link';
import { useCallback, useEffect, useRef, useState } from 'react';
import { adminRequest, AdminApiError } from '@/features/admin/client';
import { acceptEvent, parseLiveEvent, type Inbox } from './contracts';
export function NotificationBell({ csrf }: {
    csrf: string;
}) {
    const [open, setOpen] = useState(false), [inbox, setInbox] = useState<Inbox | null>(null), [count, setCount] = useState(0), [state, setState] = useState('Menghubungkan…'), [error, setError] = useState(''), [busy, setBusy] = useState(false);
    const denied = useRef(false), mounted = useRef(false);
    const refresh = useCallback(async () => { try {
        const [list, unread] = await Promise.all([adminRequest<Inbox>('admin/notifications?per_page=20'), adminRequest<{
                unread_count: number;
            }>('admin/notifications/unread-count')]);
        if (mounted.current && !denied.current) {
            setInbox(list);
            setCount(unread.unread_count);
            setError('');
        }
    }
    catch (e) {
        if (!mounted.current)
            return;
        if (e instanceof AdminApiError && (e.status === 401 || e.status === 403)) {
            denied.current = true;
            setInbox(null);
            setCount(0);
            setState('Sesi berakhir atau izin dicabut. Muat ulang untuk login.');
        }
        setError(e instanceof Error ? e.message : 'Notifikasi tidak tersedia.');
    } }, []);
    useEffect(() => {
        mounted.current = true;
        denied.current = false;
        let source: EventSource | null = null, timer: ReturnType<typeof setTimeout> | undefined, stopped = false, failures = 0;
        const seen = new Set<string>();
        const connect = () => {
            if (stopped || denied.current)
                return;
            source = new EventSource('/api/backend/admin/notifications/stream');
            source.onopen = () => { failures = 0; setState('Terhubung'); void refresh(); };
            const onEvent = (event: MessageEvent) => { const parsed = parseLiveEvent(event.data); if (parsed && acceptEvent(seen, parsed.event_id)) {
                void refresh();
                window.dispatchEvent(new Event('wbs:reports-changed'));
            } };
            source.addEventListener('report.created', onEvent);
            source.addEventListener('report.status_changed', onEvent);
            source.addEventListener('inbox.reset', () => void refresh());
            source.addEventListener('access.revoked', () => { denied.current = true; source?.close(); setInbox(null); setCount(0); setState('Sesi berakhir atau izin dicabut. Muat ulang untuk login.'); });
            // Native EventSource preserves Last-Event-ID across automatic reconnects.
            source.onerror = () => { if (stopped)
                return; setState('Koneksi terputus; menghubungkan ulang…'); void refresh().then(() => { if (denied.current)
                source?.close(); }); failures++; if (failures > 10) {
                source?.close();
                timer = setTimeout(() => { failures = 0; connect(); }, 30000);
            } };
        };
        connect();
        return () => { stopped = true; mounted.current = false; source?.close(); if (timer)
            clearTimeout(timer); };
    }, [refresh]);
    async function mark(id?: string) { setBusy(true); try {
        await adminRequest(`admin/notifications/${id ? `${id}/read` : 'read-all'}`, { method: 'POST', headers: { 'X-CSRF-Token': csrf }, body: '{}' });
        await refresh();
    }
    catch (e) {
        setError(e instanceof Error ? e.message : 'Perubahan gagal.');
    }
    finally {
        setBusy(false);
    } }
    return <div className="relative"><button className="admin-button-secondary" aria-label={`Notifikasi, ${count} belum dibaca`} aria-expanded={open} aria-controls="notification-panel" onClick={() => setOpen(!open)}>Notifikasi <span aria-live="polite">{count > 99 ? '99+' : count}</span></button>{open && <section id="notification-panel" aria-label="Pusat notifikasi" className="absolute right-0 z-20 mt-2 max-h-[70vh] w-[min(90vw,26rem)] overflow-y-auto rounded-xl border border-zinc-300 bg-white p-4 shadow-xl dark:bg-zinc-950"><h2 className="font-semibold">Notifikasi laporan</h2><p role="status" className="my-2 text-sm">{state}</p>{error && <p role="alert">{error}</p>}{!inbox && !error && <p>Memuat…</p>}<button className="my-3 underline" disabled={busy || state.startsWith('Sesi') || count === 0} onClick={() => void mark()}>Tandai semua dibaca</button>{inbox?.items.length === 0 && <p>Belum ada notifikasi.</p>}<ul className="space-y-3">{inbox?.items.map(n => <li key={n.id} className={`rounded border p-3 ${n.read_at ? 'opacity-70' : 'border-blue-400'}`}><Link href={`/admin/reports/${n.report_id}`} className="font-mono underline" onClick={() => { setOpen(false); if (!n.read_at)
        void mark(n.id); }}>{n.title}</Link><p>{n.message}</p><time className="text-xs" dateTime={n.created_at}>{new Date(n.created_at).toLocaleString('id-ID')}</time>{!n.read_at && <button disabled={busy} className="ml-3 text-sm underline" onClick={() => void mark(n.id)}>Tandai dibaca</button>}</li>)}</ul>{inbox && inbox.pagination.total > 20 && <p className="mt-3 text-sm">20 notifikasi terbaru. Daftar laporan menyediakan seluruh tiket.</p>}</section>}</div>;
}
