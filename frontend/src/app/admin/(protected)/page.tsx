import { requireSession } from "@/features/admin/server";
import { hasPermission } from "@/features/admin/contracts";
import { AdminDashboard } from "@/features/admin-reports/dashboard";
export default async function AdminHome() { const session = await requireSession(); if (!hasPermission(session.user, "report.view"))
    return <section className="admin-panel permission-state"><span className="permission-icon" aria-hidden="true">i</span><div><h1>Ruang administrasi</h1><p>Dashboard laporan memerlukan akses laporan. Gunakan menu sesuai hak akses akun Anda.</p></div></section>; return <AdminDashboard />; }
