import { ResourceManagementPage } from "@/features/admin/resource-page";
import { reportTypesResource } from "@/features/admin/contracts";

export default function ReportTypesPage() {
  return <ResourceManagementPage resource={reportTypesResource} />;
}
