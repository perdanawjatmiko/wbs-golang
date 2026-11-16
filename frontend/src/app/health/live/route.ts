export const dynamic = "force-dynamic";
// Process liveness only; backend/database readiness is checked separately.
export function GET() {
    return Response.json({ status: "ok" }, { headers: { "Cache-Control": "no-store" } });
}
