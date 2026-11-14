"use client";
import { useCallback, useEffect, useRef, useState } from 'react';
import { adminRequest, AdminApiError } from '@/features/admin/client';
import { AccessError, PageControls, ReportSkeleton } from '@/features/admin-reports/shared';
import { canRetry, type Deliveries, type Delivery } from './contracts';
export function DeliveryTable({ items, onRetry, busy }: {
    items: Delivery[];
    onRetry: (d: Delivery) => void;
    busy: boolean;
}) { return <div className="overflow-x-auto"><table className="w-full text-left text-sm"><caption className="sr-only">Pengiriman notifikasi</caption><thead><tr>{['Event / ID', 'Channel', 'Status', 'Percobaan', 'Retry berikutnya', 'Ringkasan error', 'Tindakan'].map(t => <th className="p-3" key={t} scope="col">{t}</th>)}</tr></thead><tbody>{items.map(d => <tr key={d.id} className="border-t"><td className="p-3">{d.event_key}<p className="font-mono text-xs">{d.id}</p><p>{new Date(d.created_at).toLocaleString('id-ID')}</p></td><td className="p-3">{d.channel}</td><td className="p-3">{d.status}</td><td className="p-3">{d.attempts}/{d.max_attempts}</td><td className="p-3">{['pending', 'retrying'].includes(d.status) ? new Date(d.next_attempt_at).toLocaleString('id-ID') : '—'}</td><td className="p-3">{d.last_error ?? '—'}</td><td className="p-3">{canRetry(d) && <button className="underline" disabled={busy} onClick={() => onRetry(d)}>Ulangi</button>}</td></tr>)}</tbody></table>{items.length === 0 && <p className="p-6">Tidak ada delivery yang cocok.</p>}</div>; }
export function DeliveryMonitoring({ csrf }: {
    csrf: string;
}) {
    const dialog=useRef<HTMLDialogElement>(null);
    const [data, setData] = useState<Deliveries | null>(null), [page, setPage] = useState(1), [channel, setChannel] = useState(''), [status, setStatus] = useState(''), [error, setError] = useState(''), [http, setHttp] = useState(0), [loading, setLoading] = useState(true), [busy, setBusy] = useState(false), [confirm, setConfirm] = useState<Delivery | null>(null), [feedback, setFeedback] = useState('');
    const load = useCallback(async () => { setLoading(true); setError(''); try {
        const q = new URLSearchParams({ page: String(page), per_page: '20', channel, status });
        setData(await adminRequest<Deliveries>(`admin/notification-deliveries?${q}`));
    }
    catch (e) {
        setData(null);
        setHttp(e instanceof AdminApiError ? e.status : 0);
        setError(e instanceof Error ? e.message : 'Pengiriman tidak tersedia.');
    }
    finally {
        setLoading(false);
    } }, [page, channel, status]);
    useEffect(() => { const timer = setTimeout(() => void load(), 0); return () => clearTimeout(timer); }, [load]);
    useEffect(()=>{if(confirm)dialog.current?.showModal();},[confirm]);
    async function retry() { if (!confirm)
        return; setBusy(true); try {
        await adminRequest(`admin/notification-deliveries/${confirm.id}/retry`, { method: 'POST', headers: { 'X-CSRF-Token': csrf }, body: '{}' });
        setConfirm(null);
        setFeedback('Delivery dijadwalkan ulang.');
        await load();
    }
    catch (e) {
        setFeedback(e instanceof Error ? e.message : 'Retry gagal. Muat ulang sebelum mencoba lagi.');
    }
    finally {
        setBusy(false);
    } }
    return <section className="space-y-6"><h1 className="text-2xl font-bold">Pengiriman notifikasi</h1><p className="text-sm">Provider nonaktif menyisakan delivery pending. Retry manual dibatasi satu kali dan tetap mengikuti batas percobaan.</p><div className="flex flex-wrap gap-4"><label>Channel<select className="admin-input" value={channel} onChange={e => { setPage(1); setChannel(e.target.value); }}><option value="">Semua</option>{['EMAIL', 'TELEGRAM'].map(v => <option key={v}>{v}</option>)}</select></label><label>Status<select className="admin-input" value={status} onChange={e => { setPage(1); setStatus(e.target.value); }}><option value="">Semua</option>{['pending', 'processing', 'sent', 'retrying', 'failed'].map(v => <option key={v}>{v}</option>)}</select></label><button className="admin-button-secondary" disabled={loading} onClick={() => void load()}>Muat ulang</button></div>{feedback && <p role="status">{feedback}</p>}{loading ? <ReportSkeleton /> : error ? <AccessError message={error} status={http} retry={() => void load()}/> : data && <><dl className="flex flex-wrap gap-5">{Object.entries(data.metrics).map(([k, v]) => <div key={k}><dt>{k}</dt><dd className="text-xl font-semibold">{v}</dd></div>)}</dl><DeliveryTable items={data.items} onRetry={setConfirm} busy={busy}/><PageControls meta={data.pagination} onPage={setPage}/></>}{confirm && <dialog ref={dialog} aria-labelledby="retry-title" onCancel={e=>{e.preventDefault();if(!busy)setConfirm(null);}} className="fixed inset-0 m-auto max-w-lg rounded-xl bg-white p-5 text-zinc-900 backdrop:bg-black/60 dark:bg-zinc-900 dark:text-zinc-100"><div className="max-w-md rounded-xl bg-white p-6 dark:bg-zinc-900"><h2 id="retry-title" className="font-bold">Ulangi delivery?</h2><p className="my-4">Provider mungkin telah menerima pesan sebelum timeout. Retry berisiko mengirim duplikat.</p><div className="flex gap-3"><button autoFocus className="admin-button-secondary" disabled={busy} onClick={() => setConfirm(null)}>Batal</button><button className="admin-button" disabled={busy} onClick={() => void retry()}>{busy ? 'Menjadwalkan…' : 'Konfirmasi retry'}</button></div></div></dialog>}</section>;
}
