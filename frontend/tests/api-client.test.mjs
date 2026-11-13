// Test the server-only TypeScript boundary without introducing a test framework.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import ts from 'typescript';
import { fileURLToPath } from 'node:url';
const testDirectory = path.dirname(fileURLToPath(import.meta.url));
function load(relative, fetcher, env = {}) {
  const cache = new Map();
  function moduleAt(file) {
    if (cache.has(file)) return cache.get(file);
    const exports = {};
    cache.set(file, exports);
    const code = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
      compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    }).outputText;
    const requireLocal = (name) => {
      if (name === 'server-only') return {};
      if (name.startsWith('@/')) return moduleAt(path.resolve(testDirectory, '../src', name.slice(2) + '.ts'));
      throw new Error('Unexpected import: ' + name);
    };
    vm.runInNewContext(code, { exports, require: requireLocal, fetch: fetcher, URL, AbortSignal, process: { env } }, { filename: file });
    return exports;
  }
  return moduleAt(path.resolve(testDirectory, relative));
}
const validData = { app_name: 'test', environment: 'local', version: 'dev' };
const success = (data) => ({ ok: true, status: 200, json: async () => ({ success: true, data, meta: { request_id: 'test-id' } }) });
test('system client fetches configured origin, no-store, bounded signal and validates metadata', async () => {
  let request;
  const { getSystemInfo } = load('../src/features/system/api.ts', async (url, init) => {
    request = { url, init }; return success(validData);
  }, { API_BASE_URL: 'http://127.0.0.1:18080' });
  const result = await getSystemInfo();
  assert.equal(result.app_name, 'test');
  assert.equal(request.url, 'http://127.0.0.1:18080/api/v1/system/info');
  assert.equal(request.init.cache, 'no-store');
  assert.equal(request.init.redirect, 'error');
  assert.ok(request.init.signal instanceof AbortSignal);
});
for (const [name, fetcher] of [
  ['network failure', async () => { throw new Error('private-host password'); }],
  ['invalid JSON', async () => ({ ok: true, status: 200, json: async () => { throw new Error('private response'); } })],
  ['invalid system payload', async () => success({ app_name: 123 })],
  ['backend error', async () => ({ ok: false, status: 503, json: async () => ({ success: false, error: { code: 'NOT_READY', message: 'private detail' }, meta: { request_id: 'id' } }) })],
]) test(name + ' returns a safe typed error', async () => {
  const { getSystemInfo } = load('../src/features/system/api.ts', fetcher);
  await assert.rejects(getSystemInfo(), (error) => error.name === 'ApiError' && !/private|password/.test(error.message));
});
test('server environment rejects credential-bearing API URL', () => {
  const { serverConfig } = load('../src/lib/config/server.ts', undefined, { API_BASE_URL: 'https://user:password@example.com' });
  assert.throws(serverConfig, /without credentials/);
});
