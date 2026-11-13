"use client";
import { useEffect, useState } from "react";
import { adminRequest, AdminApiError } from "@/features/admin/client";
import { ReportTable, ReportSkeleton, AccessError } from "./shared";
import type { DashboardData } from "./contracts";
export function DashboardMetrics({ data }: {
    data: DashboardData;
}) { return <div className="space-y-8"><div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">{[["Total", data.total_reports], ["Open", data.open_reports], ["On Progress", data.on_progress_reports], ["Done", data.done_reports], ["Rejected", data.rejected_reports]].map(([label, total]) => <section aria-label={String(label)} key={label} className="rounded-xl border border-zinc-200 p-5 dark:border-zinc-800"><h2 className="text-sm text-zinc-500">{label}</h2><p className="mt-2 text-3xl font-semibold">{total}</p></section>)}</div><div className="grid gap-6 lg:grid-cols-2"><section><h2 className="mb-4 text-xl font-semibold">Distribusi status</h2><table className="w-full text-left"><caption className="sr-only">Jumlah per status</caption><thead><tr><th scope="col">Status</th><th scope="col">Jumlah</th></tr></thead><tbody>{data.by_status.map(row => <tr key={row.status}><th className="py-3 font-normal" scope="row">{row.status}</th><td>{row.total}<div aria-hidden="true" className="mt-1 h-2 rounded bg-blue-600" style={{ width: `${data.total_reports ? row.total / data.total_reports * 100 : 0}%` }}/></td></tr>)}</tbody></table></section><section><h2 className="mb-4 text-xl font-semibold">Distribusi jenis laporan</h2>{data.by_report_type.length === 0 ? <p>Belum ada data.</p> : <table className="w-full text-left"><caption className="sr-only">Jumlah per jenis laporan</caption><thead><tr><th scope="col">Jenis</th><th scope="col">Jumlah</th></tr></thead><tbody>{data.by_report_type.map(row => <tr key={row.id}><th scope="row" className="py-3 font-normal">{row.name}</th><td>{row.total}</td></tr>)}</tbody></table>}</section></div><section><h2 className="mb-4 text-xl font-semibold">Laporan terbaru</h2><ReportTable items={data.recent_reports}/></section></div>; }
export function AdminDashboard() { const [data, setData] = useState<DashboardData | null>(null); const [error, setError] = useState<{
    message: string;
    status: number;
} | null>(null); const [revision, setRevision] = useState(0); useEffect(() => { let ignore = false; adminRequest<DashboardData>("admin/dashboard").then(result => { if (!ignore) {
    setData(result);
    setError(null);
} }, err => { if (!ignore) {
    setData(null);
    setError({ message: err.message, status: err instanceof AdminApiError ? err.status : 0 });
} }); return () => { ignore = true; }; }, [revision]); return <div className="space-y-6"><h1 className="text-3xl font-semibold">Dashboard laporan</h1>{error ? <AccessError {...error} retry={() => { setError(null); setRevision(v => v + 1); }}/> : data ? <DashboardMetrics data={data}/> : <ReportSkeleton />}</div>; }
