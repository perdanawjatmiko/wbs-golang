"use client";
import { useEffect, useState } from "react";
import { faqs } from "./client";
import { relevantFAQs, type FAQ } from "./contracts";
export function InteractiveFAQ({ reportType = "" }: {
    reportType?: string;
}) {
    const [state, setState] = useState<{
        type: string;
        items: FAQ[];
        error: string;
    } | null>(null);
    const [retry, setRetry] = useState(0);
    useEffect(() => { let canceled = false; Promise.all([faqs("general"), reportType ? faqs(reportType) : Promise.resolve([])]).then(([general, specific]) => { if (!canceled)
        setState({ type: reportType, items: relevantFAQs(general, specific), error: "" }); }, () => { if (!canceled)
        setState({ type: reportType, items: [], error: "FAQ belum dapat dimuat." }); }); return () => { canceled = true; }; }, [reportType, retry]);
    const loading = state === null || state.type !== reportType;
    return <section aria-label="Pertanyaan yang sering diajukan" className="space-y-4"><h2 className="text-xl font-semibold">Pertanyaan yang sering diajukan</h2>{loading ? <p role="status">Memuat FAQ…</p> : state.error ? <p role="alert">{state.error} <button className="underline" onClick={() => setRetry(n => n + 1)}>Coba lagi</button></p> : state.items.length === 0 ? <p className="text-zinc-500">Belum ada FAQ untuk pilihan ini.</p> : state.items.map(faq => <details className="rounded-xl border border-zinc-200 p-4 dark:border-zinc-800" key={faq.id}><summary className="cursor-pointer font-medium">{faq.question}</summary><p className="mt-3 whitespace-pre-wrap text-zinc-600 dark:text-zinc-300">{faq.answer}</p></details>)}</section>;
}
