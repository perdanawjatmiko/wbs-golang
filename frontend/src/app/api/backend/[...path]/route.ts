import { NextRequest } from "next/server";
import { serverConfig } from "@/lib/config/server";
import { bridgePath } from "@/features/admin/bridge";
export const dynamic = "force-dynamic";
function failure(status: number, message: string) { return Response.json({ success: false, error: { code: "BRIDGE_ERROR", message }, meta: { request_id: "frontend" } }, { status, headers: { "Cache-Control": "no-store" } }); }
async function forward(request: NextRequest, context: {
    params: Promise<{
        path: string[];
    }>;
}) {
    const { path } = await context.params;
    const route = bridgePath(path, request.method);
    if (!route)
        return failure(404, "Endpoint tidak ditemukan.");
    const config = serverConfig();
    if (request.method !== "GET" && request.headers.get("origin") !== config.appOrigin)
        return failure(403, "Origin request tidak valid.");
    const headers = new Headers({ Accept: "application/json" });
    const token = request.cookies.get("wbs_session")?.value;
    if (token && /^[a-f0-9]{64}$/.test(token))
        headers.set("Cookie", `wbs_session=${token}`);
    if (request.method !== "GET") {
        headers.set("Content-Type", "application/json");
        headers.set("Origin", config.appOrigin);
        headers.set("X-CSRF-Token", request.headers.get("x-csrf-token") ?? "");
    }
    try {
        let body: string | undefined;
        if (["POST", "PATCH"].includes(request.method)) {
            // Bound streamed input before buffering or forwarding.
            const reader = request.body?.getReader();
            let size = 0;
            const chunks: Uint8Array[] = [];
            if (reader) {
                for (;;) {
                    const chunk = await reader.read();
                    if (chunk.done)
                        break;
                    size += chunk.value.byteLength;
                    if (size > 1048576) {
                        await reader.cancel();
                        return failure(413, "Request terlalu besar.");
                    }
                    chunks.push(chunk.value);
                }
            }
            body = Buffer.concat(chunks).toString("utf8");
        }
        const query = new URLSearchParams();
        for (const key of ["page", "page_size", "search", "is_active", "report_type_id"]) {
            const v = request.nextUrl.searchParams.get(key);
            if (v !== null)
                query.set(key, v);
        }
        const upstream = await fetch(`${config.apiBaseURL}/api/v1/${route}?${query}`, { method: request.method, headers, body, cache: "no-store", redirect: "error", signal: AbortSignal.timeout(7000) });
        const responseHeaders = new Headers({ "Content-Type": "application/json", "Cache-Control": "no-store" });
        for (const cookie of upstream.headers.getSetCookie())
            if (cookie.startsWith("wbs_session="))
                responseHeaders.append("Set-Cookie", cookie);
        return new Response(await upstream.text(), { status: upstream.status, headers: responseHeaders });
    }
    catch {
        return failure(503, "Backend tidak tersedia. Silakan coba lagi.");
    }
}
export { forward as GET, forward as POST, forward as PATCH, forward as DELETE };
