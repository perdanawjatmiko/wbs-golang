# Phase 6F — Incremental shadcn/ui integration

## Baseline audit

- Read the full `project_blueprint.md`, Phase 6 documentation, README/CHANGELOG, and frontend structure before editing.
- Baseline was clean in Git. Frontend is Next.js 16.4.6, React 19.3.0, TypeScript strict, Tailwind CSS 4.3.3, npm (`package-lock.json`), App Router, and `@/*` → `src/*` alias.
- CSS-first Tailwind setup is `src/app/globals.css`; there is no `tailwind.config.*` or `postcss.config.*`. Geist and WBS v1 CSS variables and local controls were already established.
- Existing test runner is Node's built-in `node:test` with source/logic and opt-in HTTP tests. No DOM component test runner or Playwright/Cypress installation exists.
- Existing `window.confirm` in master-data deletion was the clearest benefit for a managed accessible primitive. Forms use labelled native inputs/selects and custom validation. StatusBadge is a business component. NotificationBell is an SSE-connected business popover. These remain local.

## Configuration and installation

`components.json` was added for the current CSS-first Tailwind v4 project and existing aliases. The official shadcn CLI validated the configuration with `npx shadcn@latest add alert-dialog --dry-run`, then generated Button and AlertDialog starting files with `npx shadcn@latest add alert-dialog --yes`. The CLI's all-Radix aggregator and `cn` package were replaced with direct AlertDialog/Slot package imports and the local `@/lib/utils` helper. This avoids loading unrelated Radix primitives and avoids the `cn` package's Node 22 engine floor while the documented app baseline permits Node 20.9+.

Official references: [Next.js installation](https://ui.shadcn.com/docs/installation/next), [components.json configuration](https://ui.shadcn.com/docs/components-json).

Only the following exact frontend dependencies were added:

| Dependency | Purpose |
|---|---|
| `@radix-ui/react-alert-dialog` | Accessible destructive confirmation with modal focus management |
| `@radix-ui/react-slot` | Button `asChild` composition used by AlertDialog action/cancel |
| `class-variance-authority` | Typed Button variants |
| `clsx` | Class composition |
| `tailwind-merge` | Resolve Tailwind class overrides in WBS primitive wrappers |

No icon, animation, chart, form, state, table, or full component-suite dependency was added.

## Token and appearance protection

Existing WBS values remain the source tokens. Tailwind v4 `@theme inline` now maps shadcn semantic utility names to those values: background/foreground, card, popover, primary/secondary, muted, accent, destructive, border, input, ring and WBS danger/primary-hover. Existing root colors, Geist, body background/font, app layout, spacing declarations and breakpoint rules were not replaced. Button uses the WBS 42px height, 8px radius, colors and 3px focus outline. AlertDialog uses WBS border/surface/text and 12px panel radius.

Only master-data deletion now uses AlertDialog; DELETE URL, request method, CSRF header, request payload, error mapping, and post-success refresh are unchanged. Dialog action closes on confirmation as the browser-native confirmation did before the asynchronous DELETE completes. Notification retry's existing native dialog was deliberately retained because its current flow remains open while the provider retry is pending and has specific duplicate-delivery warning behavior.

Button and AlertDialog live under `components/ui/`. WBS business components and their props/data contracts stay in existing feature folders. Native form controls, StatusBadge, cards/tables, navigation, notifications and all public routes were intentionally retained.

## Verification

- `npx shadcn@latest add alert-dialog --dry-run`: PASS; files/dependencies inspected before applying.
- `npm run lint`: PASS.
- `npx tsc --noEmit`: PASS.
- `npm test`: PASS, 44 passed, 4 environment-gated HTTP integration tests skipped (requires QA backend/database).
- `npm run build`: PASS on Next.js 16.4.6/Tailwind 4.3.3.
- `npm audit --omit=dev`: PASS, 0 production dependency advisories. Full `npm audit` reports 7 moderate advisories in the pre-existing TypeScript ESLint/minimatch development toolchain (those packages are present in the baseline lockfile); no unrelated lint-tool upgrade was made.
- Focused node:test assertions cover WBS token aliases, primitive styling, controlled AlertDialog integration, DELETE confirmation primitive usage and preserved Geist. They inspect source contracts; they do not simulate key presses.
- Public landing page rendered in the current local browser; its accessible tree exposed the skip link, named public navigation, heading hierarchy and existing links. The FAQ reached its existing empty state. Reporting form and tracking pages were not browser-checked in this session.

## Visual/accessibility limits

No baseline screenshot artifacts existed. A source snapshot of `HEAD` was prepared for baseline viewing, but its temporary `node_modules` symlink was outside the Turbopack workspace root, so the snapshot server returned an internal error. The current browser tool exposed the accessibility tree but no screenshot/viewport capture API. Protected routes also require an authorized QA session that was not available. Therefore matching before/after screenshots at 375/768/1280/1440, all-route visual comparison, and keyboard/screen-reader/automated WCAG checks are **BLOCKED**. No pixel-perfect or WCAG 2.2 AA claim is made.

## Acceptance criteria

| # | Criterion | Result | Evidence / reason |
|---:|---|---|---|
| 1 | Existing design tokens preserved | PASS | Original root token values retained; CSS diff adds aliases only |
| 2 | shadcn/ui configured | PASS | `components.json` and successful CLI dry-run/add |
| 3 | Tailwind compatibility verified | PASS | Tailwind 4 lint/type/build pass |
| 4 | No destructive config overwrite | PASS | No `next.config`, tsconfig, font or existing component overwrite |
| 5 | Fonts/global styles preserved | PASS | Geist and body rules unchanged |
| 6 | Interaction primitives evaluated | PASS | AlertDialog adopted for native deletion confirm; custom notification remains |
| 7 | Form primitives evaluated | PASS | Existing native controls retained, API/form architecture unchanged |
| 8 | Presentation primitives evaluated | PASS | Business StatusBadge, CSS cards and server table retained |
| 9 | Business components preserved | PASS | No wholesale replacement or data mapping change |
| 10 | Existing component contracts preserved | PASS | Only internal delete confirmation changed |
| 11 | Admin visual appearance preserved | BLOCKED | No authorized before/after admin screenshot comparison |
| 12 | Public visual appearance preserved | BLOCKED | No matching screenshot comparison |
| 13 | Mobile appearance verified | BLOCKED | No viewport capture/comparison across routes |
| 14 | Tablet appearance verified | BLOCKED | No viewport capture/comparison |
| 15 | Desktop appearance verified | BLOCKED | No viewport capture/comparison |
| 16 | Authentication behavior preserved | PASS | Auth code untouched; existing regression tests pass |
| 17 | RBAC behavior preserved | PASS | RBAC code untouched; existing permission tests pass |
| 18 | Reporting flow preserved | PASS | Report components/API untouched; existing tests pass |
| 19 | Tracking flow preserved | PASS | Tracking components/API untouched; existing tests pass |
| 20 | Notification behavior preserved | PASS | Bell/SSE/retry code untouched; regression tests pass |
| 21 | Frontend lint | PASS | `npm run lint` |
| 22 | TypeScript checks | PASS | `npx tsc --noEmit` |
| 23 | Frontend tests | PASS | 44 pass, 4 opt-in integration tests skipped |
| 24 | Production build | PASS | `npm run build` |
| 25 | Accessibility checks completed | BLOCKED | Radix primitive used; key/focus and WCAG audit not run |

**Result: 19 PASS, 6 BLOCKED.** Integration is ready for continued component adoption, but visual preservation and accessibility sign-off remain outstanding.

## Known limitations and next step

Only AlertDialog/Button are integrated so far. Tooltip, Popover, DropdownMenu, Dialog, Tabs and presentation primitives were evaluated and retained where there was no clear need or their business-specific behavior made migration risky. The current lint/test setup does not mount React components in a DOM, so it cannot verify keyboard/focus behavior. Next step should be a QA session with baseline/current screenshots and an authenticated non-production test user, followed by keyboard testing of AlertDialog. Do not migrate notification/popover or form components until that comparison and targeted interaction test can run.
