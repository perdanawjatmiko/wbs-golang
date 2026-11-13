import { test } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import ts from 'typescript';
import { fileURLToPath } from 'node:url';
const directory=path.dirname(fileURLToPath(import.meta.url));
function load(relative, mocks={}) {
 const cache=new Map();
 function moduleAt(file){if(cache.has(file))return cache.get(file);const exports={};cache.set(file,exports);
 const code=ts.transpileModule(fs.readFileSync(file,'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText;
 function localRequire(name){if(name==='server-only')return {};if(mocks[name])return mocks[name];if(name.startsWith('@/'))return moduleAt(path.resolve(directory,'../src',name.slice(2)+'.ts'));if(name.startsWith('.'))return moduleAt(path.resolve(path.dirname(file),name+'.ts'));throw new Error('Unexpected import '+name);}
 vm.runInNewContext(code,{exports,require:localRequire,fetch:mocks.fetch,AbortSignal,URL,TextEncoder,process:{env:mocks.env??{}}},{filename:file});return exports;}
 return moduleAt(path.resolve(directory,relative));
}
const contracts=load('../src/features/admin/contracts.ts');
const user={id:'id',name:'Test',email:'test@example.test',role_id:'role',role_name:'Investigator',permissions:[]};
test('login form validates credentials',()=>{assert.ok(contracts.validateLogin('bad',''));assert.ok(contracts.validateLogin('test@example.test',''));assert.equal(contracts.validateLogin('test@example.test','secure-password-123'),null);});
test('permission-aware navigation defaults deny and restricts security administration',()=>{for(const r of contracts.resources)assert.equal(contracts.canManage(user,r),false);const master={...user,permissions:['master.manage','users.manage']};for(const r of contracts.resources)assert.equal(contracts.canManage(master,r),!r.security);for(const r of contracts.resources)assert.equal(contracts.canManage({...user,role_name:'Super Admin'},r),true);});
test('CRUD forms validate required data, password, email, order and placeholders',()=>{const get=p=>contracts.resources.find(r=>r.path===p);assert.ok(contracts.validateForm(get('roles'),{name:''},false));assert.ok(contracts.validateForm(get('users'),{name:'User',email:'bad',password:'short',role_id:'a'},false));assert.ok(contracts.validateForm(get('faqs'),{question:'Q',answer:'A',sort_order:-1},false));assert.ok(contracts.validateForm(get('email-templates'),{event_key:'EVENT',subject:'{{unknown}}',body_html:'Body'},false));assert.equal(contracts.validateForm(get('report-types'),{name:'Type',is_active:true},false),null);});
test('protected route redirects before fetching when unauthenticated',async()=>{let fetched=false;const server=load('../src/features/admin/server.ts',{'next/headers':{cookies:async()=>({get:()=>undefined})},'next/navigation':{redirect:to=>{throw new Error('REDIRECT '+to);}},fetch:async()=>{fetched=true;}});await assert.rejects(server.requireSession(),/REDIRECT \/admin\/login/);assert.equal(fetched,false);});
test('expired session redirects and forwards only the session cookie',async()=>{let init;const server=load('../src/features/admin/server.ts',{'next/headers':{cookies:async()=>({get:()=>({value:'a'.repeat(64)})})},'next/navigation':{redirect:to=>{throw new Error('REDIRECT '+to);}},fetch:async(_url,options)=>{init=options;return {status:401};}});await assert.rejects(server.requireSession(),/REDIRECT/);assert.equal(init.headers.Cookie,'wbs_session='+'a'.repeat(64));assert.equal(init.cache,'no-store');});
test('browser API handles session expiry, access denial and network errors safely',async()=>{for(const status of [401,403]){const client=load('../src/features/admin/client.ts',{fetch:async()=>({ok:false,status,json:async()=>({success:false,error:{message:'private internal detail'}})})});await assert.rejects(client.adminRequest('auth/me'),err=>err.status===status&&!err.message.includes('private'));}const client=load('../src/features/admin/client.ts',{fetch:async()=>{throw new Error('secret connection string');}});await assert.rejects(client.adminRequest('auth/me'),err=>err.status===0&&!err.message.includes('secret'));});
test('bridge only allows Phase 1 routes',()=>{const {bridgePath}=load('../src/features/admin/bridge.ts');assert.equal(bridgePath(['auth','login'],'POST'),'auth/login');assert.equal(bridgePath(['admin','users'],'GET'),'admin/users');for(const p of [['https:','evil.test'],['admin','reports','invalid'],['admin','users','..'],['auth','login','extra']])assert.equal(bridgePath(p,'GET'),null);});
