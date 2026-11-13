"use client";
import { useEffect, useState, type FormEvent } from "react";
import { adminRequest, AdminApiError } from "@/features/admin/client";
import { ReportTable, PageControls, ReportSkeleton, AccessError } from "./shared";
import { defaultFilters, reportQuery, validateFilters, statuses, type Filters, type ReportList, type Pagination } from "./contracts";
export function AdminReportList() {
    const [draft, setDraft] = useState<Filters>(defaultFilters);
    const [filters, setFilters] = useState<Filters>(defaultFilters);
    const [page, setPage] = useState(1);
    const [data, setData] = useState<ReportList | null>(null);
    const [error, setError] = useState<{
        message: string;
        status: number;
    } | null>(null);
    const [busy, setBusy] = useState(true);
    const [revision, setRevision] = useState(0);
    const [options, setOptions] = useState<{
        id: string;
        name: string;
        is_active: boolean;
    }[]>([]);
    const [optionError, setOptionError] = useState("");
    const [validation, setValidation] = useState("");
    useEffect(() => { let ignore = false; adminRequest<ReportList>(`admin/reports?${reportQuery(filters, page)}`).then(result => { if (!ignore) {
        setData(result);
        setError(null);
        setBusy(false);
    } }, err => { if (!ignore) {
        setData(null);
        setError({ message: err.message, status: err instanceof AdminApiError ? err.status : 0 });
        setBusy(false);
    } }); return () => { ignore = true; }; }, [filters, page, revision]);
    useEffect(() => { let ignore = false; async function load() { const rows: typeof options = []; for (let page = 1; page <= 100; page++) {
        const result = await adminRequest<{
            items: typeof options;
            pagination: Pagination;
        }>(`admin/reports/options?per_page=100&page=${page}`);
        rows.push(...result.items);
        if (rows.length >= result.pagination.total)
            break;
    } return rows; } load().then(rows => { if (!ignore)
        setOptions(rows); }, () => { if (!ignore)
        setOptionError("Pilihan jenis laporan belum dapat dimuat. Muat ulang halaman."); }); return () => { ignore = true; }; }, []);
    function apply(e: FormEvent) { e.preventDefault(); const message = validateFilters(draft); setValidation(message ?? ""); if (message)
        return; setBusy(true); setFilters({ ...draft }); setPage(1); }
    function field(key: keyof Filters, value: string) { setDraft(prev => ({ ...prev, [key]: value })); }
    if (error && (error.status === 401 || error.status === 403))
        return <AccessError {...error} retry={() => { }}/>;
    return <section className="space-y-6"><h1 className="text-3xl font-semibold">Laporan</h1><form className="grid gap-4 rounded-xl border border-zinc-200 p-5 sm:grid-cols-2 lg:grid-cols-4 dark:border-zinc-800" onSubmit={apply}><label>Cari kode, pelapor, contact<input className="admin-input" maxLength={200} value={draft.search} onChange={e => field("search", e.target.value)}/></label><label>Status<select className="admin-input" value={draft.status} onChange={e => field("status", e.target.value)}><option value="">Semua status</option>{statuses.map(s => <option key={s}>{s}</option>)}</select></label><label>Jenis laporan<select className="admin-input" value={draft.report_type_id} onChange={e => field("report_type_id", e.target.value)}><option value="">Semua jenis</option>{options.map(o => <option key={o.id} value={o.id}>{o.name}{o.is_active ? "" : " (nonaktif)"}</option>)}</select></label><label>Dari tanggal<input className="admin-input" type="date" value={draft.date_from} onChange={e => field("date_from", e.target.value)}/></label><label>Sampai tanggal<input className="admin-input" type="date" value={draft.date_to} onChange={e => field("date_to", e.target.value)}/></label><label>Urut berdasarkan<select className="admin-input" value={draft.sort} onChange={e => field("sort", e.target.value)}><option value="created_at">Dibuat</option><option value="updated_at">Diperbarui</option><option value="status">Status</option></select></label><label>Arah<select className="admin-input" value={draft.direction} onChange={e => field("direction", e.target.value)}><option value="desc">Menurun</option><option value="asc">Menaik</option></select></label><label>Data per halaman<select className="admin-input" value={draft.per_page} onChange={e => field("per_page", e.target.value)}>{[20, 50, 100].map(n => <option key={n}>{n}</option>)}</select></label><button className="admin-button" disabled={busy}>Terapkan filter</button></form>{validation && <p role="alert">{validation}</p>}{optionError && <p role="alert">{optionError}</p>}{busy ? <ReportSkeleton /> : error ? <AccessError {...error} retry={() => { setBusy(true); setRevision(v => v + 1); }}/> : data && <><ReportTable items={data.items}/><PageControls meta={data.pagination} onPage={page => { setBusy(true); setPage(page); }}/></>}</section>;
}
