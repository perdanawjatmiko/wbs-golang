import {test} from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import ts from 'typescript';
const root=new URL('../',import.meta.url);
function load(relative){const exports={};const code=ts.transpileModule(fs.readFileSync(new URL(relative,root),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText;vm.runInNewContext(code,{exports});return exports;}
test('frontend framing policy preserves App Router scripts and local HTTP',async()=>{
 const config=load('next.config.ts').default;
 assert.equal(config.output,'standalone');assert.equal(config.poweredByHeader,false);
 const rules=await config.headers();const headers=Object.fromEntries(rules[0].headers.map(h=>[h.key.toLowerCase(),h.value]));
 assert.equal(headers['x-content-type-options'],'nosniff');assert.equal(headers['x-frame-options'],'DENY');
 assert.ok(headers['content-security-policy'].includes("frame-ancestors 'none'"));assert.ok(headers['content-security-policy'].includes("object-src 'none'"));
 assert.ok(!headers['content-security-policy'].includes("script-src 'none'"));assert.ok(!headers['strict-transport-security']);
});
