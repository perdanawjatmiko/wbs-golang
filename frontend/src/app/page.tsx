import { Suspense } from "react";
import { Connectivity } from "@/features/system/connectivity";
import { StatusCard } from "@/components/ui/status-card";
import { serverConfig } from "@/lib/config/server";
export const dynamic = "force-dynamic";
export default function Home() {
  const { appName } = serverConfig();
  return <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-8 px-6 py-20">
    <header><p className="mb-3 text-sm text-zinc-500">Phase 0 · Development foundation</p><h1 className="text-4xl font-semibold tracking-tight">{appName}</h1><p className="mt-4 text-zinc-600 dark:text-zinc-400">Whistleblowing System &amp; Ticketing Application</p></header>
    <Suspense fallback={<StatusCard title="Koneksi backend"><p role="status">Memeriksa koneksi…</p></StatusCard>}><Connectivity /></Suspense>
  </main>;
}
