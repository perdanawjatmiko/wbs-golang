import { PermissionsManagementPage } from "@/features/admin/permissions-management-page";
import { requireSession } from "@/features/admin/server";

export default async function PermissionsPage() {
  const session = await requireSession();

  if (session.user.role_name !== "Super Admin") {
    return (
      <section className="admin-panel permission-state" role="alert">
        <span className="permission-icon" aria-hidden="true">!</span>
        <div>
          <h1>Akses terbatas</h1>
          <p>Anda tidak memiliki izin untuk mengelola permissions.</p>
        </div>
      </section>
    );
  }

  return <PermissionsManagementPage csrf={session.csrf_token} />;
}
