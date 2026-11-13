"use client";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { adminRequest, AdminApiError } from "./client";
export function LogoutButton({ csrf }: {
    csrf: string;
}) { const router = useRouter(); const [error, setError] = useState(""); const [busy, setBusy] = useState(false); async function logout() { setBusy(true); try {
    await adminRequest("auth/logout", { method: "POST", headers: { "X-CSRF-Token": csrf } });
    router.replace("/admin/login");
    router.refresh();
}
catch (error) {
    if (error instanceof AdminApiError && error.status === 401) {
        router.replace("/admin/login");
        router.refresh();
        return;
    }
    setError(error instanceof Error ? error.message : "Logout gagal.");
}
finally {
    setBusy(false);
} } return <div><button className="admin-button-secondary" onClick={logout} disabled={busy}>Logout</button>{error && <p role="alert">{error}</p>}</div>; }
