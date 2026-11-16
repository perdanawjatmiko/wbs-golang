"use client";
import { useEffect, useState } from "react";
import Link from "next/link";
import { adminRequest, AdminApiError } from "@/features/admin/client";
import { ReportTable, ReportSkeleton, AccessError, StatusBadge } from "./shared";
import type { DashboardData } from "./contracts";
export function DashboardMetrics({ data }: {
    data: DashboardData;
}) { const metrics = [["Total laporan", data.total_reports, "#1d4ed8"], ["Open", data.open_reports, "#1d4ed8"], ["On Progress", data.on_progress_reports, "#b45309"], ["Done", data.done_reports, "#0f766e"], ["Rejected", data.rejected_reports, "#b91c1c"]] as const; return <div className="dashboard-stack"><div className="page-heading"><div><h1>Ringkasan laporan</h1><p>Pantau status laporan dan aktivitas terbaru.</p></div></div><div className="metrics-grid">{metrics.map(([label, total, color]) => <section aria-label={label} key={label} className="metric-card"><p>{label}</p><strong>{total.toLocaleString("id-ID")}</strong><span className="metric-accent" style={{ "--metric-color": color } as React.CSSProperties}/></section>)}</div><div className="dashboard-panels"><section className="admin-panel dashboard-panel"><div className="section-heading"><div><h2>Distribusi status</h2><p>Jumlah laporan pada setiap status.</p></div></div>{data.total_reports === 0 ? <p className="empty-copy">Belum ada data laporan.</p> : <div className="distribution-list">{data.by_status.map(row => <div className="distribution-row" key={row.status}><div className="distribution-meta"><StatusBadge status={row.status}/><strong>{row.total}</strong></div><div className="distribution-track"><span className={`distribution-fill distribution-${row.status === "On Progress" ? "progress" : row.status.toLowerCase()}`} style={{ width: `${Math.max(2, row.total / data.total_reports * 100)}%` }}/></div></div>)}</div>}</section><section className="admin-panel dashboard-panel"><div className="section-heading"><div><h2>Distribusi jenis laporan</h2><p>Distribusi berdasarkan kategori yang tersedia.</p></div></div>{data.by_report_type.length === 0 ? <p className="empty-copy">Belum ada data jenis laporan.</p> : <div className="type-distribution">{data.by_report_type.map(row => <div className="type-row" key={row.id}><span>{row.name}</span><strong>{row.total}</strong></div>)}</div>}</section></div><section className="admin-panel dashboard-panel"><div className="section-heading"><div><h2>Laporan terbaru</h2><p>Aktivitas terbaru yang dapat Anda akses.</p></div><Link className="text-link" href="/admin/reports">Semua laporan <span aria-hidden="true">→</span></Link></div><ReportTable items={data.recent_reports}/></section></div>; }
export function AdminDashboard() { const [data, setData] = useState<DashboardData | null>(null); const [error, setError] = useState<{
    message: string;
    status: number;
} | null>(null); const [revision, setRevision] = useState(0); useEffect(() => { let ignore = false; adminRequest<DashboardData>("admin/dashboard").then(result => { if (!ignore) {
    setData(result);
    setError(null);
} }, err => { if (!ignore) {
    setData(null);
    setError({ message: err.message, status: err instanceof AdminApiError ? err.status : 0 });
} }); return () => { ignore = true; }; }, [revision]); return <div className="dashboard-stack">{error ? <AccessError {...error} retry={() => { setError(null); setRevision(v => v + 1); }}/> : data ? <DashboardMetrics data={data}/> : <ReportSkeleton />}</div>; }
