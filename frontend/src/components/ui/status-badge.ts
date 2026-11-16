import { createElement } from "react";
import type { ReportStatus } from "@/features/reports/contracts";

const classes: Record<ReportStatus, string> = {
  Open: "status-open",
  "On Progress": "status-progress",
  Done: "status-done",
  Rejected: "status-rejected",
};

export function StatusBadge({ status }: { status: ReportStatus }) {
  return createElement("span", { className: `status-badge ${classes[status]}` }, createElement("span", { className: "sr-only" }, "Status: "), status);
}
