"use client";

import Link from "next/link";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { AdminApiError, adminRequest } from "./client";
import type { PageData, Row } from "./contracts";

type Permission = Row & { slug: string; module_name: string };
type Role = Row & {
  name: string;
  description: string;
  permission_ids: string[];
};
const PAGE_SIZE = 20;
const EMPTY_PAGE: PageData = {
  items: [],
  total: 0,
  page: 1,
  page_size: PAGE_SIZE,
};

export function RolesManagementPage({ csrf }: { csrf: string }) {
  const [data, setData] = useState<PageData>(EMPTY_PAGE);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [query, setQuery] = useState("");
  const [revision, setRevision] = useState(0);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState("");
  const [expired, setExpired] = useState(false);
  const [editing, setEditing] = useState<Role | null | undefined>(undefined);
  const [pendingDelete, setPendingDelete] = useState<Role | null>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([]);

  const reload = useCallback(() => {
    setLoading(true);
    setRevision((value) => value + 1);
  }, []);

  useEffect(() => {
    let cancelled = false;
    adminRequest<PageData>(
      `admin/roles?page=${page}&page_size=${PAGE_SIZE}&search=${encodeURIComponent(query)}`,
    ).then(
      (result) => {
        if (!cancelled) {
          setData(result);
          setError("");
          setLoading(false);
        }
      },
      (reason: unknown) => {
        if (!cancelled) {
          setData({ ...EMPTY_PAGE, page });
          setError(
            reason instanceof Error ? reason.message : "Gagal memuat roles.",
          );
          setExpired(reason instanceof AdminApiError && reason.status === 401);
          setLoading(false);
        }
      },
    );
    return () => {
      cancelled = true;
    };
  }, [page, query, revision]);

  useEffect(() => {
    let cancelled = false;
    adminRequest<PageData>("admin/permissions?page=1&page_size=100").then(
      (result) => {
        if (!cancelled) setPermissions(result.items as Permission[]);
      },
      (reason: unknown) => {
        if (!cancelled)
          setError(
            reason instanceof Error
              ? reason.message
              : "Gagal memuat permissions.",
          );
      },
    );
    return () => {
      cancelled = true;
    };
  }, []);

  function open(role: Role | null) {
    setEditing(role);
    setName(role?.name ?? "");
    setDescription(role?.description ?? "");
    setSelectedPermissions(role?.permission_ids ?? []);
    setError("");
    setFeedback("");
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim()) {
      setError("Nama role wajib diisi.");
      return;
    }
    if (name.trim().length > 100) {
      setError("Nama role maksimal 100 karakter.");
      return;
    }
    if (description.length > 10000) {
      setError("Deskripsi maksimal 10.000 karakter.");
      return;
    }
    setSaving(true);
    setError("");
    try {
      await adminRequest(`admin/roles${editing ? `/${editing.id}` : ""}`, {
        method: editing ? "PATCH" : "POST",
        headers: { "X-CSRF-Token": csrf },
        body: JSON.stringify({
          name: name.trim(),
          description: description.trim(),
          permission_ids: selectedPermissions,
        }),
      });
      setEditing(undefined);
      setFeedback("Role berhasil disimpan.");
      reload();
    } catch (reason) {
      setError(
        reason instanceof Error ? reason.message : "Gagal menyimpan role.",
      );
      if (reason instanceof AdminApiError && reason.status === 401)
        setExpired(true);
    } finally {
      setSaving(false);
    }
  }

  async function remove() {
    const role = pendingDelete;
    if (!role) return;
    setSaving(true);
    setPendingDelete(null);
    setError("");
    try {
      await adminRequest(`admin/roles/${role.id}`, {
        method: "DELETE",
        headers: { "X-CSRF-Token": csrf },
      });
      setFeedback("Role berhasil dihapus.");
      reload();
    } catch (reason) {
      setError(
        reason instanceof Error ? reason.message : "Gagal menghapus role.",
      );
      if (reason instanceof AdminApiError && reason.status === 401)
        setExpired(true);
    } finally {
      setSaving(false);
    }
  }

  if (expired)
    return (
      <section>
        <h1 className="text-2xl font-semibold">Sesi berakhir</h1>
        <p role="alert" className="my-4">
          Silakan login kembali untuk melanjutkan.
        </p>
        <Link className="admin-button" href="/admin/login">
          Login kembali
        </Link>
      </section>
    );

  return (
    <section className="space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold">Roles</h1>
          <p className="mt-1 text-sm text-slate-600">
            Atur role administrator dan permission yang dimilikinya.
          </p>
        </div>
        <button
          className="admin-button"
          onClick={() => open(null)}
          disabled={saving}
        >
          Tambah role
        </button>
      </header>
      {feedback && (
        <p role="status" className="text-emerald-700">
          {feedback}
        </p>
      )}
      {error && (
        <div role="alert" className="text-red-700">
          {error}{" "}
          <button className="underline" onClick={reload}>
            Coba lagi
          </button>
        </div>
      )}
      <form
        className="flex flex-wrap items-end gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          setPage(1);
          setQuery(search);
          setLoading(true);
          setRevision((value) => value + 1);
        }}
      >
        <label className="min-w-48 flex-1">
          Cari role
          <input
            className="admin-input"
            value={search}
            maxLength={200}
            onChange={(event) => setSearch(event.target.value)}
          />
        </label>
        <button className="admin-button-secondary">Cari</button>
      </form>
      {loading ? (
        <p role="status">Memuat roles…</p>
      ) : (
        <>
          <div className="overflow-x-auto rounded-xl border border-slate-200">
            <table className="w-full text-left text-sm shadow-md">
              <caption className="sr-only">Daftar roles</caption>
              <thead>
                <tr>
                  <th className="p-4" scope="col">
                    Nama
                  </th>
                  <th className="p-4" scope="col">
                    Deskripsi
                  </th>
                  <th className="p-4" scope="col">
                    Tindakan
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white">
                {data.items.map((item) => {
                  const role = item as Role;
                  return (
                    <tr className="border-t border-slate-200" key={role.id}>
                      <td className="p-4 font-medium">{role.name}</td>
                      <td className="max-w-md truncate p-4">
                        {role.description || "—"}
                      </td>
                      <td className="whitespace-nowrap p-4">
                        <button
                          className="mr-4 underline"
                          disabled={saving}
                          onClick={() => open(role)}
                        >
                          Edit
                        </button>
                        <button
                          className="text-red-700 underline"
                          disabled={saving}
                          onClick={() => setPendingDelete(role)}
                        >
                          Hapus
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
            {!data.items.length && (
              <p className="p-6">Belum ada role yang cocok.</p>
            )}
          </div>
          <nav
            aria-label="Pagination roles"
            className="flex items-center gap-4"
          >
            <button
              className="admin-button-secondary"
              disabled={page <= 1}
              onClick={() => {
                setLoading(true);
                setPage((value) => value - 1);
              }}
            >
              Sebelumnya
            </button>
            <span>
              Halaman {page} · {data.total} data
            </span>
            <button
              className="admin-button-secondary"
              disabled={page * PAGE_SIZE >= data.total}
              onClick={() => {
                setLoading(true);
                setPage((value) => value + 1);
              }}
            >
              Berikutnya
            </button>
          </nav>
        </>
      )}
      {editing !== undefined && (
        <form
          onSubmit={save}
          className="space-y-5 rounded-xl border border-slate-300 p-6"
          aria-label={editing ? "Edit role" : "Tambah role"}
        >
          <h2 className="text-xl font-semibold">
            {editing ? "Edit role" : "Tambah role"}
          </h2>
          <label className="block">
            Nama <span aria-hidden="true">*</span>
            <input
              className="admin-input"
              required
              maxLength={100}
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
          </label>
          <label className="block">
            Deskripsi
            <textarea
              className="admin-input"
              rows={4}
              maxLength={10000}
              value={description}
              onChange={(event) => setDescription(event.target.value)}
            />
          </label>
          <fieldset className="space-y-2">
            <legend className="mb-3 font-medium">Permissions</legend>
            {permissions.map((permission) => (
              <label key={permission.id} className="block">
                <input
                  type="checkbox"
                  className="mr-3"
                  checked={selectedPermissions.includes(permission.id)}
                  onChange={(event) =>
                    setSelectedPermissions((current) =>
                      event.target.checked
                        ? [...current, permission.id]
                        : current.filter((id) => id !== permission.id),
                    )
                  }
                />
                {permission.slug}
              </label>
            ))}
            {!permissions.length && (
              <p className="text-sm text-slate-600">
                Belum ada permission tersedia.
              </p>
            )}
          </fieldset>
          <div className="flex gap-3">
            <button className="admin-button" disabled={saving}>
              {saving ? "Menyimpan…" : "Simpan"}
            </button>
            <button
              type="button"
              className="admin-button-secondary"
              disabled={saving}
              onClick={() => setEditing(undefined)}
            >
              Batal
            </button>
          </div>
        </form>
      )}
      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(openState) => {
          if (!openState && !saving) setPendingDelete(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader className="text-left">
            <AlertDialogTitle>Hapus role ini?</AlertDialogTitle>
            <AlertDialogDescription>
              Role yang masih digunakan atau direferensikan tidak dapat dihapus.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="outline" disabled={saving}>
              Batal
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={saving}
              onClick={() => void remove()}
            >
              Hapus
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}
