import { requireSession } from "@/features/admin/server";
import { resources, canManage } from "@/features/admin/contracts";
import { AdminShell } from "@/components/layout/admin-shell";
import { redirect } from "next/navigation";
export const dynamic = "force-dynamic";
export default async function AdminLayout({ children }: {
    children: React.ReactNode;
}) { const { user, csrf_token } = await requireSession(); if (user.must_change_password) redirect("/admin/change-password"); return <AdminShell user={user} csrf={csrf_token} resources={resources.filter(r => canManage(user, r))}>{children}</AdminShell>; }
