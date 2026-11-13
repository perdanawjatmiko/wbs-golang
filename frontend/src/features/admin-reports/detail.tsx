"use client";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { adminRequest, AdminApiError } from "@/features/admin/client";
import { AccessError, PageControls, ReportSkeleton } from "./shared";
import { statuses, validateStatus, validateNotes, workflowErrorMessage, type ReportDetail, type ReportHistory, type StatusInput, type NoteInput } from "./contracts";
import type { ReportStatus } from "@/features/reports/contracts";
export function AdminHistoryTimeline({ histories }: {
    histories: ReportHistory[];
}) { return <ol aria-label="Riwayat administrasi" className="space-y-6 border-l border-zinc-300 pl-5">{histories.map(h => <li key={h.id} className="space-y-3"><p className="font-semibold">{h.old_status === h.new_status ? `Catatan · ${h.new_status}` : `${h.old_status} → ${h.new_status}`}</p><p className="text-sm text-zinc-500">{new Date(h.created_at).toLocaleString("id-ID")} · {h.admin.name}</p>{h.internal_note && <section className="rounded-lg border-l-4 border-amber-500 bg-amber-50 p-4 text-zinc-900"><h4 className="font-semibold">Internal · hanya administrator</h4><p className="mt-2 whitespace-pre-wrap break-words">{h.internal_note}</p></section>}{h.external_note && <section className="rounded-lg border-l-4 border-blue-500 bg-blue-50 p-4 text-zinc-900"><h4 className="font-semibold">Eksternal · terlihat pelapor</h4><p className="mt-2 whitespace-pre-wrap break-words">{h.external_note}</p></section>}</li>)}</ol>; }
function WorkflowForm({ report, csrf, notes, onDone, onConflict }: {
    report: ReportDetail;
    csrf: string;
    notes: boolean;
    onDone: () => void;
    onConflict: (message: string) => void;
}) {
    const [status, setStatus] = useState<ReportStatus>(report.status);
    const [internal, setInternal] = useState("");
    const [external, setExternal] = useState("");
    const [error, setError] = useState("");
    const [busy, setBusy] = useState(false);
    const gate = useRef(false);
    async function submit(e: FormEvent) { e.preventDefault(); if (gate.current)
        return; const input: StatusInput = { status, internal_note: internal.trim(), external_note: external.trim(), expected_updated_at: report.updated_at }; const message = notes ? validateNotes(input) : validateStatus(input, report.status); if (message) {
        setError(message);
        return;
    } gate.current = true; setBusy(true); setError(""); try {
        const payload: StatusInput | NoteInput = notes ? { internal_note: input.internal_note, external_note: input.external_note, expected_updated_at: input.expected_updated_at } : input;
        await adminRequest(`admin/reports/${report.id}/${notes ? "notes" : "status"}`, { method: notes ? "POST" : "PATCH", headers: { "X-CSRF-Token": csrf }, body: JSON.stringify(payload) });
        onDone();
    }
    catch (err) {
        const message = workflowErrorMessage(err instanceof AdminApiError?err.code:undefined,err instanceof Error ? err.message : "Perubahan belum dapat dipastikan.");
        if (err instanceof AdminApiError && (err.status === 409 || err.status === 0 || err.status >= 500 || err.status === 401 || err.status === 403)) {
            onConflict(message + " Data dimuat ulang; tinjau sebelum mengirim kembali.");
        }
        else
            setError(message);
    }
    finally {
        gate.current = false;
        setBusy(false);
    } }
    return <form onSubmit={submit} className="space-y-4 rounded-xl border border-zinc-200 p-5 dark:border-zinc-800"><h3 className="text-xl font-semibold">{notes ? "Tambah catatan" : "Ubah status"}</h3><fieldset disabled={busy} className="space-y-4"><legend className="sr-only">{notes ? "Catatan" : "Perubahan status"}</legend>{!notes && <label className="block">Status tujuan<select className="admin-input" value={status} onChange={e => setStatus(e.target.value as ReportStatus)}>{statuses.map(s => <option key={s}>{s}</option>)}</select></label>}<label className="block">Catatan internal<textarea className="admin-input" maxLength={10000} rows={4} value={internal} onChange={e => setInternal(e.target.value)}/><span className="text-sm text-zinc-500">Hanya administrator. Maksimum 10.000 karakter.</span></label><label className="block">Catatan eksternal<textarea className="admin-input" maxLength={10000} rows={4} value={external} required={!notes && status === "Rejected"} onChange={e => setExternal(e.target.value)}/><span className="text-sm text-zinc-500">Terlihat pelapor melalui tracking.{!notes && status === "Rejected" ? " Wajib menjelaskan alasan penolakan." : ""}</span></label><button className="admin-button" disabled={busy}>{busy ? "Menyimpan…" : notes ? "Simpan catatan" : "Simpan status"}</button></fieldset>{error && <p role="alert" className="text-red-700 dark:text-red-400">{error}</p>}</form>;
}
export function ReportSummary({ report }: {
    report: ReportDetail;
}) { return <section className="space-y-5 rounded-xl border border-zinc-200 p-6 dark:border-zinc-800"><h2 className="text-xl font-semibold">Detail laporan</h2><dl className="grid gap-3 sm:grid-cols-[10rem_1fr]">{[["Kode", report.code], ["Pelapor", report.reporter_name], ["Contact", report.contact], ["Jenis contact", report.contact_type], ["Jenis laporan", report.report_type], ["Status", report.status], ["Dibuat", new Date(report.created_at).toLocaleString("id-ID")], ["Diperbarui", new Date(report.updated_at).toLocaleString("id-ID")]].map(([label, value]) => <div className="contents" key={label}><dt className="text-zinc-500">{label}</dt><dd className="break-all">{value}</dd></div>)}</dl><p className="whitespace-pre-wrap break-words">{report.detail}</p>{report.attachment.available ? <a className="admin-button-secondary" href={`/api/backend/admin/reports/${report.id}/attachment`}>Unduh attachment{report.attachment.size ? ` (${Math.ceil(report.attachment.size / 1024)} KiB)` : ""}</a> : <p className="text-sm text-zinc-500">Attachment tidak tersedia.</p>}</section>; }
export function AdminReportDetail({ id, csrf, canUpdate }: {
    id: string;
    csrf: string;
    canUpdate: boolean;
}) {
    const [report, setReport] = useState<ReportDetail | null>(null);
    const [error, setError] = useState<{
        message: string;
        status: number;
    } | null>(null);
    const [revision, setRevision] = useState(0);
    const [historyPage, setHistoryPage] = useState(1);
    const [loading, setLoading] = useState(true);
    const [notice, setNotice] = useState("");
    useEffect(() => { let ignore = false; adminRequest<ReportDetail>(`admin/reports/${id}?history_page=${historyPage}&history_per_page=50`).then(data => { if (!ignore) {
        setReport(data);
        setError(null);
        setLoading(false);
    } }, err => { if (!ignore) {
        setReport(null);
        setError({ message: err.message, status: err instanceof AdminApiError ? err.status : 0 });
        setLoading(false);
    } }); return () => { ignore = true; }; }, [id, revision, historyPage]);
    function refresh(message: string) { setNotice(message); setLoading(true); setRevision(n => n + 1); }
    if (error)
        return <AccessError {...error} retry={() => refresh("")}/>;
    return <div className="space-y-7"><h1 className="text-3xl font-semibold">Pengelolaan laporan</h1>{notice && <p role="status" className="rounded-xl border border-blue-300 p-4">{notice}</p>}{loading || !report ? <ReportSkeleton /> : <><ReportSummary report={report}/>{canUpdate && <div key={report.updated_at + revision} className="grid gap-6 lg:grid-cols-2"><WorkflowForm report={report} csrf={csrf} notes={false} onDone={() => refresh("Status berhasil disimpan.")} onConflict={refresh}/><WorkflowForm report={report} csrf={csrf} notes onDone={() => refresh("Catatan berhasil disimpan.")} onConflict={refresh}/></div>}<section className="space-y-5"><h2 className="text-xl font-semibold">Riwayat laporan</h2>{report.histories.length === 0 ? <p>Belum ada history perubahan atau catatan. Laporan dibuat dengan status Open.</p> : <AdminHistoryTimeline histories={report.histories}/>}<PageControls meta={report.history_pagination} onPage={page => { setLoading(true); setHistoryPage(page); }}/></section></>}</div>;
}
