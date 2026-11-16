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
