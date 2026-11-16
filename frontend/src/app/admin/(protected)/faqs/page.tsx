import { ResourceManagementPage } from "@/features/admin/resource-page";
import { faqsResource } from "@/features/admin/contracts";

export default function FAQsPage() {
  return <ResourceManagementPage resource={faqsResource} />;
}
