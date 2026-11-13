"use client";
import { useRef, useState, type FormEvent } from "react";
import { trackReport } from "./client";
import { type TrackedReport, type TimelineEvent, contactError } from "./contracts";
export function PublicTimeline({ events }: {
    events: TimelineEvent[];
}) { return <ol className="mt-5 space-y-5 border-l border-zinc-300 pl-6" aria-label="Timeline laporan">{events.map((event, index) => <li key={`${event.created_at}-${index}`}><p className="font-semibold">{event.new_status}</p><time className="text-sm text-zinc-500" dateTime={event.created_at}>{new Date(event.created_at).toLocaleString("id-ID")}</time>{event.external_note && <p className="mt-2 whitespace-pre-wrap">{event.external_note}</p>}</li>)}</ol>; }
export function TrackingForm() {
    const [code, setCode] = useState("");
    const [contact, setContact] = useState("");
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");
    const [report, setReport] = useState<TrackedReport | null>(null);
    const inFlight = useRef(false);
    async function submit(event: FormEvent<HTMLFormElement>) { event.preventDefault(); if (inFlight.current)
        return; setReport(null); if (!/^WBS-[0-9]{6}-[A-Z0-9]{2}[0-9]{3,19}$/.test(code.trim()) || contactError(contact.includes("@") ? "EMAIL" : "PHONE", contact)) {
        setError("Masukkan kode laporan dan email/nomor internasional yang valid.");
        return;
    } setError(""); setBusy(true); inFlight.current = true; try {
        setReport(await trackReport(code.trim(), contact));
    }
    catch (error) {
        setError(error instanceof Error ? error.message : "Tracking belum tersedia.");
    }
    finally {
        setBusy(false);
        inFlight.current = false;
    } }
    return <div className="space-y-10"><form onSubmit={submit} noValidate className="max-w-xl space-y-5" aria-label="Tracking laporan"><label className="block" htmlFor="report-code">Kode laporan<input id="report-code" className="admin-input font-mono" value={code} required autoComplete="off" maxLength={100} onChange={e => { setCode(e.target.value); setReport(null); }}/></label><label className="block" htmlFor="tracking-contact">Email atau nomor telepon internasional<input id="tracking-contact" className="admin-input" value={contact} required autoComplete="off" maxLength={254} onChange={e => { setContact(e.target.value); setReport(null); }}/></label><p className="text-sm text-zinc-500">Gunakan contact yang sama saat mengirim laporan. Data tracking tidak dimasukkan ke URL atau disimpan otomatis.</p><button className="admin-button" disabled={busy}>{busy ? "Memeriksa…" : "Lihat status"}</button>{error && <p role="alert" className="text-red-700 dark:text-red-400">{error}</p>}</form>
 {report && <section aria-live="polite" className="rounded-2xl border border-zinc-200 p-6 dark:border-zinc-800"><p className="break-all font-mono text-sm text-zinc-500">{report.report_code}</p><h2 className="mt-2 text-2xl font-semibold">Status: {report.status}</h2><dl className="my-6 grid gap-2 sm:grid-cols-[10rem_1fr]"><dt>Nama</dt><dd>{report.name}</dd><dt>Jenis laporan</dt><dd>{report.report_type}</dd><dt>Dibuat</dt><dd>{new Date(report.created_at).toLocaleString("id-ID")}</dd><dt>Diperbarui</dt><dd>{new Date(report.updated_at).toLocaleString("id-ID")}</dd></dl><h3 className="font-semibold">Detail laporan</h3><p className="mt-3 whitespace-pre-wrap break-words">{report.detail}</p><h3 className="mt-8 text-lg font-semibold">Perjalanan laporan</h3><PublicTimeline events={report.timeline}/></section>}
 </div>;
}
