import { notFound } from "next/navigation";
import { requireSession } from "@/features/admin/server";
import { resources, canManage } from "@/features/admin/contracts";
import { Management } from "@/features/admin/management";
export default async function ResourcePage({ params }: {
    params: Promise<{
        resource: string;
    }>;
}) { const { resource } = await params; const definition = resources.find(r => r.path === resource); if (!definition)
    notFound(); const session = await requireSession(); if (!canManage(session.user, definition))
    return <section className="admin-panel permission-state" role="alert"><span className="permission-icon" aria-hidden="true">!</span><div><h1>Akses terbatas</h1><p>Anda tidak memiliki izin untuk mengelola {definition.title.toLowerCase()}.</p></div></section>; return <Management key={definition.path} resource={definition} csrf={session.csrf_token}/>; }
