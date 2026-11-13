"use client";
import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { adminRequest } from "./client";
import { validateLogin, type Session } from "./contracts";
export function LoginForm() {
    const router = useRouter();
    const [email, setEmail] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState("");
    const [busy, setBusy] = useState(false);
    async function submit(event: FormEvent<HTMLFormElement>) { event.preventDefault(); const invalid = validateLogin(email, password); if (invalid) {
        setError(invalid);
        return;
    } setBusy(true); setError(""); try {
        await adminRequest<Session>("auth/login", { method: "POST", body: JSON.stringify({ email, password }) });
        setPassword("");
        router.replace("/admin");
        router.refresh();
    }
    catch (error) {
        setError(error instanceof Error ? error.message : "Login gagal.");
    }
    finally {
        setBusy(false);
    } }
    return <form onSubmit={submit} className="space-y-5"><label className="block">Email<input className="admin-input" type="email" name="email" autoComplete="username" required maxLength={254} value={email} onChange={e => setEmail(e.target.value)}/></label><label className="block">Password<input className="admin-input" type="password" name="password" autoComplete="current-password" required value={password} onChange={e => setPassword(e.target.value)}/></label>{error && <p role="alert" className="text-red-700 dark:text-red-400">{error}</p>}<button className="admin-button" disabled={busy}>{busy ? "Memproses…" : "Login"}</button></form>;
}
