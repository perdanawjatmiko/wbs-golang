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

The palette is expressed as CSS custom properties in `src/app/globals.css`; components use semantic classes. Dark-mode variants are opt-in and the current UI stays light. Geist Sans remains the existing font; no new font or component dependency was added. Body copy is 14px in admin and 16px in public content, with responsive title scales and a 4/8px spacing rhythm. Inputs/buttons use 8px radius; panels use 10–12px.

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

No UI library was present. A small local SVG icon component keeps icon stroke/size consistent without adding a package. CSS utility conventions remain compatible with Tailwind 4 and existing components; no broad component-library migration or unrelated state library was added.
