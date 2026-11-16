# UI component inventory

## Shared UI

| Component/pattern | Location | Usage |
|---|---|---|
| Button | `frontend/src/components/ui/button.tsx` | shadcn/CVA button primitive; WBS dimensions, colors, radius, focus and disabled styling are retained |
| AlertDialog | `frontend/src/components/ui/alert-dialog.tsx` | Radix-backed, accessible destructive confirmation used by master-data deletion |
| NativeSelect | `frontend/src/components/ui/native-select.tsx` | shadcn-styled native HTML select used by all report, admin workflow, CRUD and delivery filters |
| `cn` utility | `frontend/src/lib/utils.ts` | `clsx` + `tailwind-merge` for predictable primitive class composition |
| Icon | `src/components/ui/icon.tsx` | Small inline SVG icon set with decorative SVG hidden from assistive tech |
| StatusBadge | `src/components/ui/status-badge.ts` | Shared text-plus-color report status indicator |
| AdminShell | `src/components/layout/admin-shell.tsx` | Permission-derived navigation, collapsible rail/mobile drawer, breadcrumb, bell, profile, logout |
| Button/input/textarea/select | Global `.admin-button`, `.admin-button-secondary`, `.admin-input` | Shared dimensions, focus, disabled and error states |
| Metric card / filter panel / table / pagination | `src/app/globals.css`, report shared components | Consistent dashboard and operational data patterns |
| Empty/error/loading states | `features/admin-reports/shared.tsx` and public form state | Accessible explanations and retry where existing operation allows it |
| FAQ accordion | `features/reports/faq.tsx` | Native `<details>`/`<summary>`, existing data source/filtering |
| Timeline | report detail and tracking form | Status and notes presented chronologically using existing DTOs |

## Screens and contracts

- Public `/`: service introduction, submission/tracking CTAs, usage steps and existing active FAQ source.
- Public `/lapor`: existing report form, contact selector, active report types, upload policy, FAQ, submission confirmation/code/copy/tracking link.
- Public `/tracking`: existing report code+contact POST, public report summary, status and external-only timeline.
- Admin `/admin/login`: existing Go-backed login and cookie/CSRF flow, password visibility toggle.
- Admin `/admin`: existing dashboard API metrics, status/type distributions and bounded recent reports.
- Admin `/admin/reports`, `/admin/reports/[id]`: existing filtering/pagination, attachment endpoint, workflow/notes, concurrency feedback, history.
- Existing dynamic master resource route: users, roles, permissions, report types, FAQs, templates.
- Existing notification bell/inbox/SSE and delivery monitoring/retry remain on current adapters and permissions.

No API routes, permission policy, data fields, workflows, report content, or backend code were added by Phase 6.

## shadcn/ui layer (Phase 6F)

`components.json` configures the official CLI for the existing Tailwind v4 CSS file, Geist-based app, `@/*` aliases, CSS variables, and the Radix-backed New York component source. AlertDialog/Button are used for destructive confirmation; NativeSelect wraps a real HTML `<select>` and is used for every select control. Text inputs, textareas, checkboxes, WBS `StatusBadge`, notification popover, data tables, cards, and business forms remain local because they are already contract-specific or have no clear migration benefit. Do not replace these wholesale.
