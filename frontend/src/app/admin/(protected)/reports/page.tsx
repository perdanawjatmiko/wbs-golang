import { requireSession } from "@/features/admin/server";
import { hasPermission } from "@/features/admin/contracts";
import { AdminReportList } from "@/features/admin-reports/list";
export default async function ReportsPage() { const session = await requireSession(); if (!hasPermission(session.user, "report.view"))
    return <section className="admin-panel permission-state" role="alert"><span className="permission-icon" aria-hidden="true">!</span><div><h1>Akses laporan tidak tersedia</h1><p>Akun Anda tidak memiliki permission report.view.</p></div></section>; return <AdminReportList />; }
