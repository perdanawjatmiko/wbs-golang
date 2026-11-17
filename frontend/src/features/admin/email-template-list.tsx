"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { adminRequest, AdminApiError } from "./client";
import type { PageData, Row } from "./contracts";

type Template = Row & { event_key: string; subject: string };
const pageSize = 20;

export function EmailTemplateList() {
  const [data, setData] = useState<PageData | null>(null);
  const [page, setPage] = useState(1);
  const [revision, setRevision] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [expired, setExpired] = useState(false);
  useEffect(() => {
    let cancelled = false;
    adminRequest<PageData>(`admin/email-templates?page=${page}&page_size=${pageSize}`).then((result) => {
      if (!cancelled) { setData(result); setError(""); setLoading(false); }
    }, (reason: unknown) => {
      if (!cancelled) { setError(reason instanceof Error ? reason.message : "Template email gagal dimuat."); setExpired(reason instanceof AdminApiError && reason.status === 401); setLoading(false); }
    });
    return () => { cancelled = true; };
  }, [page, revision]);
  return <section className="space-y-6">
    <header className="flex flex-wrap items-center justify-between gap-4"><div><p className="public-eyebrow">Master data</p><h1 className="text-3xl font-semibold">Email templates</h1><p className="mt-1 text-sm text-slate-600">Kelola isi notifikasi email untuk event laporan.</p></div><Link className="admin-button" href="/admin/email-templates/new">Buat template</Link></header>
    {expired ? <p role="alert">Sesi berakhir. <Link className="underline" href="/admin/login">Login kembali</Link></p> : error ? <p role="alert" className="text-red-700">{error} <button className="underline" onClick={() => { setLoading(true); setRevision((value) => value + 1); }}>Coba lagi</button></p> : loading ? <p role="status">Memuat email templates…</p> : <>
      <div className="overflow-x-auto rounded-xl border border-slate-200"><table className="w-full text-left text-sm"><caption className="sr-only">Daftar email templates</caption><thead><tr><th scope="col" className="p-4">Event key</th><th scope="col" className="p-4">Subject</th><th scope="col" className="p-4">Tindakan</th></tr></thead><tbody>{data?.items.map((item) => { const template = item as Template; return <tr key={template.id} className="border-t border-slate-200"><td className="p-4 font-mono">{template.event_key}</td><td className="p-4">{template.subject}</td><td className="p-4"><Link className="underline" href={`/admin/email-templates/${template.id}/edit`}>Edit</Link></td></tr>; })}</tbody></table>{data?.items.length === 0 && <p className="p-6">Belum ada email template.</p>}</div>
      <nav aria-label="Pagination email templates" className="flex items-center gap-4"><button className="admin-button-secondary" disabled={page <= 1} onClick={() => { setLoading(true); setPage((value) => value - 1); }}>Sebelumnya</button><span>Halaman {page} · {data?.total ?? 0} data</span><button className="admin-button-secondary" disabled={!data || page * pageSize >= data.total} onClick={() => { setLoading(true); setPage((value) => value + 1); }}>Berikutnya</button></nav>
    </>}
  </section>;
}
