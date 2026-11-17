"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Underline from "@tiptap/extension-underline";
import LinkExtension from "@tiptap/extension-link";
import { adminRequest, AdminApiError } from "./client";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";

const events = ["REPORT_CREATED", "REPORT_ON_PROGRESS", "REPORT_DONE", "REPORT_REJECTED"] as const;
const placeholders = ["report_code", "user_name", "status", "external_note"] as const;
type EmailTemplate = { id: string; event_key: string; subject: string; body_html: string };

export function EmailTemplateEditor({ csrf, id }: { csrf: string; id?: string }) {
  const [eventKey, setEventKey] = useState<string>(events[0]);
  const [subject, setSubject] = useState("");
  const [ready, setReady] = useState(!id);
  const [loading, setLoading] = useState(!!id);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState("");
  const [preview, setPreview] = useState(false);
  const [initial, setInitial] = useState("");
  const [initialSubject, setInitialSubject] = useState("");
  const editor = useEditor({
    immediatelyRender: false,
    extensions: [StarterKit.configure({ link: false, underline: false }), Underline, LinkExtension.configure({ openOnClick: false, autolink: false, protocols: ["http", "https", "mailto"] })],
    content: "",
    onUpdate: () => setReady(true),
    editorProps: { attributes: { class: "min-h-64 rounded-b-lg border border-t-0 border-slate-300 bg-white p-4 text-sm leading-6 outline-none focus-visible:ring-2 focus-visible:ring-blue-700" } },
  });

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    adminRequest<EmailTemplate>(`admin/email-templates/${id}`).then((item) => {
      if (cancelled) return;
      setEventKey(item.event_key); setSubject(item.subject); setInitialSubject(item.subject); editor?.commands.setContent(item.body_html); setInitial(item.body_html); setLoading(false);
    }, (reason: unknown) => {
      if (!cancelled) { setError(reason instanceof Error ? reason.message : "Template tidak dapat dimuat."); setLoading(false); }
    });
    return () => { cancelled = true; };
  }, [editor, id]);

  const body = editor?.getHTML() ?? "";
  const dirty = !!editor && (body !== initial || subject !== initialSubject);
  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => { event.preventDefault(); event.returnValue = ""; };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);

  const insertPlaceholder = useCallback((value: string) => {
    editor?.chain().focus().insertContent(value).run();
  }, [editor]);

  async function save() {
    if (!editor) return;
    if (!subject.trim()) { setError("Subject wajib diisi."); return; }
    if (!editor.getText().trim()) { setError("Isi template wajib diisi."); return; }
    setSaving(true); setError(""); setFeedback("");
    try {
      await adminRequest(`admin/email-templates${id ? `/${id}` : ""}`, { method: id ? "PATCH" : "POST", headers: { "X-CSRF-Token": csrf }, body: JSON.stringify({ event_key: eventKey, subject: subject.trim(), body_html: editor.getHTML() }) });
      setInitial(editor.getHTML()); setInitialSubject(subject); setFeedback("Template berhasil disimpan.");
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Template tidak dapat disimpan.");
      if (reason instanceof AdminApiError && reason.status === 401) setFeedback("Sesi berakhir. Simpanan tidak dilakukan.");
    } finally { setSaving(false); }
  }

  if (loading) return <p role="status">Memuat template email…</p>;
  return <section className="mx-auto max-w-4xl space-y-6">
    <header><p className="public-eyebrow">Master data</p><h1 className="text-3xl font-semibold">{id ? "Edit email template" : "Buat email template"}</h1><p className="mt-2 text-sm text-slate-600">Konten disanitasi kembali oleh backend sebelum digunakan worker.</p></header>
    {error && <p role="alert" className="text-red-700">{error}</p>}{feedback && <p role="status">{feedback}</p>}
    <div className="admin-panel space-y-5">
      <label className="block">Event key<NativeSelect className="admin-input" value={eventKey} onChange={(event) => setEventKey(event.target.value)} disabled={!!id}>{events.map((event) => <NativeSelectOption key={event} value={event}>{event}</NativeSelectOption>)}</NativeSelect></label>
      <label className="block">Subject<input className="admin-input" maxLength={255} required value={subject} onChange={(event) => setSubject(event.target.value)} /></label>
      <section aria-label="Email content editor"><div className="flex flex-wrap items-center gap-2 rounded-t-lg border border-slate-300 bg-slate-50 p-2">
        <button type="button" className="admin-button-secondary" aria-pressed={editor?.isActive("bold")} onClick={() => editor?.chain().focus().toggleBold().run()}>Bold</button>
        <button type="button" className="admin-button-secondary" aria-pressed={editor?.isActive("italic")} onClick={() => editor?.chain().focus().toggleItalic().run()}>Italic</button>
        <button type="button" className="admin-button-secondary" aria-pressed={editor?.isActive("underline")} onClick={() => editor?.chain().focus().toggleUnderline().run()}>Underline</button>
        <button type="button" className="admin-button-secondary" onClick={() => editor?.chain().focus().toggleHeading({ level: 2 }).run()}>Heading</button>
        <button type="button" className="admin-button-secondary" onClick={() => editor?.chain().focus().setParagraph().run()}>Paragraph</button>
        <button type="button" className="admin-button-secondary" onClick={() => editor?.chain().focus().toggleOrderedList().run()}>Ordered list</button>
        <button type="button" className="admin-button-secondary" onClick={() => editor?.chain().focus().toggleBulletList().run()}>Bullet list</button>
        <button type="button" className="admin-button-secondary" onClick={() => { const href = window.prompt("URL tautan (https:// atau mailto:)"); if (href && /^(https?:|mailto:)/i.test(href)) editor?.chain().focus().extendMarkRange("link").setLink({ href }).run(); }}>Link</button>
        <button type="button" className="admin-button-secondary" onClick={() => editor?.chain().focus().undo().run()}>Undo</button>
        <button type="button" className="admin-button-secondary" onClick={() => editor?.chain().focus().redo().run()}>Redo</button>
      </div><EditorContent editor={editor} /></section>
      <div><h2 className="font-medium">Placeholder tersedia</h2><div className="mt-2 flex flex-wrap gap-2">{placeholders.map((name) => <button type="button" key={name} className="admin-button-secondary font-mono" onClick={() => insertPlaceholder(`{{${name}}}`)}>{`{{${name}}}`}</button>)}</div></div>
      <section><div className="flex items-center justify-between"><h2 className="font-medium">Preview aman</h2><button type="button" className="admin-button-secondary" onClick={() => setPreview((value) => !value)}>{preview ? "Sembunyikan" : "Tampilkan preview"}</button></div>{preview && <iframe title="Preview email" sandbox="" referrerPolicy="no-referrer" className="mt-3 h-80 w-full rounded-lg border bg-white" srcDoc={`<!doctype html><meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data:; style-src 'none'; script-src 'none'; form-action 'none'; base-uri 'none'"><h2>${subject.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!)}</h2>${body}`} />}</section>
      <div className="flex flex-wrap gap-3"><button type="button" className="admin-button" disabled={saving || !ready} onClick={() => void save()}>{saving ? "Menyimpan…" : "Simpan template"}</button><Link href="/admin/email-templates" className="admin-button-secondary">Kembali</Link></div>
    </div>
  </section>;
}
