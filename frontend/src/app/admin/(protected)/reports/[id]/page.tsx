import { notFound } from "next/navigation";
import { requireSession } from "@/features/admin/server";
import { hasPermission } from "@/features/admin/contracts";
import { AdminReportDetail } from "@/features/admin-reports/detail";
export default async function ReportPage({ params }: {
    params: Promise<{
        id: string;
    }>;
}) { const session = await requireSession(); if (!hasPermission(session.user, "report.view"))
    return <p role="alert">Akses laporan tidak diizinkan.</p>; const { id } = await params; if (!/^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i.test(id))
    notFound(); return <AdminReportDetail id={id} csrf={session.csrf_token} canUpdate={hasPermission(session.user, "report.update")}/>; }
