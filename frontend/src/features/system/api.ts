import "server-only";
import { apiGet } from "@/lib/api/client";
export type SystemInfo = { app_name: string; environment: string; version: string };
function isSystemInfo(value: unknown): value is SystemInfo {
  return !!value && typeof value === "object" &&
    "app_name" in value && typeof value.app_name === "string" &&
    "environment" in value && typeof value.environment === "string" &&
    "version" in value && typeof value.version === "string";
}
export const getSystemInfo = () => apiGet("/api/v1/system/info", isSystemInfo);
