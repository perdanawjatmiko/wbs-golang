import { ResourceManagementPage } from "@/features/admin/resource-page";
import { usersResource } from "@/features/admin/contracts";

export default function UsersPage() {
  return <ResourceManagementPage resource={usersResource} />;
}
