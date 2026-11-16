import Link from "next/link";
import type { ReportItem, Pagination } from "./contracts";
import { StatusBadge } from "@/components/ui/status-badge";
export { StatusBadge } from "@/components/ui/status-badge";

export function ReportTable({ items }: { items: ReportItem[] }) {
  return <div className="report-table-wrap">
    <div className="report-table-scroll"><table className="report-table"><caption className="sr-only">Daftar laporan</caption><thead><tr>{["Kode", "Pelapor", "Contact", "Jenis", "Status", "Dibuat", "Diperbarui", "Tindakan"].map(label => <th key={label} scope="col">{label}</th>)}</tr></thead><tbody>{items.map(row => <tr key={row.id}><td><Link className="report-code-link" href={`/admin/reports/${row.id}`}>{row.code}</Link></td><td>{row.reporter_name}</td><td className="contact-cell">{row.contact}</td><td>{row.report_type}</td><td><StatusBadge status={row.status}/></td><td>{new Date(row.created_at).toLocaleDateString("id-ID", { dateStyle: "medium" })}</td><td>{new Date(row.updated_at).toLocaleDateString("id-ID", { dateStyle: "medium" })}</td><td><Link className="table-action" href={`/admin/reports/${row.id}`}>Lihat detail <span aria-hidden="true">→</span></Link></td></tr>)}</tbody></table></div>
    <div className="report-cards" aria-label="Daftar laporan dalam tampilan ringkas">{items.map(row => <article className="report-mobile-card" key={row.id}><div className="report-mobile-top"><Link className="report-code-link" href={`/admin/reports/${row.id}`}>{row.code}</Link><StatusBadge status={row.status}/></div><p className="report-mobile-name">{row.reporter_name}</p><p className="report-mobile-meta">{row.report_type} · {new Date(row.created_at).toLocaleDateString("id-ID", { dateStyle: "medium" })}</p><Link className="table-action" href={`/admin/reports/${row.id}`}>Lihat detail <span aria-hidden="true">→</span></Link></article>)}</div>
    {items.length === 0 && <div className="empty-state"><span className="empty-icon" aria-hidden="true">—</span><h3>Belum ada laporan</h3><p>Tidak ada laporan yang sesuai dengan tampilan ini.</p></div>}
  </div>;
}

export function PageControls({ meta, onPage, disabled = false }: { meta: Pagination; onPage: (page: number) => void; disabled?: boolean }) {
  return <nav aria-label="Pagination laporan" className="pagination-controls"><span>Menampilkan halaman <strong>{meta.page}</strong> dari <strong>{Math.max(1, meta.total_pages)}</strong><span className="pagination-total"> · {meta.total.toLocaleString("id-ID")} data</span></span><div><button className="admin-button-secondary" disabled={disabled || meta.page <= 1} onClick={() => onPage(meta.page - 1)}>Sebelumnya</button><button className="admin-button-secondary" disabled={disabled || meta.page >= meta.total_pages} onClick={() => onPage(meta.page + 1)}>Berikutnya</button></div></nav>;
}

export function AccessError({ message, status, retry }: { message: string; status: number; retry: () => void }) {
  const title = status === 401 ? "Sesi Anda berakhir" : status === 403 ? "Akses tidak tersedia" : status === 404 ? "Data tidak ditemukan" : "Data belum dapat dimuat";
  return <section className="error-state" aria-labelledby="error-title"><span className="error-icon" aria-hidden="true">!</span><div><h2 id="error-title">{title}</h2><p role="alert">{message}</p>{status === 401 ? <Link className="admin-button" href="/admin/login">Login kembali</Link> : status !== 403 && <button className="admin-button-secondary" onClick={retry}>Coba lagi</button>}</div></section>;
}

export function ReportSkeleton() {
  return <div role="status" aria-label="Memuat data laporan" className="loading-block"><span className="sr-only">Memuat data…</span><div className="skeleton-line skeleton-title"/><div className="skeleton-line"/><div className="skeleton-line"/><div className="skeleton-line short"/></div>;
}
