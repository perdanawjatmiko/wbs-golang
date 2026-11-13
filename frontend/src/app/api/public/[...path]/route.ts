import { NextRequest } from "next/server";
import { allowedPublicPath } from "@/features/reports/bridge";
import { serverConfig } from "@/lib/config/server";
export const dynamic = "force-dynamic";
function failure(status: number, code: string, message: string) { return Response.json({ success: false, error: { code, message }, meta: { request_id: "frontend" } }, { status, headers: { "Cache-Control": "no-store, private", "X-Content-Type-Options": "nosniff" } }); }
async function forward(request: NextRequest, { params }: {
    params: Promise<{
        path: string[];
    }>;
}) {
    const { path } = await params;
    if (!allowedPublicPath(path, request.method))
        return failure(404, "NOT_FOUND", "Endpoint tidak ditemukan.");
    const config = serverConfig();
    if (request.method === "POST" && request.headers.get("origin") !== config.appOrigin)
        return failure(403, "ORIGIN_FORBIDDEN", "Origin request tidak valid.");
    const upload = request.method === "POST" && path.length === 1;
    const max = upload ? config.uploadMaxRequestBytes : 65536;
    const declared = request.headers.get("content-type") ?? "";
    if (request.method === "POST" && !(upload ? declared.startsWith("multipart/form-data;") : declared.split(";")[0] === "application/json"))
        return failure(415, "INVALID_CONTENT_TYPE", "Content-Type tidak sesuai.");
    if (Number(request.headers.get("content-length")) > max)
        return failure(413, "BODY_TOO_LARGE", "Ukuran request terlalu besar.");
    let tooLarge = false;
    const timeout = AbortSignal.timeout(15000);
    const headers = new Headers({ Accept: "application/json" });
    if (request.method === "POST") {
        headers.set("Content-Type", declared);
        headers.set("Origin", config.appOrigin);
    }
    // Never forward admin Cookie, Authorization, report credentials in URL, or arbitrary upstream paths.
    const query = new URLSearchParams();
    if (request.method === "GET")
        for (const key of ["page", "page_size", "report_type_id"]) {
            const value = request.nextUrl.searchParams.get(key);
            if (value !== null)
                query.set(key, value);
        }
    let body: ReadableStream<Uint8Array> | undefined;
    if (request.method === "POST" && request.body) {
        let size = 0;
        body = request.body.pipeThrough(new TransformStream<Uint8Array, Uint8Array>({ transform(chunk, controller) { size += chunk.byteLength; if (size > max) {
                tooLarge = true;
                controller.error(new Error("BODY_TOO_LARGE"));
                return;
            } controller.enqueue(chunk); } }));
    }
    try {
        const init: RequestInit & {
            duplex?: "half";
        } = { method: request.method, headers, body, cache: "no-store", redirect: "error", signal: AbortSignal.any([request.signal, timeout]) };
        if (body)
            init.duplex = "half";
        const upstream = await fetch(`${config.apiBaseURL}/api/v1/public/${path.join("/")}${query.size ? `?${query}` : ""}`, init);
        const responseHeaders = new Headers({ "Content-Type": "application/json", "Cache-Control": "no-store, private", "X-Content-Type-Options": "nosniff" });
        const retry = upstream.headers.get("retry-after");
        if (retry)
            responseHeaders.set("Retry-After", retry);
        if (tooLarge)
            return failure(413, "BODY_TOO_LARGE", "Ukuran request terlalu besar.");
        return new Response(await upstream.text(), { status: upstream.status, headers: responseHeaders });
    }
    catch {
        if (tooLarge)
            return failure(413, "BODY_TOO_LARGE", "Ukuran request terlalu besar.");
        return failure(503, upload ? "SUBMISSION_UNCERTAIN" : "SERVICE_UNAVAILABLE", upload ? "Pengiriman belum dapat dipastikan. Jangan langsung mengirim ulang." : "Layanan belum tersedia.");
    }
}
export { forward as GET, forward as POST };
