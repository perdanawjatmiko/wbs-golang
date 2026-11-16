import { ResourceManagementPage } from "@/features/admin/resource-page";
import { emailTemplatesResource } from "@/features/admin/contracts";

export default function EmailTemplatesPage() {
  return <ResourceManagementPage resource={emailTemplatesResource} />;
}
