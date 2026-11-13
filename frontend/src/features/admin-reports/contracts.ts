import type { ReportStatus } from "@/features/reports/contracts";
export { statuses } from "@/features/reports/contracts";
export type Pagination = {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
};
export type ReportItem = {
    id: string;
    code: string;
    reporter_name: string;
    contact: string;
    report_type_id: string;
    report_type: string;
    status: ReportStatus;
    created_at: string;
    updated_at: string;
};
export type ReportList = {
    items: ReportItem[];
    pagination: Pagination;
};
export type ReportHistory = {
    id: string;
    old_status: ReportStatus;
    new_status: ReportStatus;
    internal_note: string;
    external_note: string;
    created_at: string;
    admin: {
        id: string;
        name: string;
    };
};
export type ReportDetail = ReportItem & {
    contact_type: "EMAIL" | "PHONE";
    detail: string;
    attachment: {
        available: boolean;
        name?: string;
        size?: number;
        content_type?: string;
    };
    histories: ReportHistory[];
    history_pagination: Pagination;
};
export type DashboardData = {
    total_reports: number;
    open_reports: number;
    on_progress_reports: number;
    done_reports: number;
    rejected_reports: number;
    recent_reports: ReportItem[];
    by_status: {
        status: ReportStatus;
        total: number;
    }[];
    by_report_type: {
        id: string;
        name: string;
        total: number;
    }[];
};
export type StatusInput = {
    status: ReportStatus;
    internal_note: string;
    external_note: string;
    expected_updated_at: string;
};
export type NoteInput = Omit<StatusInput, "status">;
export type Filters = {
    search: string;
    status: string;
    report_type_id: string;
    date_from: string;
    date_to: string;
    sort: string;
    direction: string;
    per_page: string;
};
export const defaultFilters: Filters = { search: "", status: "", report_type_id: "", date_from: "", date_to: "", sort: "created_at", direction: "desc", per_page: "20" };
export function reportQuery(filters: Filters, page: number): string { const query = new URLSearchParams({ page: String(page) }); for (const [key, value] of Object.entries(filters))
    if (value)
        query.set(key, value); return query.toString(); }
export function validateFilters(filters: Filters): string | null { if (Array.from(filters.search).length > 200)
    return "Pencarian maksimum 200 karakter."; if (filters.date_from && filters.date_to && filters.date_from > filters.date_to)
    return "Tanggal akhir tidak boleh sebelum tanggal awal."; return null; }
export function validateNotes(input: NoteInput): string | null { if (!input.internal_note.trim() && !input.external_note.trim())
    return "Isi minimal satu catatan."; if ([input.internal_note, input.external_note].some(v => Array.from(v.trim()).length > 10000))
    return "Maksimum 10.000 karakter per catatan."; if (!input.expected_updated_at)
    return "Muat ulang versi laporan."; return null; }
export function validateStatus(input: StatusInput, current: ReportStatus): string | null { if (!["Open", "On Progress", "Done", "Rejected"].includes(input.status))
    return "Status tidak valid."; if (input.status === current)
    return "Status sudah sama. Gunakan form catatan."; if (input.status === "Rejected" && !input.external_note.trim())
    return "Alasan penolakan pada catatan eksternal wajib diisi."; if ([input.internal_note, input.external_note].some(v => Array.from(v.trim()).length > 10000))
    return "Maksimum 10.000 karakter per catatan."; if (!input.expected_updated_at)
    return "Muat ulang versi laporan."; return null; }

export function workflowErrorMessage(code:string|undefined,message:string):string{return code==="REPORT_ACTIVE_LIMIT_REACHED"?"Reaktivasi akan melampaui batas maksimal 5 laporan aktif untuk contact ini.":message;}
