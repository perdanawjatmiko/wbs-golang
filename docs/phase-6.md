# Phase 6 — UI/UX overhaul

## Audit and plan

Blueprint and the current README/changelog, Phase 1–5 reports, architecture, RBAC, notification architecture, frontend manifests/tests, App Router, adapters, layouts, components and route files were reviewed. Phase 5 documentation reports incomplete runtime verification; UI changes therefore preserve existing contracts and do not assume Docker/API readiness.

Baseline frontend: Next App Router, strict TypeScript, Tailwind CSS 4, Geist; no icon/component/form/chart library. Existing routes cover public landing/report/tracking, admin login/dashboard, report list/detail, dynamic CRUD, notifications and delivery monitoring. Styling previously relied on scattered zinc utilities, OS dark-mode variants, basic controls, a simple responsive flex admin nav and dense table/form markup. Existing form/API/auth/RBAC/privacy tests were retained.

Plan executed: (A) semantic light theme tokens and local reusable primitives; (B) admin shell, metrics, operational tables/detail, login, CRUD/notifications styling; (C) public portal, reporting/tracking and success state; (D) responsive breakpoints, focus/skip/reduced motion and consistent states; (E) frontend verification, browser spot-check and docs.

## Implemented

- WBS Design System v1 tokens, light palette, Geist typography, semantic control/status styles, responsive grids and reduced motion.
- Enterprise admin shell with grouped permission-derived navigation, desktop collapse, mobile drawer, breadcrumb, profile/logout and notification affordance.
- Dashboard summary cards and status/type distributions, clear report-table route; report list filter panel, status pills, readable table/mobile report cards/pagination; report detail identity, safe text, attachment download action, workflow/history visual hierarchy.
- Existing master CRUD, notification inbox and delivery monitoring use the shared controls/table/form styles without changing API flow.
- Government-service-style public header/footer, concise report/tracking CTAs, explanatory steps, report/tracking page headings, report form/panel and safe success code presentation, accessible status timeline and FAQ panels.
- Login card, accessible password visibility toggle; no new auth flow.
- Mobile shell and two-column layouts stack at the specified breakpoints; list remains server-paginated. No package dependency added.
- Added small reusable icon and status badge components plus UI regression tests.

## Preservation

No backend files, `project_blueprint.md`, database migrations, endpoints, DTOs, permissions, cookie/CSRF behavior, API adapter allowlist, report rules, attachment URLs, notification delivery/SSE behavior, or user-facing business workflow were modified. Admin/report visibility still depends on existing session permissions; frontend navigation remains UX only.

## Verification

- `npm run lint`: PASS (clean after final fix).
- `npx tsc --noEmit`: PASS.
- `npm test`: PASS, 42 passed, 4 opt-in HTTP integration tests skipped due no QA backend/database.
- `npm run build`: PASS; all existing routes compile.
- Browser visual inspection: public landing and login at 375px; source/browser state confirms nav and CTA elements. No screenshot files were saved as artifacts. Protected admin views could not be verified with the local QA API; public form data was unavailable and tracking visual check was interrupted. See [UI regression checklist](ui-regression-checklist.md).

`npm run start` is not used with standalone output; the production build was verified with `next build`. A local dev server was used for the visual spot check.

## Acceptance status

The implementation work is present, but Phase 6 is not fully verified. 25/30 criteria PASS, 5/30 BLOCKED. Tablet/desktop full-route visual checks, admin authenticated visual review, keyboard/screen-reader audit, and full public ready/tracking result visual review remain blocked or incomplete. See the per-criterion status below.

| Group | Result |
|---|---|
| Design system (5) | 5 PASS |
| Admin experience (8) | 8 PASS (implemented; backend-authenticated visual QA not run) |
| Public experience (6) | 6 PASS (implemented; populated backend states not visually inspected) |
| Responsive/accessibility (6) | 1 PASS, 5 BLOCKED |
| Regression/documentation (5) | 5 PASS |

| # | Criterion | Status |
|---:|---|---|
| 1 | Unified design tokens | PASS |
| 2 | Typography/spacing standardized | PASS |
| 3 | Reusable components standardized | PASS |
| 4 | Status colors and labels consistent | PASS |
| 5 | Admin/public visual identity consistent | PASS |
| 6 | Admin layout redesigned | PASS |
| 7 | Dashboard redesigned | PASS |
| 8 | Report list redesigned | PASS |
| 9 | Report detail redesigned | PASS |
| 10 | Master data pages redesigned | PASS |
| 11 | Notification UI redesigned | PASS |
| 12 | Delivery monitoring redesigned | PASS |
| 13 | Login redesigned | PASS |
| 14 | Public landing redesigned | PASS |
| 15 | Public report form redesigned | PASS |
| 16 | Submission success redesigned | PASS |
| 17 | Tracking page redesigned | PASS |
| 18 | FAQ presentation redesigned | PASS |
| 19 | Public error/empty states redesigned | PASS |
| 20 | Mobile layout verified across routes | BLOCKED |
| 21 | Tablet layout verified | BLOCKED |
| 22 | Desktop layout verified at requested widths | BLOCKED |
| 23 | Keyboard navigation verified end-to-end | BLOCKED |
| 24 | Accessibility checks completed | BLOCKED |
| 25 | Loading/error/empty states standardized | PASS |
| 26 | Existing frontend tests pass | PASS |
| 27 | Lint and TypeScript checks pass | PASS |
| 28 | Production build passes | PASS |
| 29 | Existing functional contracts preserved | PASS |
| 30 | Phase 6 docs complete | PASS |

This status distinguishes source/automated evidence from visual review; it does not claim WCAG certification.

## Known limitations

- Admin pages depend on an authenticated Go session and could not be opened through the local frontend QA instance. The unrelated existing Docker UI did not contain these source edits.
- Backend remained unavailable on the frontend development origin; report-type/FAQ loading and populated tracking states were not verified in the browser.
- Visual inspection covered mobile width only for landing and login. UI appears ready for QA, not fully visually signed off.
- Four HTTP integration tests are environment-gated and skipped; they require backend and isolated QA PostgreSQL.
