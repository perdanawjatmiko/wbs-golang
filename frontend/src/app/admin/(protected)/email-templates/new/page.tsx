import { EmailTemplateEditor } from "@/features/admin/email-template-editor";
import { emailTemplatesResource, canManage } from "@/features/admin/contracts";
import { requireSession } from "@/features/admin/server";

export default async function NewEmailTemplatePage() {
  const session = await requireSession();
  if (!canManage(session.user, emailTemplatesResource)) return <section className="admin-panel permission-state" role="alert"><span className="permission-icon" aria-hidden="true">!</span><div><h1>Akses terbatas</h1><p>Anda tidak memiliki izin untuk mengelola email templates.</p></div></section>;
  return <EmailTemplateEditor csrf={session.csrf_token} />;
}
