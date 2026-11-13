import type { ApiResponse } from "@/types/api";
export class AdminApiError extends Error {
    constructor(public readonly status: number, message: string,public readonly code="REQUEST_FAILED") { super(message); }
}
export async function adminRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
    try {
        const response = await fetch(`/api/backend/${path}`, { ...options, credentials: "same-origin", cache: "no-store", signal: AbortSignal.timeout(8000), headers: { "Content-Type": "application/json", ...options.headers } });
        const envelope: ApiResponse<T> = await response.json();
        if (!response.ok || envelope.success !== true) {
            const message = response.status === 401 ? "Sesi berakhir. Silakan login kembali." : response.status === 403 ? "Akses ditolak atau token CSRF tidak valid. Muat ulang halaman." : envelope.success === false && typeof envelope.error?.message === "string" ? envelope.error.message : "Request gagal.";
            throw new AdminApiError(response.status, message,envelope.success===false?envelope.error?.code:"REQUEST_FAILED");
        }
        return envelope.data;
    }
    catch (error) {
        if (error instanceof AdminApiError)
            throw error;
        throw new AdminApiError(0, "Backend tidak tersedia. Silakan coba lagi.");
    }
}
