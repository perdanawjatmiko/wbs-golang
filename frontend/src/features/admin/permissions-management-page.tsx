"use client";

import Link from "next/link";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { AdminApiError, adminRequest } from "./client";
import type { PageData, Row } from "./contracts";

type Permission = Row & { slug: string; module_name: string; description: string };
const PAGE_SIZE = 20;
const EMPTY_PAGE: PageData = { items: [], total: 0, page: 1, page_size: PAGE_SIZE };

export function PermissionsManagementPage({ csrf }: { csrf: string }) {
  const [data, setData] = useState<PageData>(EMPTY_PAGE);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [query, setQuery] = useState("");
  const [revision, setRevision] = useState(0);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState("");
  const [expired, setExpired] = useState(false);
  const [editing, setEditing] = useState<Permission | null | undefined>(undefined);
  const [pendingDelete, setPendingDelete] = useState<Permission | null>(null);
  const [slug, setSlug] = useState("");
  const [moduleName, setModuleName] = useState("");
  const [description, setDescription] = useState("");

  const reload = useCallback(() => { setLoading(true); setRevision((value) => value + 1); }, []);

  useEffect(() => {
    let cancelled = false;
    adminRequest<PageData>(`admin/permissions?page=${page}&page_size=${PAGE_SIZE}&search=${encodeURIComponent(query)}`).then(
      (result) => { if (!cancelled) { setData(result); setError(""); setLoading(false); } },
      (reason: unknown) => {
        if (!cancelled) {
          setData({ ...EMPTY_PAGE, page });
          setError(reason instanceof Error ? reason.message : "Gagal memuat permissions.");
          setExpired(reason instanceof AdminApiError && reason.status === 401);
          setLoading(false);
        }
      },
    );
    return () => { cancelled = true; };
  }, [page, query, revision]);

  function open(permission: Permission | null) {
    setEditing(permission); setSlug(permission?.slug ?? ""); setModuleName(permission?.module_name ?? ""); setDescription(permission?.description ?? ""); setError(""); setFeedback("");
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedSlug = slug.trim();
    const normalizedModule = moduleName.trim();
    if (!/^[a-z][a-z-]*\.[a-z][a-z-]*$/.test(normalizedSlug)) { setError("Slug harus mengikuti format module.action."); return; }
    if (!normalizedModule) { setError("Nama module wajib diisi."); return; }
    if (normalizedModule.length > 100 || description.length > 10000) { setError("Module maksimal 100 karakter dan deskripsi maksimal 10.000 karakter."); return; }
    setSaving(true); setError("");
    try {
      await adminRequest(`admin/permissions${editing ? `/${editing.id}` : ""}`, {
        method: editing ? "PATCH" : "POST",
        headers: { "X-CSRF-Token": csrf },
        body: JSON.stringify({ slug: normalizedSlug, module_name: normalizedModule, description: description.trim() }),
      });
      setEditing(undefined); setFeedback("Permission berhasil disimpan."); reload();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Gagal menyimpan permission.");
      if (reason instanceof AdminApiError && reason.status === 401) setExpired(true);
    } finally { setSaving(false); }
  }

  async function remove() {
    const permission = pendingDelete;
    if (!permission) return;
    setSaving(true); setPendingDelete(null); setError("");
    try {
      await adminRequest(`admin/permissions/${permission.id}`, { method: "DELETE", headers: { "X-CSRF-Token": csrf } });
      setFeedback("Permission berhasil dihapus."); reload();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Gagal menghapus permission.");
      if (reason instanceof AdminApiError && reason.status === 401) setExpired(true);
    } finally { setSaving(false); }
  }

  if (expired) return <section><h1 className="text-2xl font-semibold">Sesi berakhir</h1><p role="alert" className="my-4">Silakan login kembali untuk melanjutkan.</p><Link className="admin-button" href="/admin/login">Login kembali</Link></section>;

  return (
    <section className="space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div><h1 className="text-3xl font-semibold">Permissions</h1><p className="mt-1 text-sm text-slate-600">Kelola permission yang dapat diberikan melalui role.</p></div>
        <button className="admin-button" onClick={() => open(null)} disabled={saving}>Tambah permission</button>
      </header>
      {feedback && <p role="status" className="text-emerald-700">{feedback}</p>}
      {error && <div role="alert" className="text-red-700">{error} <button className="underline" onClick={reload}>Coba lagi</button></div>}
      <form className="flex flex-wrap items-end gap-3" onSubmit={(event) => { event.preventDefault(); setPage(1); setQuery(search); setLoading(true); setRevision((value) => value + 1); }}>
        <label className="min-w-48 flex-1">Cari permission<input className="admin-input" value={search} maxLength={200} onChange={(event) => setSearch(event.target.value)} /></label>
        <button className="admin-button-secondary">Cari</button>
      </form>
      {loading ? <p role="status">Memuat permissions…</p> : <>
        <div className="overflow-x-auto rounded-xl border border-slate-200"><table className="w-full text-left text-sm"><caption className="sr-only">Daftar permissions</caption><thead><tr><th className="p-4" scope="col">Slug</th><th className="p-4" scope="col">Module</th><th className="p-4" scope="col">Deskripsi</th><th className="p-4" scope="col">Tindakan</th></tr></thead><tbody>{data.items.map((item) => { const permission = item as Permission; return <tr className="border-t border-slate-200" key={permission.id}><td className="p-4 font-mono">{permission.slug}</td><td className="p-4">{permission.module_name}</td><td className="max-w-md truncate p-4">{permission.description || "—"}</td><td className="whitespace-nowrap p-4"><button className="mr-4 underline" disabled={saving} onClick={() => open(permission)}>Edit</button><button className="text-red-700 underline" disabled={saving} onClick={() => setPendingDelete(permission)}>Hapus</button></td></tr>; })}</tbody></table>{!data.items.length && <p className="p-6">Belum ada permission yang cocok.</p>}</div>
        <nav aria-label="Pagination permissions" className="flex items-center gap-4"><button className="admin-button-secondary" disabled={page <= 1} onClick={() => { setLoading(true); setPage((value) => value - 1); }}>Sebelumnya</button><span>Halaman {page} · {data.total} data</span><button className="admin-button-secondary" disabled={page * PAGE_SIZE >= data.total} onClick={() => { setLoading(true); setPage((value) => value + 1); }}>Berikutnya</button></nav>
      </>}
      <Dialog open={editing !== undefined} onOpenChange={(openState) => { if (!openState && !saving) setEditing(undefined); }}>
        {editing !== undefined && <DialogContent><form onSubmit={save} className="space-y-5" aria-label={editing ? "Edit permission" : "Tambah permission"}><DialogHeader><DialogTitle>{editing ? "Edit permission" : "Tambah permission"}</DialogTitle><DialogDescription>Gunakan format module.action yang konsisten untuk permission.</DialogDescription></DialogHeader><label className="block">Slug <span aria-hidden="true">*</span><span className="mb-1 block text-sm text-slate-600">Contoh: report.view.</span><input className="admin-input font-mono" required maxLength={100} value={slug} onChange={(event) => setSlug(event.target.value)} /></label><label className="block">Module <span aria-hidden="true">*</span><input className="admin-input" required maxLength={100} value={moduleName} onChange={(event) => setModuleName(event.target.value)} /></label><label className="block">Deskripsi<textarea className="admin-input" rows={4} maxLength={10000} value={description} onChange={(event) => setDescription(event.target.value)} /></label><div className="flex gap-3"><button className="admin-button" disabled={saving}>{saving ? "Menyimpan…" : "Simpan"}</button><button type="button" className="admin-button-secondary" disabled={saving} onClick={() => setEditing(undefined)}>Batal</button></div></form></DialogContent>}
      </Dialog>
      <AlertDialog open={pendingDelete !== null} onOpenChange={(openState) => { if (!openState && !saving) setPendingDelete(null); }}><AlertDialogContent><AlertDialogHeader className="text-left"><AlertDialogTitle>Hapus permission ini?</AlertDialogTitle><AlertDialogDescription>Permission yang masih digunakan atau direferensikan tidak dapat dihapus.</AlertDialogDescription></AlertDialogHeader><AlertDialogFooter><AlertDialogCancel variant="outline" disabled={saving}>Batal</AlertDialogCancel><AlertDialogAction variant="destructive" disabled={saving} onClick={() => void remove()}>Hapus</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog>
    </section>
  );
}
