"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";

type ImportRow = { line: number; data: Record<string, string>; errors: string[]; action?: string };
type Preview = { preview_token: string; expires_in_seconds: number; total_rows: number; valid_rows: number; invalid_rows: number; insert_candidates: number; update_candidates: number; rows: ImportRow[]; credential_provisioning?: string };

export function ImportExport({ module, csrf }: { module: "users" | "report-types" | "faqs"; csrf: string }) {
  const [open, setOpen] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState("");

  async function upload() {
    if (!file) { setError("Pilih file .xlsx terlebih dahulu."); return; }
    if (!file.name.toLowerCase().endsWith(".xlsx") || file.size === 0 || file.size > 5 * 1024 * 1024) { setError("File harus .xlsx, tidak kosong, dan maksimum 5 MiB."); return; }
    setLoading(true); setError(""); setPreview(null); setResult("");
    const body = new FormData(); body.set("file", file);
    try {
      const response = await fetch(`/api/backend/admin/${module}/import/preview`, { method: "POST", credentials: "same-origin", cache: "no-store", signal: AbortSignal.timeout(30000), headers: { "X-CSRF-Token": csrf }, body });
      const envelope = await response.json();
      if (!response.ok || envelope.success !== true) throw new Error(envelope?.error?.message || "Preview import gagal.");
      setPreview(envelope.data as Preview);
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Preview import gagal."); }
    finally { setLoading(false); }
  }

  async function commit() {
    if (!preview || preview.invalid_rows > 0) return;
    setLoading(true); setError("");
    try {
      const response = await fetch(`/api/backend/admin/${module}/import/commit`, { method: "POST", credentials: "same-origin", cache: "no-store", signal: AbortSignal.timeout(30000), headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify({ preview_token: preview.preview_token }) });
      const envelope = await response.json();
      if (!response.ok || envelope.success !== true) throw new Error(envelope?.error?.message || "Import tidak dapat dikomit.");
      const data = envelope.data;
      setResult(`Selesai: ${data.inserted ?? 0} dibuat, ${data.updated ?? 0} diperbarui.${data.created_users_inactive ? ` ${data.created_users_inactive} akun baru dibuat nonaktif.` : ""}`);
      setPreview(null); setFile(null);
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Import tidak dapat dikomit."); }
    finally { setLoading(false); }
  }

  return <>
    <div className="flex flex-wrap gap-2"><a className="admin-button-secondary" href={`/api/backend/admin/${module}/template`}>Unduh template</a><a className="admin-button-secondary" href={`/api/backend/admin/${module}/export`}>Export XLSX</a><button type="button" className="admin-button-secondary" onClick={() => { setOpen(true); setError(""); setResult(""); }}>Import XLSX</button></div>
    {result && <p role="status" className="text-emerald-700">{result}</p>}
    <Dialog open={open} onOpenChange={(value) => { if (!loading) setOpen(value); }}>
      {open && <DialogContent className="max-w-5xl">
        <DialogHeader><DialogTitle>Import {module === "users" ? "admin users" : module === "report-types" ? "jenis laporan" : "FAQ"}</DialogTitle><DialogDescription>Unggah template .xlsx. Preview hanya memvalidasi baris; perubahan data baru terjadi setelah konfirmasi.</DialogDescription></DialogHeader>
        <label className="block">File Excel<input className="admin-input" type="file" accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" onChange={(event) => { setFile(event.target.files?.[0] ?? null); setPreview(null); setError(""); }} /></label>
        {error && <p role="alert" className="text-red-700">{error}</p>}
        {preview && <section className="space-y-3" aria-label="Preview import"><div className="grid grid-cols-2 gap-3 sm:grid-cols-4"><p>Total baris <strong>{preview.total_rows}</strong></p><p>Akan dibuat <strong>{preview.insert_candidates}</strong></p><p>Akan diperbarui <strong>{preview.update_candidates}</strong></p><p>Baris invalid <strong>{preview.invalid_rows}</strong></p></div>{preview.credential_provisioning && module === "users" && <p className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm">{preview.credential_provisioning}</p>}<div className="max-h-72 overflow-auto rounded-lg border border-slate-200"><table className="w-full text-left text-sm"><caption className="sr-only">Hasil validasi baris XLSX</caption><thead className="sticky top-0 bg-slate-50"><tr><th scope="col" className="p-2">Baris</th><th scope="col" className="p-2">Data</th><th scope="col" className="p-2">Hasil</th><th scope="col" className="p-2">Validasi</th></tr></thead><tbody>{preview.rows.map((row) => <tr className="border-t border-slate-200" key={row.line}><td className="p-2">{row.line}</td><td className="max-w-sm p-2"><pre className="whitespace-pre-wrap break-words font-sans">{Object.entries(row.data).map(([key, value]) => `${key}: ${value}`).join(" · ")}</pre></td><td className="p-2">{row.action || "—"}</td><td className="p-2">{row.errors.length ? <ul className="list-inside list-disc text-red-700">{row.errors.map((item) => <li key={item}>{item}</li>)}</ul> : "Valid"}</td></tr>)}</tbody></table></div></section>}
        <div className="flex flex-wrap justify-end gap-2"><button type="button" className="admin-button-secondary" disabled={loading} onClick={() => setOpen(false)}>Tutup</button><button type="button" className="admin-button-secondary" disabled={loading || !file} onClick={() => void upload()}>{loading && !preview ? "Memeriksa…" : "Validasi & preview"}</button>{preview && <button type="button" className="admin-button" disabled={loading || preview.invalid_rows > 0} onClick={() => void commit()}>{loading ? "Mengimpor…" : "Konfirmasi import"}</button>}</div>
      </DialogContent>}
    </Dialog>
  </>;
}
