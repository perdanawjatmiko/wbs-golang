"use client";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { adminRequest, AdminApiError } from "@/features/admin/client";
import {
  AccessError,
  PageControls,
  ReportSkeleton,
  StatusBadge,
} from "./shared";
import {
  statuses,
  validateStatus,
  validateNotes,
  workflowErrorMessage,
  type ReportDetail,
  type ReportHistory,
  type StatusInput,
  type NoteInput,
} from "./contracts";
import type { ReportStatus } from "@/features/reports/contracts";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/native-select";
export function AdminHistoryTimeline({
  histories,
}: {
  histories: ReportHistory[];
}) {
  return (
    <ol
      aria-label="Riwayat administrasi"
      className="space-y-6 border-l border-zinc-300 pl-5"
    >
      {histories.map((h) => (
        <li key={h.id} className="space-y-3">
          <p className="font-semibold">
            {h.old_status === h.new_status
              ? `Catatan · ${h.new_status}`
              : `${h.old_status} → ${h.new_status}`}
          </p>
          <p className="text-sm text-zinc-500">
            {new Date(h.created_at).toLocaleString("id-ID")} · {h.admin.name}
          </p>
          {h.internal_note && (
            <section className="rounded-lg border-l-4 border-amber-500 bg-amber-50 p-4 text-zinc-900">
              <h4 className="font-semibold">Internal · hanya administrator</h4>
              <p className="mt-2 whitespace-pre-wrap break-words">
                {h.internal_note}
              </p>
            </section>
          )}
          {h.external_note && (
            <section className="rounded-lg border-l-4 border-blue-500 bg-blue-50 p-4 text-zinc-900">
              <h4 className="font-semibold">Eksternal · terlihat pelapor</h4>
              <p className="mt-2 whitespace-pre-wrap break-words">
                {h.external_note}
              </p>
            </section>
          )}
        </li>
      ))}
    </ol>
  );
}
function WorkflowForm({
  report,
  csrf,
  notes,
  onDone,
  onConflict,
}: {
  report: ReportDetail;
  csrf: string;
  notes: boolean;
  onDone: () => void;
  onConflict: (message: string) => void;
}) {
  const [status, setStatus] = useState<ReportStatus>(report.status);
  const [internal, setInternal] = useState("");
  const [external, setExternal] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const gate = useRef(false);
  async function submit(e: FormEvent) {
    e.preventDefault();
    if (gate.current) return;
    const input: StatusInput = {
      status,
      internal_note: internal.trim(),
      external_note: external.trim(),
      expected_updated_at: report.updated_at,
    };
    const message = notes
      ? validateNotes(input)
      : validateStatus(input, report.status);
    if (message) {
      setError(message);
      return;
    }
    gate.current = true;
    setBusy(true);
    setError("");
    try {
      const payload: StatusInput | NoteInput = notes
        ? {
            internal_note: input.internal_note,
            external_note: input.external_note,
            expected_updated_at: input.expected_updated_at,
          }
        : input;
      await adminRequest(
        `admin/reports/${report.id}/${notes ? "notes" : "status"}`,
        {
          method: notes ? "POST" : "PATCH",
          headers: { "X-CSRF-Token": csrf },
          body: JSON.stringify(payload),
        },
      );
      onDone();
    } catch (err) {
      const message = workflowErrorMessage(
        err instanceof AdminApiError ? err.code : undefined,
        err instanceof Error
          ? err.message
          : "Perubahan belum dapat dipastikan.",
      );
      if (
        err instanceof AdminApiError &&
        (err.status === 409 ||
          err.status === 0 ||
          err.status >= 500 ||
          err.status === 401 ||
          err.status === 403)
      ) {
        onConflict(
          message + " Data dimuat ulang; tinjau sebelum mengirim kembali.",
        );
      } else setError(message);
    } finally {
      gate.current = false;
      setBusy(false);
    }
  }
  return (
    <form
      onSubmit={submit}
      className="space-y-4 rounded-xl border border-zinc-200 p-5 dark:border-zinc-800"
    >
      <h3 className="text-xl font-semibold">
        {notes ? "Tambah catatan" : "Ubah status"}
      </h3>
      <fieldset disabled={busy} className="space-y-4">
        <legend className="sr-only">
          {notes ? "Catatan" : "Perubahan status"}
        </legend>
        {!notes && (
          <label className="block">
            Status tujuan
            <NativeSelect
              className="admin-input"
              value={status}
              onChange={(e) => setStatus(e.target.value as ReportStatus)}
            >
              {statuses.map((s) => (
                <NativeSelectOption key={s} value={s}>
                  {s}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </label>
        )}
        <label className="block">
          Catatan internal
          <textarea
            className="admin-input"
            maxLength={10000}
            rows={4}
            value={internal}
            onChange={(e) => setInternal(e.target.value)}
          />
          <span className="text-sm text-zinc-500">
            Hanya administrator. Maksimum 10.000 karakter.
          </span>
        </label>
        <label className="block">
          Catatan eksternal
          <textarea
            className="admin-input"
            maxLength={10000}
            rows={4}
            value={external}
            required={!notes && status === "Rejected"}
            onChange={(e) => setExternal(e.target.value)}
          />
          <span className="text-sm text-zinc-500">
            Terlihat pelapor melalui tracking.
            {!notes && status === "Rejected"
              ? " Wajib menjelaskan alasan penolakan."
              : ""}
          </span>
        </label>
        <button className="admin-button" disabled={busy}>
          {busy ? "Menyimpan…" : notes ? "Simpan catatan" : "Simpan status"}
        </button>
      </fieldset>
      {error && (
        <p role="alert" className="text-red-700 dark:text-red-400">
          {error}
        </p>
      )}
    </form>
  );
}
export function ReportSummary({ report }: { report: ReportDetail }) {
  return (
    <div className="w-full">
      <div className="report-detail-main">
        <section className="admin-panel report-overview">
          <div className="report-overview-heading">
            <div>
              <p className="public-eyebrow">Laporan masuk</p>
              <h2 className="report-code-heading">{report.code}</h2>
            </div>
            <StatusBadge status={report.status} />
          </div>
          <div className="report-identity-grid">
            <div>
              <span>Pelapor</span>
              <strong>{report.reporter_name}</strong>
            </div>
            <div>
              <span>Contact</span>
              <strong className="break-all">{report.contact}</strong>
            </div>
            <div>
              <span>Jenis contact</span>
              <strong>
                {report.contact_type === "EMAIL" ? "Email" : "Telepon"}
              </strong>
            </div>
            <div>
              <span>Jenis laporan</span>
              <strong>{report.report_type}</strong>
            </div>
            <div>
              <span>Dibuat</span>
              <strong>
                {new Date(report.created_at).toLocaleString("id-ID")}
              </strong>
            </div>
            <div>
              <span>Diperbarui</span>
              <strong>
                {new Date(report.updated_at).toLocaleString("id-ID")}
              </strong>
            </div>
          </div>
          <div className="report-detail-copy">
            <h3>Detail laporan</h3>
            <p>{report.detail}</p>
          </div>
          <div className="report-attachment">
            <div>
              <strong>Bukti pendukung</strong>
              <span>
                {report.attachment.available
                  ? `${report.attachment.name ?? report.attachment.content_type ?? "File"}${report.attachment.size ? ` · ${Math.ceil(report.attachment.size / 1024)} KiB` : ""}`
                  : "Tidak ada attachment"}
              </span>
            </div>
            {report.attachment.available && (
              <a
                className="admin-button-secondary"
                href={`/api/backend/admin/reports/${report.id}/attachment`}
              >
                Unduh file
              </a>
            )}
          </div>
        </section>
      </div>
      {/* <aside className="report-detail-side">
        <section className="admin-panel status-context">
          <h2>Status saat ini</h2>
          <StatusBadge status={report.status} />
          <p>Perubahan status tercatat dalam riwayat laporan.</p>
        </section>
      </aside> */}
    </div>
  );
}
export function AdminReportDetail({
  id,
  csrf,
  canUpdate,
}: {
  id: string;
  csrf: string;
  canUpdate: boolean;
}) {
  const [report, setReport] = useState<ReportDetail | null>(null);
  const [error, setError] = useState<{
    message: string;
    status: number;
  } | null>(null);
  const [revision, setRevision] = useState(0);
  const [historyPage, setHistoryPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [notice, setNotice] = useState("");
  const [showNoteForm, setShowNoteForm] = useState(false);
  useEffect(() => {
    let ignore = false;
    adminRequest<ReportDetail>(
      `admin/reports/${id}?history_page=${historyPage}&history_per_page=50`,
    ).then(
      (data) => {
        if (!ignore) {
          setReport(data);
          setError(null);
          setLoading(false);
        }
      },
      (err) => {
        if (!ignore) {
          setReport(null);
          setError({
            message: err.message,
            status: err instanceof AdminApiError ? err.status : 0,
          });
          setLoading(false);
        }
      },
    );
    return () => {
      ignore = true;
    };
  }, [id, revision, historyPage]);
  function refresh(message: string) {
    setNotice(message);
    setLoading(true);
    setRevision((n) => n + 1);
  }
  if (error) return <AccessError {...error} retry={() => refresh("")} />;
  return (
    <div className="report-detail-page">
      <div className="page-heading">
        <div>
          <p className="public-eyebrow">Manajemen laporan</p>
          <h1>Detail laporan</h1>
          <p>Informasi pelapor, status terkini, dan riwayat tindak lanjut.</p>
        </div>
      </div>
      {notice && (
        <p role="status" className="action-feedback">
          {notice}
        </p>
      )}
      {loading || !report ? (
        <ReportSkeleton />
      ) : (
        <>
          <ReportSummary report={report} />
          <div className="grid gap-4 grid-cols-1 md:grid-cols-3">
            {canUpdate && (
              <aside key={report.updated_at + revision} className="col-span-1">
                <WorkflowForm
                  report={report}
                  csrf={csrf}
                  notes={false}
                  onDone={() => refresh("Status berhasil disimpan.")}
                  onConflict={refresh}
                />
              </aside>
            )}
            <section className="admin-panel history-panel col-span-2">
              <div className="section-heading">
                <div>
                  <h2>Riwayat aktivitas</h2>
                  <p>Catatan internal hanya terlihat oleh administrator.</p>
                </div>
                {canUpdate && (
                  <button
                    type="button"
                    className="admin-button-secondary"
                    aria-expanded={showNoteForm}
                    aria-controls="report-note-form"
                    onClick={() => setShowNoteForm((visible) => !visible)}
                  >
                    {showNoteForm ? "Tutup form catatan" : "Tambah catatan"}
                  </button>
                )}
              </div>
              {report.histories.length === 0 ? (
                <div className="empty-state">
                  <span className="empty-icon" aria-hidden="true">
                    —
                  </span>
                  <h3>Belum ada aktivitas</h3>
                  <p>Laporan dibuat dengan status Open.</p>
                </div>
              ) : (
                <AdminHistoryTimeline histories={report.histories} />
              )}
              {canUpdate && showNoteForm && (
                <div id="report-note-form" className="history-note-form">
                  <WorkflowForm
                    report={report}
                    csrf={csrf}
                    notes
                    onDone={() => {
                      setShowNoteForm(false);
                      refresh("Catatan berhasil disimpan.");
                    }}
                    onConflict={refresh}
                  />
                </div>
              )}
              <PageControls
                meta={report.history_pagination}
                onPage={(page) => {
                  setLoading(true);
                  setHistoryPage(page);
                }}
              />
            </section>
          </div>
        </>
      )}
    </div>
  );
}
