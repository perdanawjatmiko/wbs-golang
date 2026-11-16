import { RolesManagementPage } from "@/features/admin/roles-management-page";
import { requireSession } from "@/features/admin/server";

export default async function RolesPage() {
  const session = await requireSession();

  if (session.user.role_name !== "Super Admin") {
    return (
      <section className="admin-panel permission-state" role="alert">
        <span className="permission-icon" aria-hidden="true">!</span>
        <div>
          <h1>Akses terbatas</h1>
          <p>Anda tidak memiliki izin untuk mengelola roles.</p>
        </div>
      </section>
    );
  }

  return <RolesManagementPage csrf={session.csrf_token} />;
}
