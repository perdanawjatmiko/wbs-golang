import Link from "next/link";
import type { ReportItem, Pagination } from "./contracts";
export function ReportTable({ items }: {
    items: ReportItem[];
}) { return <div className="overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800"><table className="w-full text-left text-sm"><caption className="sr-only">Daftar laporan</caption><thead><tr>{["Kode", "Pelapor", "Contact", "Jenis", "Status", "Dibuat", "Diperbarui", "Tindakan"].map(label => <th key={label} scope="col" className="p-4">{label}</th>)}</tr></thead><tbody>{items.map(row => <tr key={row.id} className="border-t border-zinc-200 dark:border-zinc-800"><td className="whitespace-nowrap p-4 font-mono">{row.code}</td><td className="p-4">{row.reporter_name}</td><td className="break-all p-4">{row.contact}</td><td className="p-4">{row.report_type}</td><td className="whitespace-nowrap p-4">{row.status}</td><td className="p-4">{new Date(row.created_at).toLocaleString("id-ID")}</td><td className="p-4">{new Date(row.updated_at).toLocaleString("id-ID")}</td><td className="p-4"><Link className="underline" href={`/admin/reports/${row.id}`}>Detail</Link></td></tr>)}</tbody></table>{items.length === 0 && <p className="p-6">Belum ada laporan yang cocok.</p>}</div>; }
export function PageControls({ meta, onPage, disabled = false }: {
    meta: Pagination;
    onPage: (page: number) => void;
    disabled?: boolean;
}) { return <nav aria-label="Pagination laporan" className="flex flex-wrap items-center gap-4"><button className="admin-button-secondary" disabled={disabled || meta.page <= 1} onClick={() => onPage(meta.page - 1)}>Sebelumnya</button><span>Halaman {meta.page} / {Math.max(1, meta.total_pages)} · {meta.total} data</span><button className="admin-button-secondary" disabled={disabled || meta.page >= meta.total_pages} onClick={() => onPage(meta.page + 1)}>Berikutnya</button></nav>; }
export function AccessError({ message, status, retry }: {
    message: string;
    status: number;
    retry: () => void;
}) { return <section><p role="alert" className="mb-4">{message}</p>{status === 401 ? <Link className="admin-button" href="/admin/login">Login kembali</Link> : status !== 403 && <button className="admin-button-secondary" onClick={retry}>Coba lagi</button>}</section>; }
export function ReportSkeleton() { return <div role="status" aria-label="Memuat laporan" className="animate-pulse space-y-4"><p>Memuat laporan…</p><div className="h-24 rounded-xl bg-zinc-200 dark:bg-zinc-800"/><div className="h-48 rounded-xl bg-zinc-100 dark:bg-zinc-900"/></div>; }
