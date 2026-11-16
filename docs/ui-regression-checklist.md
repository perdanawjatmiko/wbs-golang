# UI regression checklist

## Verified in this Phase 6 session

- [x] Existing frontend unit/render tests.
- [x] ESLint.
- [x] TypeScript strict typecheck.
- [x] Next.js production build.
- [x] Existing API/client workflow and privacy tests remain unchanged and pass.
- [x] New static design-system/navigation/copy tests pass.
- [x] Browser visual inspection at 375px for public landing and admin login; form and primary actions fit, status/nav remain readable.
- [x] Browser inspection confirmed semantic headings and navigation labels for public landing, report form shell and login.

## Blocked or not completed

- [ ] 768px tablet viewport inspection.
- [ ] 1280px/1440px desktop viewports for all key routes.
- [ ] Admin dashboard/list/detail/master/notification monitoring visual inspection with a QA session.
- [ ] Public form ready/error states with backend master data available; local dev showed loading because backend was unavailable.
- [ ] Tracking result and FAQ populated state; browser inspection was interrupted while opening tracking.
- [ ] Keyboard-only pass across all dialogs/forms/drawer and screen-reader announcement review.
- [ ] Automated axe/WCAG contrast scan; no audit dependency is installed.
- [ ] HTTP frontend/backend integration; existing four HTTP integration tests are opt-in and skipped without QA services.

## Manual reviewer checklist

1. Test 375px, 768px, 1280px and 1440px widths; check page width, long report code/contact, tables, dialogs, sticky actions and footer.
2. Sign in with a QA-only account and verify restricted navigation still matches server session permissions. Test users with report.view, report.update and notifications.manage independently.
3. Keyboard through skip link, drawer, bell popover, forms, file selector, pagination, native confirmation dialog, status notes and retry confirmation. Confirm focus visibility and return after closing dialogs.
4. Verify all statuses include text labels and color is supplementary.
5. Verify public tracking never displays internal notes/contact/admin metadata; never submit real personal data during visual QA.
6. Test 401, 403, 404, 409, 422, 429, 500/network states using safe QA fixtures.
