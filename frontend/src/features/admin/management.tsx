"use client";
import Link from "next/link";
import { useEffect, useState, type FormEvent } from "react";
import { adminRequest, AdminApiError } from "./client";
import { validateForm, type FieldValue, type PageData, type Resource, type Row } from "./contracts";
function initialValues(resource: Resource, row?: Row): Record<string, FieldValue> {
    return Object.fromEntries(resource.fields.map(field => [field.name, field.kind === "password" ? "" : row?.[field.name] ?? (field.kind === "boolean" ? true : field.kind === "number" ? 0 : field.kind === "permissions" ? [] : field.kind === "report-type" ? null : "")]));
}
async function options(path: string): Promise<Row[]> {
    const items: Row[] = [];
    for (let page = 1; page <= 100; page++) {
        const result = await adminRequest<PageData>(`${path}?page_size=100&page=${page}`);
        items.push(...result.items);
        if (items.length >= result.total)
            break;
    }
    return items;
}
export function Management({ resource, csrf }: {
    resource: Resource;
    csrf: string;
}) {
    const [data, setData] = useState<PageData>({ items: [], total: 0, page: 1, page_size: 20 });
    const [page, setPage] = useState(1);
    const [search, setSearch] = useState("");
    const [query, setQuery] = useState("");
    const [active, setActive] = useState("");
    const [revision, setRevision] = useState(0);
    const [busy, setBusy] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");
    const [feedback, setFeedback] = useState("");
    const [expired, setExpired] = useState(false);
    const [editing, setEditing] = useState<Row | null | undefined>(undefined);
    const [values, setValues] = useState<Record<string, FieldValue>>({});
    const [choices, setChoices] = useState<Row[]>([]);
    const [optionsError, setOptionsError] = useState("");
    useEffect(() => { let ignore = false; adminRequest<PageData>(`admin/${resource.path}?page=${page}&page_size=20&search=${encodeURIComponent(query)}&is_active=${active}`).then(result => { if (!ignore) {
        setData(result);
        setError("");
        setBusy(false);
    } }, err => { if (!ignore) {
        setData({ items: [], total: 0, page, page_size: 20 });
        setError(err.message);
        setExpired(err instanceof AdminApiError && err.status === 401);
        setBusy(false);
    } }); return () => { ignore = true; }; }, [resource.path, page, query, active, revision]);
    useEffect(() => { let ignore = false; const path = resource.path === "users" ? "admin/roles" : resource.path === "roles" ? "admin/permissions" : resource.path === "faqs" ? "public/report-types" : null; if (path)
        options(path).then(rows => { if (!ignore)
            setChoices(rows); }, err => { if (!ignore)
            setOptionsError(err.message); }); return () => { ignore = true; }; }, [resource.path]);
    function open(row: Row | null) { setEditing(row); setValues(initialValues(resource, row ?? undefined)); setError(""); setFeedback(""); }
    function reload() { setBusy(true); setRevision(n => n + 1); }
    async function save(event: FormEvent<HTMLFormElement>) { event.preventDefault(); const invalid = validateForm(resource, values, !!editing); if (invalid) {
        setError(invalid);
        return;
    } setSaving(true); setError(""); const payload = { ...values }; if (editing && !payload.password)
        delete payload.password; try {
        await adminRequest(`admin/${resource.path}${editing ? `/${editing.id}` : ""}`, { method: editing ? "PATCH" : "POST", headers: { "X-CSRF-Token": csrf }, body: JSON.stringify(payload) });
        setEditing(undefined);
        setFeedback("Data berhasil disimpan.");
        reload();
    }
    catch (err) {
        setError(err instanceof Error ? err.message : "Gagal menyimpan.");
        if (err instanceof AdminApiError && err.status === 401) {
            setExpired(true);
            setData({ ...data, items: [] });
            setEditing(undefined);
        }
    }
    finally {
        setSaving(false);
    } }
    async function remove(row: Row) { if (!window.confirm("Hapus data ini? Data yang masih direferensikan tidak dapat dihapus; gunakan nonaktifkan bila tersedia."))
        return; setSaving(true); setError(""); try {
        await adminRequest(`admin/${resource.path}/${row.id}`, { method: "DELETE", headers: { "X-CSRF-Token": csrf } });
        setFeedback("Data berhasil dihapus.");
        reload();
    }
    catch (err) {
        setError(err instanceof Error ? err.message : "Gagal menghapus.");
        if (err instanceof AdminApiError && err.status === 401) {
            setExpired(true);
            setData({ ...data, items: [] });
        }
    }
    finally {
        setSaving(false);
    } }
    if (expired)
        return <section><h1 className="text-2xl font-semibold">Sesi berakhir</h1><p role="alert" className="my-4">Silakan login kembali untuk melanjutkan.</p><Link className="admin-button" href="/admin/login">Login kembali</Link></section>;
    return <section className="space-y-6"><div className="flex flex-wrap items-center justify-between gap-4"><h1 className="text-3xl font-semibold">{resource.title}</h1><button className="admin-button" onClick={() => open(null)} disabled={saving}>Tambah data</button></div>
 {feedback && <p role="status" className="text-emerald-700 dark:text-emerald-400">{feedback}</p>}{error && <div role="alert" className="text-red-700 dark:text-red-400">{error} <button className="underline" onClick={reload}>Coba lagi</button></div>}
 <form className="flex flex-wrap items-end gap-3" onSubmit={e => { e.preventDefault(); setBusy(true); setQuery(search); setPage(1); setRevision(n => n + 1); }}><label className="min-w-48 flex-1">Cari<input className="admin-input" value={search} maxLength={200} onChange={e => setSearch(e.target.value)}/></label>{resource.fields.some(f => f.name === "is_active") && <label>Status<select className="admin-input" value={active} onChange={e => { setBusy(true); setActive(e.target.value); setPage(1); }}><option value="">Semua</option><option value="true">Aktif</option><option value="false">Nonaktif</option></select></label>}<button className="admin-button-secondary">Cari</button></form>
 {busy ? <p role="status">Memuat data…</p> : <><div className="overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800"><table className="w-full text-left text-sm"><caption className="sr-only">Daftar {resource.title}</caption><thead><tr>{resource.columns.map(column => <th className="p-4" key={column} scope="col">{resource.fields.find(f => f.name === column)?.label ?? column}</th>)}<th scope="col" className="p-4">Tindakan</th></tr></thead><tbody>{data.items.map(row => <tr className="border-t border-zinc-200 dark:border-zinc-800" key={row.id}>{resource.columns.map(column => <td key={column} className="max-w-sm truncate p-4">{typeof row[column] === "boolean" ? (row[column] ? "Aktif" : "Nonaktif") : String(row[column] ?? "—")}</td>)}<td className="whitespace-nowrap p-4"><button className="mr-4 underline" disabled={saving} onClick={() => open(row)}>Edit</button><button className="text-red-700 underline dark:text-red-400" disabled={saving} onClick={() => remove(row)}>Hapus</button></td></tr>)}</tbody></table>{data.items.length === 0 && !error && <p className="p-6">Belum ada data yang cocok.</p>}</div><nav aria-label="Pagination" className="flex items-center gap-4"><button className="admin-button-secondary" disabled={page <= 1} onClick={() => { setBusy(true); setPage(n => n - 1); }}>Sebelumnya</button><span>Halaman {page} · {data.total} data</span><button className="admin-button-secondary" disabled={page * 20 >= data.total} onClick={() => { setBusy(true); setPage(n => n + 1); }}>Berikutnya</button></nav></>}
 {editing !== undefined && <form onSubmit={save} className="space-y-5 rounded-xl border border-zinc-300 p-6 dark:border-zinc-700" aria-label={editing ? "Edit data" : "Tambah data"}><h2 className="text-xl font-semibold">{editing ? "Edit" : "Tambah"} {resource.title}</h2>{optionsError && <p role="alert">Pilihan referensi gagal dimuat: {optionsError}. Muat ulang halaman.</p>}
 {resource.fields.map(field => {
                const id = `field-${field.name}`;
                const value = values[field.name];
                const update = (v: FieldValue) => setValues(prev => ({ ...prev, [field.name]: v }));
                if (field.kind === "permissions")
                    return <fieldset key={id} className="space-y-2"><legend className="mb-3 font-medium">{field.label}</legend>{choices.map(choice => <label key={choice.id} className="block"><input type="checkbox" className="mr-3" checked={Array.isArray(value) && value.includes(choice.id)} onChange={e => { const current = Array.isArray(value) ? value : []; update(e.target.checked ? [...current, choice.id] : current.filter(v => v !== choice.id)); }}/>{String(choice.slug)}</label>)}</fieldset>;
                if (field.kind === "boolean")
                    return <label key={id} className="block"><input type="checkbox" className="mr-3" checked={value === true} onChange={e => update(e.target.checked)}/>{field.label}</label>;
                if (field.kind === "role" || field.kind === "report-type")
                    return <label key={id} className="block" htmlFor={id}>{field.label}<select id={id} className="admin-input" required={field.required} value={String(value ?? "")} onChange={e => update(e.target.value || null)}><option value="">{field.kind === "role" ? "Pilih role" : "FAQ umum"}</option>{value && !choices.some(c => c.id === value) && <option value={String(value)}>{String(value)} (referensi existing)</option>}{choices.map(choice => <option value={choice.id} key={choice.id}>{String(choice.name)}</option>)}</select></label>;
                return <label key={id} htmlFor={id} className="block">{field.label}{field.kind === "textarea" ? <textarea id={id} rows={field.name === "body_html" ? 10 : 4} className="admin-input" required={field.required} maxLength={field.max} value={String(value ?? "")} onChange={e => update(e.target.value)}/> : <input id={id} className="admin-input" type={field.kind === "number" ? "number" : field.kind === "password" ? "password" : field.kind === "email" ? "email" : "text"} autoComplete={field.kind === "password" ? "new-password" : "off"} min={field.kind === "number" ? 0 : undefined} step={field.kind === "number" ? 1 : undefined} required={field.required && !(editing && field.kind === "password")} maxLength={field.max} value={String(value ?? "")} onChange={e => update(field.kind === "number" ? Number(e.target.value) : e.target.value)}/>}</label>;
            })}
 {resource.path === "email-templates" && <div className="space-y-3"><p className="text-sm">Placeholder: {"{{report_code}}, {{user_name}}, {{status}}, {{external_note}}"}. Penyimpanan hanya mengizinkan tag pemformatan sederhana; script, style, gambar, dan URL dihapus.</p><h3 className="font-medium">Preview terisolasi</h3><iframe title="Preview email template" sandbox="" referrerPolicy="no-referrer" className="h-64 w-full rounded border bg-white" srcDoc={`<!doctype html><html><head><meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'none'; style-src 'none'; form-action 'none'; base-uri 'none'"></head><body>${String(values.body_html ?? "")}</body></html>`}/></div>}
 <div className="flex gap-3"><button className="admin-button" disabled={saving || !!optionsError}>{saving ? "Menyimpan…" : "Simpan"}</button><button type="button" className="admin-button-secondary" disabled={saving} onClick={() => setEditing(undefined)}>Batal</button></div></form>}
 </section>;
}
