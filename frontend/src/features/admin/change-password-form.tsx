"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { adminRequest } from "./client";

export function ChangePasswordForm({ csrf }: { csrf: string }) {
  const router = useRouter();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (new TextEncoder().encode(newPassword).length < 12 || new TextEncoder().encode(newPassword).length > 72) { setError("Password baru harus 12–72 byte."); return; }
    if (newPassword !== confirmPassword) { setError("Konfirmasi password tidak cocok."); return; }
    setBusy(true); setError("");
    try {
      await adminRequest("auth/change-password", { method: "POST", headers: { "X-CSRF-Token": csrf }, body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }) });
      setCurrentPassword(""); setNewPassword(""); setConfirmPassword(""); router.replace("/admin/login?password=changed"); router.refresh();
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Password tidak dapat diperbarui."); }
    finally { setBusy(false); }
  }
  return <form onSubmit={submit} className="space-y-4"><label className="block">Password saat ini<input className="admin-input" type="password" autoComplete="current-password" required maxLength={72} value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} /></label><label className="block">Password baru<input className="admin-input" type="password" autoComplete="new-password" required maxLength={72} value={newPassword} onChange={(event) => setNewPassword(event.target.value)} /></label><label className="block">Konfirmasi password baru<input className="admin-input" type="password" autoComplete="new-password" required maxLength={72} value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} /></label>{error && <p role="alert" className="text-red-700">{error}</p>}<button className="admin-button" disabled={busy}>{busy ? "Memperbarui…" : "Ubah password"}</button></form>;
}
