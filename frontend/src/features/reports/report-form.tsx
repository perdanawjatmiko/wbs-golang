"use client";
import Link from "next/link";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { reportTypes, uploadPolicy, submitReport, PublicApiError } from "./client";
import { validateReport, switchContact, type ReportInput, type ReportType, type UploadPolicy, type FieldErrors, type CreatedReport } from "./contracts";
import { InteractiveFAQ } from "./faq";
const empty: ReportInput = { name: "", contact: "", contact_type: "EMAIL", report_type_id: "", detail: "" };
export function ReportForm() {
    const [input, setInput] = useState<ReportInput>(empty);
    const [file, setFile] = useState<File | null>(null);
    const [catalog, setCatalog] = useState<{
        types: ReportType[];
        policy: UploadPolicy;
    } | null>(null);
    const [catalogError, setCatalogError] = useState("");
    const [retry, setRetry] = useState(0);
    const [errors, setErrors] = useState<FieldErrors>({});
    const [error, setError] = useState("");
    const [busy, setBusy] = useState(false);
    const [uncertain, setUncertain] = useState(false);
    const [created, setCreated] = useState<CreatedReport | null>(null);
    const [copyMessage, setCopyMessage] = useState("");
    const inFlight = useRef(false);
    useEffect(() => { let canceled = false; Promise.all([reportTypes(), uploadPolicy()]).then(([types, policy]) => { if (!canceled) {
        setCatalog({ types, policy });
        setCatalogError("");
    } }, () => { if (!canceled)
        setCatalogError("Jenis laporan atau batas upload belum dapat dimuat."); }); return () => { canceled = true; }; }, [retry]);
    async function submit(event: FormEvent<HTMLFormElement>) { event.preventDefault(); if (!catalog || inFlight.current || uncertain)
        return; const invalid = validateReport(input, file, catalog.policy, catalog.types); setErrors(invalid); if (Object.keys(invalid).length)
        return; inFlight.current = true; setBusy(true); setError(""); try {
        const result = await submitReport(input, file);
        setCreated(result);
        setInput(empty);
        setFile(null);
    }
    catch (error) {
        setError(error instanceof Error ? error.message : "Laporan belum dapat dikirim.");
        setUncertain(error instanceof PublicApiError && error.uncertain);
    }
    finally {
        setBusy(false);
        inFlight.current = false;
    } }
    function change<K extends keyof ReportInput>(key: K, value: ReportInput[K]) { setInput(prev => ({ ...prev, [key]: value })); setErrors(prev => ({ ...prev, [key]: undefined })); }
    const fieldError = (name: keyof FieldErrors) => errors[name] ? <p id={`error-${name}`} className="mt-2 text-sm text-red-700 dark:text-red-400">{errors[name]}</p> : null;
    if (created)
        return <section aria-live="polite" className="space-y-5 rounded-2xl border border-emerald-300 p-6"><h2 className="text-2xl font-semibold">Laporan berhasil diterima</h2><p>Simpan kode ini dan gunakan contact yang sama untuk tracking.</p><p className="break-all rounded-lg bg-zinc-100 p-4 font-mono text-xl text-zinc-950">{created.report_code}</p><p>Status: {created.status}</p><button className="admin-button-secondary" onClick={async () => { try {
            await navigator.clipboard.writeText(created.report_code);
            setCopyMessage("Kode disalin.");
        }
        catch {
            setCopyMessage("Penyalinan gagal. Pilih kode dan salin secara manual.");
        } }}>Salin kode laporan</button><p role="status">{copyMessage}</p><p>{created.tracking_instructions}</p><Link className="admin-button" href="/tracking">Tracking laporan</Link><p className="text-sm text-zinc-500">Kode tidak disimpan otomatis di browser. Memuat ulang halaman ini tidak mengirim ulang laporan.</p></section>;
    if (catalogError)
        return <p role="alert">{catalogError} <button className="underline" onClick={() => setRetry(n => n + 1)}>Coba lagi</button></p>;
    if (!catalog)
        return <p role="status">Memuat formulir…</p>;
    return <div className="grid gap-10 lg:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]"><form onSubmit={submit} noValidate className="space-y-6" aria-label="Form laporan publik">
 <fieldset disabled={busy || uncertain} className="space-y-6"><legend className="sr-only">Detail laporan</legend>
 <label className="block" htmlFor="name">Nama<input id="name" className="admin-input" autoComplete="name" required maxLength={150} value={input.name} aria-invalid={!!errors.name} aria-describedby={errors.name ? "error-name" : undefined} onChange={e => change("name", e.target.value)}/>{fieldError("name")}</label>
 <label className="block" htmlFor="contact-type">Jenis contact<select id="contact-type" className="admin-input" value={input.contact_type} onChange={e => { setInput(prev => switchContact(prev, e.target.value as "EMAIL" | "PHONE")); setErrors(prev => ({ ...prev, contact: undefined })); }}><option value="EMAIL">Email</option><option value="PHONE">Telepon internasional</option></select></label>
 <label className="block" htmlFor="contact">{input.contact_type === "EMAIL" ? "Email" : "Nomor telepon (+kode negara)"}<input id="contact" className="admin-input" type={input.contact_type === "EMAIL" ? "email" : "tel"} autoComplete={input.contact_type === "EMAIL" ? "email" : "tel"} required maxLength={254} value={input.contact} aria-invalid={!!errors.contact} aria-describedby={errors.contact ? "contact-help error-contact" : "contact-help"} onChange={e => change("contact", e.target.value)}/><span id="contact-help" className="mt-2 block text-sm text-zinc-500">Gunakan contact ini saat tracking. Telepon harus memakai awalan + dan kode negara.</span>{fieldError("contact")}</label>
 <label className="block" htmlFor="report-type">Jenis laporan<select id="report-type" className="admin-input" required value={input.report_type_id} aria-invalid={!!errors.report_type_id} aria-describedby={errors.report_type_id ? "error-report_type_id" : undefined} onChange={e => change("report_type_id", e.target.value)}><option value="">Pilih jenis laporan</option>{catalog.types.map(type => <option key={type.id} value={type.id}>{type.name}</option>)}</select>{fieldError("report_type_id")}</label>{catalog.types.length === 0 && <p role="status">Belum ada jenis laporan aktif. Hubungi pengelola.</p>}
 <label className="block" htmlFor="detail">Detail laporan<textarea id="detail" className="admin-input" rows={9} required maxLength={20000} value={input.detail} aria-invalid={!!errors.detail} aria-describedby={errors.detail ? "error-detail" : undefined} onChange={e => change("detail", e.target.value)}/>{fieldError("detail")}</label>
 {catalog.policy.max_files > 0 && <label className="block" htmlFor="attachment">Attachment (opsional, satu file)<input id="attachment" className="admin-input" type="file" accept={catalog.policy.allowed_types.join(",")} aria-invalid={!!errors.attachment} aria-describedby={errors.attachment ? "attachment-help error-attachment" : "attachment-help"} onChange={e => { setFile(e.target.files?.[0] ?? null); setErrors(prev => ({ ...prev, attachment: undefined })); }}/><span id="attachment-help" className="mt-2 block text-sm text-zinc-500">Maksimum {(catalog.policy.max_file_bytes / 1048576).toFixed(1)} MiB · {catalog.policy.allowed_types.join(", ")}. File diperiksa kembali oleh server.</span>{fieldError("attachment")}</label>}
 <button className="admin-button" disabled={busy || catalog.types.length === 0}>{busy ? "Mengirim laporan…" : "Kirim laporan"}</button></fieldset>
 {error && <p role="alert" className="rounded-xl border border-amber-300 p-4">{error}</p>}{uncertain && <Link className="underline" href="/tracking">Buka tracking</Link>}
 </form><aside><InteractiveFAQ reportType={input.report_type_id}/></aside></div>;
}
