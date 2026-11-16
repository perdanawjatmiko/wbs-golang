import { test } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../src');
const read = (name) => fs.readFileSync(path.join(root, name), 'utf8');

test('design system defines agreed semantic palette and reduced-motion behavior', () => {
  const css = read('app/globals.css');
  for (const token of ['--primary: #1d4ed8', '--primary-hover: #1e40af', '--navy: #0f172a', '--background: #f8fafc', '--border: #e2e8f0', '--success: #0f766e', '--warning: #b45309', '--danger: #b91c1c']) assert.ok(css.includes(token), token);
  assert.ok(css.includes('prefers-reduced-motion: reduce'));
  assert.ok(css.includes('max-width: 767px'));
});

test('responsive admin and public navigation retain current routes and permission-derived resources', () => {
  const shell = read('components/layout/admin-shell.tsx');
  const layout = read('app/admin/(protected)/layout.tsx');
  assert.ok(shell.includes('/admin/reports'));
  assert.ok(shell.includes('/admin/notification-deliveries'));
  assert.ok(shell.includes('mobileOpen'));
  assert.ok(layout.includes('resources.filter(r => canManage(user, r))'));
  assert.ok(shell.includes('aria-current={active(item.href) ? "page" : undefined}'));
});

test('public copy does not promise anonymity or legal guarantees', () => {
  const home = read('app/(public)/page.tsx');
  assert.ok(home.includes('Buat laporan'));
  assert.ok(home.includes('Lacak laporan'));
  assert.ok(!/anonim|kerahasiaan dijamin|dijamin rahasia/i.test(home));
});

test('shadcn AlertDialog uses WBS tokens and replaces native destructive confirmation', () => {
  const dialog = read('components/ui/alert-dialog.tsx');
  const button = read('components/ui/button.tsx');
  const management = read('features/admin/management.tsx');
  const rolesManagement = read('features/admin/roles-management-page.tsx');
  const permissionsManagement = read('features/admin/permissions-management-page.tsx');
  assert.ok(dialog.includes('@radix-ui/react-alert-dialog'));
  assert.ok(dialog.includes('border-border bg-white'));
  assert.ok(button.includes('min-h-[42px]'));
  assert.ok(button.includes('bg-primary-hover'));
  const deleteDialogs = [management, rolesManagement, permissionsManagement];
  assert.ok(deleteDialogs.every((source) => source.includes('pendingDelete !== null') && source.includes('<AlertDialog')));
  assert.ok(deleteDialogs.every((source) => /AlertDialogAction[\s\S]{0,160}variant="destructive"/.test(source)));
  assert.ok(deleteDialogs.every((source) => !source.includes('window.confirm(')));
});

test('shadcn semantic tokens map to existing WBS palette without changing Geist', () => {
  const css = read('app/globals.css');
  const layout = read('app/layout.tsx');
  assert.ok(css.includes('--color-primary: var(--primary)'));
  assert.ok(css.includes('--color-ring: var(--ring)'));
  assert.ok(css.includes('--primary: #1d4ed8'));
  assert.ok(css.includes('--navy: #0f172a'));
  assert.ok(layout.includes('Geist('));
});

test('all WBS select controls use the shadcn NativeSelect while preserving native semantics', () => {
  const component = read('components/ui/native-select.tsx');
  assert.ok(component.includes('<select'));
  assert.ok(component.includes('React.ComponentProps<"select">'));
  assert.ok(component.includes('function NativeSelectOption'));
  assert.ok(component.includes('relative w-full'));
  assert.ok(component.includes('@/lib/utils'));
  assert.ok(!component.includes('lucide-react'));

  const files = [
    'features/notifications/deliveries.tsx',
    'features/reports/report-form.tsx',
    'features/admin-reports/detail.tsx',
    'features/admin/management.tsx',
    'features/admin-reports/list.tsx',
  ];
  for (const file of files) {
    const source = read(file);
    assert.ok(source.includes('NativeSelect') && source.includes('NativeSelectOption') && source.includes('@/components/ui/native-select'), file);
    assert.ok(!/<select\b|<option\b/.test(source), file);
  }
  const css = read('app/globals.css');
  assert.ok(css.includes('.admin-input[data-slot="native-select"] { padding-right: 40px; }'));
});

test('admin master CRUD routes are explicit pages and keep the shared permission guard', () => {
  const routeRoot = path.join(root, 'app/admin/(protected)');
  for (const route of ['users', 'roles', 'permissions', 'report-types', 'faqs', 'email-templates']) {
    assert.ok(fs.existsSync(path.join(routeRoot, route, 'page.tsx')), route);
  }
  assert.ok(!fs.existsSync(path.join(routeRoot, '[resource]', 'page.tsx')));
  const roles = read('app/admin/(protected)/roles/page.tsx');
  const permissions = read('app/admin/(protected)/permissions/page.tsx');
  assert.ok(roles.includes('requireSession()') && roles.includes('role_name !== "Super Admin"'));
  assert.ok(permissions.includes('requireSession()') && permissions.includes('role_name !== "Super Admin"'));
  assert.ok(roles.includes('RolesManagementPage'));
  assert.ok(permissions.includes('PermissionsManagementPage'));
  assert.ok(read('features/admin/roles-management-page.tsx').includes('admin/roles'));
  assert.ok(read('features/admin/permissions-management-page.tsx').includes('admin/permissions'));
  assert.ok(read('features/admin/resource-page.tsx').includes('canManage(session.user, resource)'));
  const guide = fs.readFileSync(path.resolve(root, '../AGENTS.md'), 'utf8');
  assert.ok(guide.includes('Keep admin CRUD routes explicit and static'));
  assert.ok(guide.includes('Their fields, tables, validation, and CRUD actions belong in those feature files'));
});
