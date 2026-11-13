import { test } from 'node:test';
import assert from 'node:assert/strict';
const base=process.env.TEST_WEB_BASE_URL;
test('HTTP integration: protected page, Go-backed cookie login, CRUD bridge and logout', {skip:!base||!process.env.TEST_ADMIN_EMAIL||!process.env.TEST_ADMIN_PASSWORD}, async()=>{
 const origin=new URL(base).origin;
 let response=await fetch(`${base}/admin/users`,{redirect:'manual'});
 // Next may stream a redirect; in both cases no management data is rendered.
 const unauth=await response.text();
 assert.ok(response.status===307||unauth.includes('/admin/login'));
 assert.ok(!unauth.includes('Tambah data'));
 response=await fetch(`${base}/api/backend/auth/login`,{method:'POST',headers:{Origin:origin,'Content-Type':'application/json'},body:JSON.stringify({email:process.env.TEST_ADMIN_EMAIL,password:process.env.TEST_ADMIN_PASSWORD})});
 assert.equal(response.status,200);
 const rawCookie=response.headers.getSetCookie().find(v=>v.startsWith('wbs_session='));
 assert.ok(rawCookie?.includes('HttpOnly'));
 const cookie=rawCookie.split(';')[0];
 const login=await response.json();assert.equal(login.success,true);assert.ok(login.data.csrf_token);assert.equal(JSON.stringify(login).includes(cookie.split('=')[1]),false);
 const headers={Origin:origin,'Content-Type':'application/json',Cookie:cookie,'X-CSRF-Token':login.data.csrf_token};
 for(const route of ['','users','roles','permissions','report-types','faqs','email-templates']){
  response=await fetch(`${base}/admin/${route}`,{headers:{Cookie:cookie}});assert.equal(response.status,200);assert.ok(!(await response.text()).includes('Administrasi belum tersedia'));
 }
 response=await fetch(`${base}/api/backend/admin/report-types`,{method:'POST',headers:{...headers,'X-CSRF-Token':''},body:JSON.stringify({name:'Blocked'})});assert.equal(response.status,403);
 response=await fetch(`${base}/api/backend/admin/report-types`,{method:'POST',headers,body:JSON.stringify({name:'HTTP integration disposable type'})});assert.equal(response.status,201);const created=await response.json();const id=created.data.id;
 response=await fetch(`${base}/api/backend/admin/report-types/${id}`,{method:'PATCH',headers,body:JSON.stringify({is_active:false})});assert.equal(response.status,200);
 response=await fetch(`${base}/api/backend/admin/report-types/${id}`,{method:'DELETE',headers});assert.equal(response.status,200);
 response=await fetch(`${base}/api/backend/auth/logout`,{method:'POST',headers});assert.equal(response.status,200);assert.ok(response.headers.getSetCookie()[0].includes('Max-Age=0'));
 response=await fetch(`${base}/api/backend/auth/me`,{headers:{Cookie:cookie}});assert.equal(response.status,401);
 response=await fetch(`${base}/api/backend/admin/users`,{headers:{Cookie:cookie}});assert.equal(response.status,401);
});
