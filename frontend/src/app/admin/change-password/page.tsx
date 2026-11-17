import { ChangePasswordForm } from "@/features/admin/change-password-form";
import { requireSession } from "@/features/admin/server";

export const dynamic = "force-dynamic";

export default async function ChangePasswordPage() {
  const session = await requireSession();
  return <main className="mx-auto flex min-h-[70vh] max-w-xl items-center px-4 py-12"><section className="w-full space-y-5 rounded-xl border border-slate-200 bg-white p-6"><p className="public-eyebrow">Keamanan akun</p><h1 className="text-2xl font-semibold">Ubah password</h1><p className="text-sm text-slate-600">Password perlu diperbarui sebelum Anda dapat menggunakan aplikasi.</p><ChangePasswordForm csrf={session.csrf_token} /></section></main>;
}
