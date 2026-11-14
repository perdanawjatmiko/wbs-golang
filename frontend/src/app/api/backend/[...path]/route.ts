import { NextRequest } from "next/server";
import { serverConfig } from "@/lib/config/server";
import { bridgePath } from "@/features/admin/bridge";
import { boundedStream, readBoundedText, BodyLimitError } from "@/features/admin/stream";
export const dynamic = "force-dynamic";
function failure(status: number, message: string) { return Response.json({ success: false, error: { code: "BRIDGE_ERROR", message }, meta: { request_id: "frontend" } }, { status, headers: { "Cache-Control": "no-store, private", "X-Content-Type-Options": "nosniff" } }); }
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
    const isSSE = route === "admin/notifications/stream";
    const signal = isSSE ? request.signal : AbortSignal.any([request.signal, AbortSignal.timeout(15000)]);
    const headers = new Headers({ Accept: "application/json" });
    const token = request.cookies.get("wbs_session")?.value;
    if (token && /^[a-f0-9]{64}$/.test(token))
        headers.set("Cookie", `wbs_session=${token}`);
    if (request.method !== "GET") {
        headers.set("Content-Type", "application/json");
        headers.set("Origin", config.appOrigin);
        headers.set("X-CSRF-Token", request.headers.get("x-csrf-token") ?? "");
    }
    if(isSSE){
        const cursor=request.headers.get("last-event-id");
        if(cursor && !/^[0-9]{1,19}$/.test(cursor)) return failure(422,"Cursor tidak valid.");
        if(cursor) headers.set("Last-Event-ID",cursor);
        headers.set("Accept","text/event-stream");
        return forwardSSE(request,config.apiBaseURL,headers);
    }
    try {
        let body: string | undefined;
        if (["POST", "PATCH"].includes(request.method))
            body = await readBoundedText(request.body, 1048576, signal);
        const query = new URLSearchParams();
        for (const key of ["page", "page_size", "per_page", "search", "is_active", "report_type_id", "status", "date_from", "date_to", "sort", "direction", "recent_limit", "history_page", "history_per_page", "channel"]) {
            const value = request.nextUrl.searchParams.get(key);
            if (value !== null)
                query.set(key, value);
        }
        const upstream = await fetch(`${config.apiBaseURL}/api/v1/${route}${query.size ? `?${query}` : ""}`, { method: request.method, headers, body, cache: "no-store", redirect: "error", signal });
        const responseHeaders = new Headers({ "Content-Type": "application/json", "Cache-Control": "no-store, private", "X-Content-Type-Options": "nosniff" });
        for (const cookie of upstream.headers.getSetCookie())
            if (cookie.startsWith("wbs_session="))
                responseHeaders.append("Set-Cookie", cookie);
        if (path[0] === "admin" && path[1] === "reports" && path.length === 4 && path[3] === "attachment" && upstream.ok) {
            const kind = upstream.headers.get("content-type") ?? "";
            const ext: Record<string, string> = { "image/jpeg": "jpg", "image/png": "png", "application/pdf": "pdf" };
            const length = Number(upstream.headers.get("content-length"));
            if (!ext[kind] || !upstream.body || !Number.isInteger(length) || length < 1 || length > config.uploadMaxRequestBytes) {
                await upstream.body?.cancel();
                return failure(502, "Respons attachment tidak valid.");
            }
            responseHeaders.set("Content-Type", kind);
            responseHeaders.set("Content-Length", String(length));
            responseHeaders.set("Content-Disposition", `attachment; filename="report-${path[2]}.${ext[kind]}"`);
            return new Response(boundedStream(upstream.body, length), { status: 200, headers: responseHeaders });
        }
        let text: string;
        try {
            text = await readBoundedText(upstream.body, 16 * 1024 * 1024, signal);
        }
        catch (error) {
            if (error instanceof BodyLimitError)
                return failure(502, "Respons layanan melebihi batas.");
            throw error;
        }
        return new Response(text, { status: upstream.status, headers: responseHeaders });
    }
    catch (error) {
        if (error instanceof BodyLimitError)
            return failure(413, "Ukuran request atau response terlalu besar.");
        return failure(503, "Layanan belum tersedia atau request belum dapat dipastikan. Muat ulang sebelum mengirim kembali perubahan.");
    }
}
async function forwardSSE(request:NextRequest,base:string,headers:Headers){
    const controller=new AbortController();
    const abort=()=>controller.abort();
    request.signal.addEventListener("abort",abort,{once:true});
    const timer=setTimeout(abort,10000);
    try{
        const upstream=await fetch(`${base}/api/v1/admin/notifications/stream`,{headers,cache:"no-store",redirect:"error",signal:controller.signal});
        clearTimeout(timer);
        if(!upstream.ok || !upstream.body || !upstream.headers.get("content-type")?.startsWith("text/event-stream")){
            await upstream.body?.cancel();controller.abort();request.signal.removeEventListener("abort",abort);
            return failure(upstream.status===401?401:upstream.status===403?403:503,"Realtime tidak tersedia. Muat ulang sesi.");
        }
        const reader=upstream.body.getReader();
        const body=new ReadableStream<Uint8Array>({
            async pull(output){try{const next=await reader.read();if(next.done){output.close();controller.abort();request.signal.removeEventListener("abort",abort);}else if(next.value.byteLength>65536){throw new Error("SSE frame too large");}else output.enqueue(next.value);}catch{controller.abort();output.error(new Error("Realtime disconnected"));request.signal.removeEventListener("abort",abort);}},
            async cancel(){controller.abort();request.signal.removeEventListener("abort",abort);await reader.cancel().catch(()=>{});}
        });
        return new Response(body,{headers:{"Content-Type":"text/event-stream","Cache-Control":"no-cache, no-store, private","X-Accel-Buffering":"no","X-Content-Type-Options":"nosniff"}});
    }catch{clearTimeout(timer);controller.abort();request.signal.removeEventListener("abort",abort);return failure(503,"Realtime tidak tersedia.");}
}
export { forward as GET, forward as POST, forward as PATCH, forward as DELETE };
