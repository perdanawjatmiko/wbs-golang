import { Management } from "./management";
import { requireSession } from "./server";
import { canManage, type Resource } from "./contracts";

export async function ResourceManagementPage({
  resource,
}: {
  resource: Resource;
}) {
  const session = await requireSession();

  if (!canManage(session.user, resource)) {
    return (
      <section className="admin-panel permission-state" role="alert">
        <span className="permission-icon" aria-hidden="true">
          !
        </span>
        <div>
          <h1>Akses terbatas</h1>
          <p>
            Anda tidak memiliki izin untuk mengelola {resource.title.toLowerCase()}.
          </p>
        </div>
      </section>
    );
  }

  return <Management resource={resource} csrf={session.csrf_token} />;
}
