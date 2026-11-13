import { requireSession } from "@/features/admin/server";
import { hasPermission } from "@/features/admin/contracts";
import { AdminDashboard } from "@/features/admin-reports/dashboard";
export default async function AdminHome() { const session = await requireSession(); if (!hasPermission(session.user, "report.view"))
    return <section><h1 className="text-3xl font-semibold">Administrasi</h1><p className="mt-4">Dashboard laporan memerlukan permission report.view. Gunakan menu administrasi sesuai hak akses Anda.</p></section>; return <AdminDashboard />; }
