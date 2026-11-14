import {NotificationBell} from "@/features/notifications/bell";
import Link from "next/link";
import { requireSession } from "@/features/admin/server";
import { resources, canManage, hasPermission } from "@/features/admin/contracts";
import { LogoutButton } from "@/features/admin/logout-button";
export const dynamic = "force-dynamic";
export default async function AdminLayout({ children }: {
    children: React.ReactNode;
}) { const { user, csrf_token } = await requireSession(); return <div className="flex min-h-screen flex-col md:flex-row"><aside className="border-b border-zinc-200 p-6 md:w-64 md:border-r dark:border-zinc-800"><Link href="/admin" className="text-xl font-bold">WBS Admin</Link><nav aria-label="Administrasi" className="mt-6 flex flex-wrap gap-2 md:flex-col"><Link className="admin-nav" href="/admin">Beranda</Link>{hasPermission(user, "report.view") && <Link className="admin-nav" href="/admin/reports">Laporan</Link>}{hasPermission(user,"notifications.manage") && <Link className="admin-nav" href="/admin/notification-deliveries">Pengiriman notifikasi</Link>}{resources.filter(r => canManage(user, r)).map(r => <Link className="admin-nav" key={r.path} href={`/admin/${r.path}`}>{r.title}</Link>)}</nav></aside><div className="min-w-0 flex-1"><header className="flex flex-wrap items-center justify-between gap-4 border-b border-zinc-200 px-6 py-4 dark:border-zinc-800"><div><p className="font-medium">{user.name}</p><p className="text-sm text-zinc-500">{user.role_name}</p></div><div className="flex items-center gap-3">{hasPermission(user,"report.view") && <NotificationBell csrf={csrf_token}/>}<LogoutButton csrf={csrf_token}/></div></header><main className="p-6 lg:p-10">{children}</main></div></div>; }
