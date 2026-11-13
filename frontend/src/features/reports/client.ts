import type { ApiResponse } from "@/types/api";
import { isCreated, isTracked, type CreatedReport, type TrackedReport, type ReportInput, type Page, type FAQ, type ReportType, type UploadPolicy } from "./contracts";
export class PublicApiError extends Error {
    constructor(public readonly code: string, public readonly status: number, message: string, public readonly uncertain = false) { super(message); }
}
export async function publicRequest<T>(path: string, options: RequestInit = {}, validate?: (data: unknown) => data is T): Promise<T> {
    const submission = path === "reports" && options.method === "POST";
    try {
        const response = await fetch(`/api/public/${path}`, { ...options, cache: "no-store", credentials: "omit", signal: AbortSignal.timeout(20000), redirect: "error" });
        const body: ApiResponse<T> = await response.json();
        if (!response.ok || body.success !== true) {
            const code = body.success === false ? body.error?.code : "INVALID_RESPONSE";
            const uncertain = submission && (response.status >= 500 || code === "SUBMISSION_UNCERTAIN");
            const message = code === "REPORT_ACTIVE_LIMIT_REACHED" ? "Anda telah mencapai batas maksimal 5 laporan aktif." : code === "REPORT_NOT_FOUND" ? "Laporan tidak ditemukan. Periksa kode dan contact." : uncertain ? "Pengiriman belum dapat dipastikan. Jangan langsung mengirim ulang; coba tracking atau hubungi pengelola." : body.success === false && typeof body.error?.message === "string" ? body.error.message : "Request tidak berhasil.";
            throw new PublicApiError(code ?? "REQUEST_FAILED", response.status, message, uncertain);
        }
        if (validate && !validate(body.data))
            throw new PublicApiError("INVALID_RESPONSE", 0, submission ? "Respons pengiriman tidak dapat diverifikasi. Jangan langsung mengirim ulang." : "Respons layanan tidak valid.", submission);
        return body.data;
    }
    catch (error) {
        if (error instanceof PublicApiError)
            throw error;
        throw new PublicApiError("NETWORK_ERROR", 0, submission ? "Koneksi terputus atau timeout. Pengiriman mungkin sudah tersimpan; jangan langsung mengirim ulang. Coba tracking jika kode sudah diterima atau hubungi pengelola." : "Layanan belum dapat dihubungi. Silakan coba lagi.", submission);
    }
}
export function submitReport(input: ReportInput, file: File | null): Promise<CreatedReport> { const data = new FormData(); for (const [key, value] of Object.entries(input))
    data.set(key, value); if (file)
    data.set("attachment", file); return publicRequest("reports", { method: "POST", body: data }, isCreated); }
export function trackReport(report_code: string, contact: string): Promise<TrackedReport> { return publicRequest("reports/track", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ report_code, contact }) }, isTracked); }
export async function allMaster<T>(path: string): Promise<T[]> { const items: T[] = []; for (let page = 1; page <= 100; page++) {
    const result = await publicRequest<Page<T>>(`${path}${path.includes("?") ? "&" : "?"}page_size=100&page=${page}`);
    items.push(...result.items);
    if (items.length >= result.total)
        return items;
} throw new PublicApiError("CATALOG_TOO_LARGE", 0, "Daftar terlalu besar untuk dimuat. Hubungi pengelola."); }
export const reportTypes = () => allMaster<ReportType>("report-types");
export const faqs = (type: string) => allMaster<FAQ>(`faqs?report_type_id=${encodeURIComponent(type)}`);
export const uploadPolicy = () => publicRequest<UploadPolicy>("reports/options");
