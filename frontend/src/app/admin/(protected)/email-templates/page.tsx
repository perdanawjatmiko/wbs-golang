import { EmailTemplateList } from "@/features/admin/email-template-list";
import { emailTemplatesResource, canManage } from "@/features/admin/contracts";
import { requireSession } from "@/features/admin/server";

export default async function EmailTemplatesPage() {
  const session = await requireSession();
  if (!canManage(session.user, emailTemplatesResource)) return <section className="admin-panel permission-state" role="alert"><span className="permission-icon" aria-hidden="true">!</span><div><h1>Akses terbatas</h1><p>Anda tidak memiliki izin untuk mengelola email templates.</p></div></section>;
  return <EmailTemplateList />;
}
