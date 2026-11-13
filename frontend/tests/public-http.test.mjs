import {test} from 'node:test';
import assert from 'node:assert/strict';
const base=process.env.TEST_PUBLIC_WEB_BASE_URL;
test('HTTP public integration: landing, multipart attachment, private tracking and body limits',{skip:!base},async()=>{
 const origin=new URL(base).origin;
 for(const route of ['/','/lapor','/tracking','/development']){const response=await fetch(base+route);assert.equal(response.status,200);}
 let response=await fetch(base+'/api/public/report-types?page_size=100');assert.equal(response.status,200);const types=await response.json();assert.ok(types.data.items.length>0,'dedicated QA database must contain an active report type');
 response=await fetch(base+'/api/public/reports/options');const options=await response.json();assert.ok(options.data.max_file_bytes);
 const contact=`qa-${Date.now()}@example.test`;
 const form=new FormData();for(const [key,value]of Object.entries({name:'HTTP QA Reporter',contact,contact_type:'EMAIL',report_type_id:types.data.items[0].id,detail:'<script>plain report text</script>'}))form.set(key,value);
 // Small valid PNG; backend validates decoded content, not this filename alone.
 const png=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=','base64');
 form.set('attachment',new File([png],'proof.png',{type:'image/png'}));
 response=await fetch(base+'/api/public/reports',{method:'POST',headers:{Origin:origin,Cookie:'wbs_session=unneeded-admin-cookie'},body:form});assert.equal(response.status,201);const created=await response.json();assert.match(created.data.report_code,/^WBS-\d{6}-[A-Z0-9]{2}\d{3,}$/);assert.equal(created.data.status,'Open');assert.equal('id' in created.data,false);assert.equal('attachment_url' in created.data,false);
 const track=async(value)=>fetch(base+'/api/public/reports/track',{method:'POST',headers:{Origin:origin,'Content-Type':'application/json'},body:JSON.stringify({report_code:created.data.report_code,contact:value})});
 response=await track(contact.toUpperCase());assert.equal(response.status,200);assert.ok(response.headers.get('cache-control').includes('no-store'));const result=await response.json();assert.equal(result.data.timeline[0].new_status,'Open');assert.equal(result.data.detail,'<script>plain report text</script>');for(const key of ['internal_note','admin_id','attachment_url','contact'])assert.equal(key in result.data,false);
 response=await track('wrong@example.test');assert.equal(response.status,404);const absent=await response.json();assert.equal(absent.error.code,'REPORT_NOT_FOUND');
 response=await fetch(base+'/api/public/reports/track?contact=leaked',{method:'GET'});assert.equal(response.status,404);
 const oversized=new FormData();oversized.set('attachment',new File([new Uint8Array(options.data.max_request_bytes+1)],'proof.png',{type:'image/png'}));response=await fetch(base+'/api/public/reports',{method:'POST',headers:{Origin:origin},body:oversized});assert.equal(response.status,413);
 const streamed=new Request(base+'/api/public/reports',{method:'POST',body:oversized});response=await fetch(base+'/api/public/reports',{method:'POST',headers:{Origin:origin,'Content-Type':streamed.headers.get('content-type')},body:streamed.body,duplex:'half'});assert.equal(response.status,413);
});
