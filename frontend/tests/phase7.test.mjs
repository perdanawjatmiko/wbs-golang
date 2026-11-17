import { test } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import ts from 'typescript';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const directory = path.dirname(fileURLToPath(import.meta.url));
const nativeRequire = createRequire(import.meta.url);
function load(relative) {
  const file = path.resolve(directory, relative);
  const exports = {};
  const code = ts.transpileModule(fs.readFileSync(file, 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
  vm.runInNewContext(code, { exports, require: nativeRequire }, { filename: file });
  return exports;
}

test('Phase 7 import/export routes are finite and only enabled for supported resources', () => {
  const { bridgePath } = load('../src/features/admin/bridge.ts');
  for (const resource of ['users', 'report-types', 'faqs']) {
    assert.equal(bridgePath(['admin', resource, 'template'], 'GET'), `admin/${resource}/template`);
    assert.equal(bridgePath(['admin', resource, 'export'], 'GET'), `admin/${resource}/export`);
    assert.equal(bridgePath(['admin', resource, 'import', 'preview'], 'POST'), `admin/${resource}/import/preview`);
    assert.equal(bridgePath(['admin', resource, 'import', 'commit'], 'POST'), `admin/${resource}/import/commit`);
    assert.equal(bridgePath(['admin', resource, 'import', 'commit'], 'GET'), null);
  }
  assert.equal(bridgePath(['admin', 'roles', 'import', 'commit'], 'POST'), null);
  assert.equal(bridgePath(['admin', 'email-templates', 'import', 'commit'], 'POST'), null);
});

test('email editor preserves backend placeholders and uses isolated non-delivery preview', () => {
  const source = fs.readFileSync(path.resolve(directory, '../src/features/admin/email-template-editor.tsx'), 'utf8');
  for (const token of ['report_code', 'user_name', 'status', 'external_note']) assert.ok(source.includes(token));
  assert.match(source, /sandbox=""/);
  assert.match(source, /onClick=\{\(\) => void save\(\)\}/);
  assert.doesNotMatch(source, /sendEmail|deliverEmail|smtp/i);
});

test('master-data create/edit interactions use shared Dialog primitives', () => {
  const source = fs.readFileSync(path.resolve(directory, '../src/features/admin/management.tsx'), 'utf8');
  assert.match(source, /<Dialog/);
  assert.match(source, /<DialogContent/);
  assert.match(source, /<AlertDialog/);
});
