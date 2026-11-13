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
    return <p role="alert">Akses tidak diizinkan.</p>; return <Management key={definition.path} resource={definition} csrf={session.csrf_token}/>; }
