"use client";
import { useRef, useState, type FormEvent } from "react";
import { trackReport } from "./client";
import { type TrackedReport, type TimelineEvent, contactError } from "./contracts";
import { StatusBadge } from "@/components/ui/status-badge";
export function PublicTimeline({ events }: {
    events: TimelineEvent[];
}) { return <ol className="timeline" aria-label="Timeline laporan">{events.map((event, index) => <li key={`${event.created_at}-${index}`}><StatusBadge status={event.new_status}/><time className="mt-2 block text-xs text-slate-500" dateTime={event.created_at}>{new Date(event.created_at).toLocaleString("id-ID")}</time>{event.external_note && <p>{event.external_note}</p>}</li>)}</ol>; }
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
    return <div className="tracking-layout"><form onSubmit={submit} noValidate className="public-panel tracking-form" aria-label="Tracking laporan"><h2>Masukkan informasi laporan</h2><div className="field-stack"><label className="block" htmlFor="report-code">Kode laporan<input id="report-code" className="admin-input font-mono" value={code} required autoComplete="off" maxLength={100} onChange={e => { setCode(e.target.value); setReport(null); }}/></label><label className="block" htmlFor="tracking-contact">Email atau nomor telepon internasional<input id="tracking-contact" className="admin-input" value={contact} required autoComplete="off" maxLength={254} onChange={e => { setContact(e.target.value); setReport(null); }}/></label><p className="form-hint">Gunakan contact yang sama saat mengirim laporan. Data tracking tidak dimasukkan ke URL atau disimpan otomatis.</p><button className="admin-button" disabled={busy}>{busy ? "Memeriksa…" : "Lihat status"}</button>{error && <p role="alert" className="form-error">{error}</p>}</div></form>
 {report && <section aria-live="polite" className="tracking-result"><div className="tracking-result-head"><div><p className="eyebrow-small">Kode laporan</p><p className="tracking-code">{report.report_code}</p></div><StatusBadge status={report.status}/></div><h2 className="tracking-status">Status laporan</h2><dl className="tracking-meta"><div><dt>Nama</dt><dd>{report.name}</dd></div><div><dt>Jenis laporan</dt><dd>{report.report_type}</dd></div><div><dt>Dibuat</dt><dd>{new Date(report.created_at).toLocaleString("id-ID")}</dd></div><div><dt>Diperbarui</dt><dd>{new Date(report.updated_at).toLocaleString("id-ID")}</dd></div></dl><div className="tracking-detail"><h3>Detail laporan</h3><p>{report.detail}</p></div><h3 className="mt-7 font-semibold">Perjalanan laporan</h3><PublicTimeline events={report.timeline}/></section>}
 </div>;
}
