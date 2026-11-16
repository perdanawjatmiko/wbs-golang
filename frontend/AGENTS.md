# Frontend Architecture Instructions

## Next.js App Router

- Use the Next.js App Router under `src/app/`.
- Route files are named `page.tsx`; do not use `index.tsx` for App Router pages.
- Parenthesized folders such as `(protected)` are route groups and do not appear in public URLs.
- Keep admin CRUD routes explicit and static. Do not add a catch-all or `[resource]` page for roles, permissions, users, report types, FAQs, or email templates.

## Admin CRUD routes

Each master-data resource has its own route file:

```text
src/app/admin/(protected)/users/page.tsx
src/app/admin/(protected)/roles/page.tsx
src/app/admin/(protected)/permissions/page.tsx
src/app/admin/(protected)/report-types/page.tsx
src/app/admin/(protected)/faqs/page.tsx
src/app/admin/(protected)/email-templates/page.tsx
```

Roles and permissions are explicit feature pages: their route files perform the session/authorization check and render `features/admin/roles-management-page.tsx` or `features/admin/permissions-management-page.tsx`. Their fields, tables, validation, and CRUD actions belong in those feature files; do not pass a `Resource` object to a generic renderer for these pages.

The other master-data routes currently share `features/admin/resource-page.tsx` and `features/admin/management.tsx`. Keep those routes static. If a page is later made resource-specific, give it its own feature component and route composition instead of adding route-param lookup or making the page choose a renderer dynamically from a resource object.

Resource definitions in `features/admin/contracts.ts` also drive the permission-filtered admin navigation. When adding or renaming a resource, update its resource definition and explicit page together, and preserve its permission slug and API path.

## Authentication and authorization

- The `(protected)/layout.tsx` requires an admin session and builds navigation from the current user's allowed resources.
- Roles and permissions are restricted to Super Admin as before. Shared master-data pages retain their resource-specific `canManage` check. Preserve these guards when customizing pages.
- Backend RBAC remains the authorization boundary. Frontend navigation and permission checks are only presentation controls.
- Keep CSRF headers and the existing same-origin backend adapter for mutations. Never expose session cookies to client JavaScript.

## API and business behavior

- Do not invent API routes, payload fields, permissions, or validation rules in frontend CRUD pages.
- Preserve server-side pagination, existing CRUD contracts, safe error handling, and data-field filtering.
- UI refactors must not change backend logic, database schema, or authentication/session behavior unless a separately scoped task explicitly requests it.

## Shared UI and verification

- Use existing WBS design tokens and shared shadcn/WBS primitives. Keep business-specific components in their feature folders.
- Add or update frontend tests for route mapping and permission behavior when changing resource routes.
- Run `npm run lint`, `npx tsc --noEmit`, `npm test`, and `npm run build` after route changes. Do not claim environment-gated HTTP integration tests passed when they were skipped.
