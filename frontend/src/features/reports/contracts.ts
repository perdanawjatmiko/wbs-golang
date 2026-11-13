export type ContactType = "EMAIL" | "PHONE";
export type ReportStatus = "Open" | "On Progress" | "Done" | "Rejected";
export type ReportType = {
    id: string;
    name: string;
    description: string;
};
export type FAQ = {
    id: string;
    report_type_id: string | null;
    question: string;
    answer: string;
    sort_order: number;
};
export type Page<T> = {
    items: T[];
    total: number;
    page: number;
    page_size: number;
};
export type UploadPolicy = {
    max_file_bytes: number;
    max_request_bytes: number;
    allowed_types: string[];
    max_files: number;
};
export type ReportInput = {
    name: string;
    contact: string;
    contact_type: ContactType;
    report_type_id: string;
    detail: string;
};
export type CreatedReport = {
    report_code: string;
    status: "Open";
    created_at: string;
    tracking_instructions: string;
};
export type TimelineEvent = {
    new_status: ReportStatus;
    external_note: string;
    created_at: string;
};
export type TrackedReport = {
    report_code: string;
    name: string;
    report_type: string;
    detail: string;
    status: ReportStatus;
    created_at: string;
    updated_at: string;
    timeline: TimelineEvent[];
};
export type FieldErrors = Partial<Record<keyof ReportInput | "attachment", string>>;
export const statuses: ReportStatus[] = ["Open", "On Progress", "Done", "Rejected"];
export function contactError(kind: ContactType, value: string): string | undefined {
    const trimmed = value.trim();
    if (!trimmed || trimmed.length > 254)
        return "Contact wajib diisi (maksimum 254 karakter).";
    if (kind === "EMAIL" && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed))
        return "Masukkan email yang valid.";
    if (kind === "PHONE" && !/^\+[1-9][0-9]{7,14}$/.test(trimmed.replace(/[ ().-]/g, "")))
        return "Gunakan nomor internasional + diikuti 8–15 digit.";
}
export function switchContact(input: ReportInput, kind: ContactType): ReportInput { return { ...input, contact_type: kind, contact: "" }; }
export function fileError(file: {
    name: string;
    size: number;
    type: string;
} | null, policy: UploadPolicy): string | undefined {
    if (!file)
        return;
    if (policy.max_files === 0)
        return "Upload dinonaktifkan.";
    if (file.size === 0)
        return "File kosong.";
    if (file.size > policy.max_file_bytes)
        return "Ukuran attachment melebihi batas.";
    const ext = file.name.split(".").pop()?.toLowerCase();
    const expected = ext === "jpg" || ext === "jpeg" ? "image/jpeg" : ext === "png" ? "image/png" : ext === "pdf" ? "application/pdf" : "";
    if (!expected || !policy.allowed_types.includes(expected) || (file.type && file.type !== expected) || /[\\/]/.test(file.name) || file.name.includes(".."))
        return "Gunakan JPEG, PNG, atau PDF dengan ekstensi dan tipe yang sesuai.";
}
export function validateReport(input: ReportInput, file: {
    name: string;
    size: number;
    type: string;
} | null, policy: UploadPolicy, types: ReportType[]): FieldErrors {
    const errors: FieldErrors = {};
    if (!input.name.trim() || Array.from(input.name.trim()).length > 150)
        errors.name = "Nama wajib diisi (maksimum 150 karakter).";
    const contact = contactError(input.contact_type, input.contact);
    if (contact)
        errors.contact = contact;
    if (!types.some(type => type.id === input.report_type_id))
        errors.report_type_id = "Pilih jenis laporan yang tersedia.";
    if (!input.detail.trim() || Array.from(input.detail.trim()).length > 20000)
        errors.detail = "Detail wajib diisi (maksimum 20.000 karakter).";
    const attachment = fileError(file, policy);
    if (attachment)
        errors.attachment = attachment;
    return errors;
}
export function relevantFAQs(general: FAQ[], specific: FAQ[]): FAQ[] { return [...new Map([...general, ...specific].map(faq => [faq.id, faq])).values()].sort((a, b) => a.sort_order - b.sort_order || a.id.localeCompare(b.id)); }
export function isCreated(value: unknown): value is CreatedReport { return !!value && typeof value === "object" && "report_code" in value && typeof value.report_code === "string" && "status" in value && value.status === "Open" && "created_at" in value && typeof value.created_at === "string" && "tracking_instructions" in value && typeof value.tracking_instructions === "string"; }
export function isTracked(value: unknown): value is TrackedReport { if (!value || typeof value !== "object")
    return false; const row = value as Record<string, unknown>; return ["report_code", "name", "report_type", "detail", "status", "created_at", "updated_at"].every(k => typeof row[k] === "string") && statuses.includes(row.status as ReportStatus) && Array.isArray(row.timeline) && row.timeline.every(e => e && typeof e === "object" && statuses.includes(e.new_status) && typeof e.external_note === "string" && typeof e.created_at === "string"); }
