import "server-only";
import { serverConfig } from "@/lib/config/server";
import { ApiError, type ApiResponse } from "@/types/api";

export async function apiGet<T>(path: string, validate: (data: unknown) => data is T): Promise<T> {
  const config = serverConfig();
  if (!path.startsWith("/api/v1/") || path.includes("..")) throw new ApiError("INVALID_PATH");
  try {
    const response = await fetch(`${config.apiBaseURL}${path}`, {
      cache: "no-store",
      signal: AbortSignal.timeout(config.apiTimeoutMs),
      redirect: "error",
      headers: { Accept: "application/json" },
    });
    const body: unknown = await response.json();
    if (!body || typeof body !== "object" || !("success" in body) || !("meta" in body)) throw new ApiError("INVALID_RESPONSE", response.status);
    const envelope = body as ApiResponse<unknown>;
    if (!envelope.meta || typeof envelope.meta.request_id !== "string") throw new ApiError("INVALID_RESPONSE", response.status);
    if (!response.ok || envelope.success !== true) throw new ApiError("BACKEND_ERROR", response.status);
    if (!validate(envelope.data)) throw new ApiError("INVALID_RESPONSE", response.status);
    return envelope.data;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    throw new ApiError("BACKEND_UNAVAILABLE");
  }
}
