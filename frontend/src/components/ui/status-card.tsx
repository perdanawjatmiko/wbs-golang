import type { ReactNode } from "react";
export function StatusCard({ title, children }: { title: string; children: ReactNode }) {
  return <section className="rounded-2xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-950" aria-label={title}>
    <h2 className="mb-4 text-lg font-semibold">{title}</h2>{children}
  </section>;
}
