export class BodyLimitError extends Error {
}
export async function readBoundedText(body: ReadableStream<Uint8Array> | null, limit: number, signal: AbortSignal): Promise<string> {
    if (!body)
        return "";
    const reader = body.getReader();
    const chunks: Uint8Array[] = [];
    let size = 0;
    const cancel = () => { void reader.cancel().catch(() => { }); };
    signal.addEventListener("abort", cancel, { once: true });
    try {
        for (;;) {
            if (signal.aborted)
                throw new Error("Request timeout");
            const part = await reader.read();
            if (signal.aborted)
                throw new Error("Request timeout");
            if (part.done)
                break;
            size += part.value.byteLength;
            if (size > limit) {
                await reader.cancel();
                throw new BodyLimitError("Body too large");
            }
            chunks.push(part.value);
        }
        const bytes = new Uint8Array(size);
        let offset = 0;
        for (const chunk of chunks) {
            bytes.set(chunk, offset);
            offset += chunk.byteLength;
        }
        return new TextDecoder().decode(bytes);
    }
    finally {
        signal.removeEventListener("abort", cancel);
        reader.releaseLock();
    }
}
export function boundedStream(body: ReadableStream<Uint8Array>, max: number): ReadableStream<Uint8Array> { let size = 0; return body.pipeThrough(new TransformStream<Uint8Array, Uint8Array>({ transform(chunk, controller) { size += chunk.byteLength; if (size > max) {
        controller.error(new BodyLimitError("Response too large"));
        return;
    } controller.enqueue(chunk); } })); }
