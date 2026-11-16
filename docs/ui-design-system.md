# WBS Design System v1

Light-first interface system for the existing WBS public service portal and authenticated administration. Business copy, fields, states, permissions, API calls and status values continue to follow current frontend/backend contracts and `project_blueprint.md`.

## Visual foundations

| Token | Value | Use |
|---|---|---|
| `--primary` | `#1D4ED8` | Primary actions, links, focus |
| `--primary-hover` | `#1E40AF` | Hover state |
| `--navy` | `#0F172A` | Admin navigation |
| `--background` | `#F8FAFC` | App canvas |
| `--surface` | `#FFFFFF` | Panels and forms |
| `--border` | `#E2E8F0` | Dividers, controls |
| `--foreground` | `#0F172A` | Primary copy |
| `--muted` | `#64748B` | Supporting copy |
| `--success` | `#0F766E` | Completed state |
| `--warning` | `#B45309` | In-progress state |
| `--danger` | `#B91C1C` | Rejected/error state |

The palette is expressed as CSS custom properties in `src/app/globals.css`; components use semantic classes. Dark-mode variants are opt-in and the current UI stays light. Geist Sans remains the existing font; Phase 6F adds only the dependencies required for the chosen primitives. Body copy is 14px in admin and 16px in public content, with responsive title scales and a 4/8px spacing rhythm. Inputs/buttons use 8px radius; panels use 10–12px.

### shadcn semantic mapping

Phase 6F adds semantic Tailwind v4 theme aliases (`--color-primary`, `--color-border`, `--color-ring`, `--color-card`, `--color-popover`, and related foreground/muted/destructive tokens) that reference the existing WBS custom properties. This mapping does not alter the existing palette or body styles. AlertDialog uses the existing white surface, slate text, border, 12px panel radius and 3px WBS focus outline. Its action buttons retain the 42px WBS control height and 8px radius. No global font, base background, spacing, or responsive selector was replaced.

All application select fields use shadcn `NativeSelect`, which wraps the native HTML `<select>` for browser/mobile behavior. Its wrapper fills the existing form grid; `.admin-input` continues to define WBS height, padding, border, surface and text color. Reserved right padding makes room for the local chevron. Controlled values, labels, change handlers, validation attributes and option values are unchanged.

## Status semantics

Reusable `StatusBadge` maps Open to blue/informational, On Progress to amber, Done to teal/complete, Rejected to red. Visible status text is always present; a small dot and color are supplementary. Status values are not changed.

## Layout

- Public content is centered within a maximum 1120px width, with a 720px form measure and simple header/footer.
- Admin sidebar is 248px wide on desktop and can collapse to 76px. At tablet it uses a narrow icon rail. Below 768px it becomes a 270px drawer with a keyboard-focusable close/backdrop button.
- Admin topbar is sticky, 72px desktop/64px mobile, with page context, notification center, profile and logout.
- Admin content is capped at 1500px with 18–42px responsive padding. Dashboard uses five metric columns on wide desktop, three on tablet and two on mobile.
- Report tables retain a readable desktop data table; mobile switches to per-report cards without fetching a larger dataset.
- Report detail uses a content/metadata split with workflow forms and history below, stacking on mobile.
- Reduced-motion preference disables nonessential transitions/animation.

## Accessibility conventions

Semantic headings, labels, table headers, `aria-current`, button labels, live/error regions, explicit status text, visible focus outline, skip links, keyboard-closeable native dialogs, readable text contrast, and `prefers-reduced-motion` support are included in source. Existing API-derived text remains normal React text nodes; email preview keeps its existing sandbox. Automated WCAG/contrast tooling was not run, so WCAG 2.2 AA conformance is not claimed.

## Technical choices

Before Phase 6F no UI library was present. The UI now uses shadcn/ui as a source-managed primitive layer, currently for Button and AlertDialog only. The official CLI generated the component starting point; WBS token styling and direct Radix package imports were retained to avoid library defaults and an unnecessary all-primitives bundle. A small local SVG icon component remains in use; no icon library or unrelated state library was added. The Tailwind 4 CSS-first setup and existing components remain intact.
