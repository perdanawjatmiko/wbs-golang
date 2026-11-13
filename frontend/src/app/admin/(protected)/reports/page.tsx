import { requireSession } from "@/features/admin/server";
import { hasPermission } from "@/features/admin/contracts";
import { AdminReportList } from "@/features/admin-reports/list";
export default async function ReportsPage() { const session = await requireSession(); if (!hasPermission(session.user, "report.view"))
    return <p role="alert">Akses laporan tidak diizinkan.</p>; return <AdminReportList />; }
