import { requireSession } from "@/features/admin/server";
export default async function AdminHome() { await requireSession(); return <><h1 className="text-3xl font-semibold">Administrasi</h1><p className="mt-4 text-zinc-600 dark:text-zinc-400">Kelola akun, hak akses, dan master data melalui navigasi sesuai permission Anda.</p></>; }
