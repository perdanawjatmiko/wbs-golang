import "server-only";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { serverConfig } from "@/lib/config/server";
import { isSession, type Session } from "./contracts";
export async function currentSession(): Promise<Session | null> {
    const token = (await cookies()).get("wbs_session")?.value;
    if (!token || !/^[a-f0-9]{64}$/.test(token))
        return null;
    const response = await fetch(`${serverConfig().apiBaseURL}/api/v1/auth/me`, { cache: "no-store", redirect: "error", signal: AbortSignal.timeout(5000), headers: { Cookie: `wbs_session=${token}` } });
    if (response.status === 401)
        return null;
    if (!response.ok)
        throw new Error("Layanan autentikasi belum tersedia.");
    const body: unknown = await response.json();
    if (!body || typeof body !== "object" || !("success" in body) || body.success !== true || !("data" in body) || !isSession(body.data))
        throw new Error("Respons autentikasi tidak valid.");
    return body.data;
}
export async function requireSession() { const session = await currentSession(); if (!session)
    redirect("/admin/login"); return session; }
