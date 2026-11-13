export type ApiResponse<T> =
  | { success: true; data: T; meta: { request_id: string } }
  | { success: false; error: { code: string; message: string }; meta: { request_id: string } };
export class ApiError extends Error {
  constructor(public readonly code: string, public readonly status?: number) {
    super("Koneksi backend belum tersedia. Silakan coba lagi.");
    this.name = "ApiError";
  }
}
