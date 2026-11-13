export type User = {
    id: string;
    name: string;
    email: string;
    role_id: string;
    role_name: string;
    permissions: string[];
};
export type Session = {
    user: User;
    csrf_token: string;
};
export type FieldValue = string | boolean | number | string[] | null;
export type Row = {
    id: string;
    [key: string]: FieldValue;
};
export type PageData = {
    items: Row[];
    total: number;
    page: number;
    page_size: number;
};
export type Field = {
    name: string;
    label: string;
    kind?: "password" | "email" | "textarea" | "boolean" | "number" | "role" | "report-type" | "permissions";
    required?: boolean;
    max?: number;
};
export type Resource = {
    path: string;
    title: string;
    permission: string;
    security?: boolean;
    fields: Field[];
    columns: string[];
};
export const resources: Resource[] = [
    { path: "users", title: "Admin Users", permission: "users.manage", security: true, columns: ["name", "email", "is_active"], fields: [{ name: "name", label: "Nama", required: true, max: 150 }, { name: "email", label: "Email", kind: "email", required: true, max: 254 }, { name: "role_id", label: "Role", kind: "role", required: true }, { name: "password", label: "Password (12–72 byte; kosongkan saat edit untuk mempertahankan)", kind: "password", required: true }, { name: "is_active", label: "Aktif", kind: "boolean" }] },
    { path: "roles", title: "Roles", permission: "roles.manage", security: true, columns: ["name", "description"], fields: [{ name: "name", label: "Nama", required: true, max: 100 }, { name: "description", label: "Deskripsi", kind: "textarea", max: 10000 }, { name: "permission_ids", label: "Permissions", kind: "permissions" }] },
    { path: "permissions", title: "Permissions", permission: "permissions.manage", security: true, columns: ["slug", "module_name"], fields: [{ name: "slug", label: "Slug (module.action)", required: true, max: 100 }, { name: "module_name", label: "Module", required: true, max: 100 }, { name: "description", label: "Deskripsi", kind: "textarea", max: 10000 }] },
    { path: "report-types", title: "Jenis Laporan", permission: "report-types.manage", columns: ["name", "is_active"], fields: [{ name: "name", label: "Nama", required: true, max: 150 }, { name: "description", label: "Deskripsi", kind: "textarea", max: 10000 }, { name: "is_active", label: "Aktif", kind: "boolean" }] },
    { path: "faqs", title: "FAQ", permission: "faqs.manage", columns: ["question", "sort_order", "is_active"], fields: [{ name: "report_type_id", label: "Jenis laporan (kosong = FAQ umum)", kind: "report-type" }, { name: "question", label: "Pertanyaan", required: true, kind: "textarea", max: 5000 }, { name: "answer", label: "Jawaban", required: true, kind: "textarea", max: 20000 }, { name: "sort_order", label: "Urutan", kind: "number" }, { name: "is_active", label: "Aktif", kind: "boolean" }] },
    { path: "email-templates", title: "Email Templates", permission: "email-templates.manage", columns: ["event_key", "subject"], fields: [{ name: "event_key", label: "Event key", required: true, max: 100 }, { name: "subject", label: "Subject", required: true, max: 255 }, { name: "body_html", label: "HTML body", required: true, kind: "textarea", max: 100000 }] },
];
export function canManage(user: User, resource: Resource): boolean {
    if (user.role_name === "Super Admin")
        return true;
    if (resource.security)
        return false;
    return user.permissions.includes(resource.permission) || user.permissions.includes("master.manage");
}
export function isSession(value: unknown): value is Session {
    if (!value || typeof value !== "object" || !("csrf_token" in value) || typeof value.csrf_token !== "string" || !("user" in value))
        return false;
    const user = value.user;
    return !!user && typeof user === "object" && ["id", "name", "email", "role_id", "role_name"].every(k => k in user && typeof (user as Record<string, unknown>)[k] === "string") && "permissions" in user && Array.isArray(user.permissions) && user.permissions.every(p => typeof p === "string");
}
export function validateLogin(email: string, password: string): string | null {
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) || email.length > 254)
        return "Email tidak valid.";
    if (!password || new TextEncoder().encode(password).length > 72)
        return "Password wajib diisi (maksimum 72 byte).";
    return null;
}
export function validateForm(resource: Resource, values: Record<string, FieldValue>, editing: boolean): string | null {
    for (const field of resource.fields) {
        const value = values[field.name];
        if (field.kind === "password") {
            if (editing && !value)
                continue;
            const size = new TextEncoder().encode(String(value ?? "")).length;
            if (size < 12 || size > 72)
                return "Password harus 12–72 byte.";
        }
        else if (field.required && (typeof value !== "string" || !value.trim()))
            return `${field.label} wajib diisi.`;
        if (typeof value === "string" && field.max && Array.from(value).length > field.max)
            return `${field.label} terlalu panjang.`;
        if (field.kind === "number" && (!Number.isInteger(value) || Number(value) < 0 || Number(value) > 2147483647))
            return "Urutan harus integer nonnegatif.";
        if (field.kind === "email" && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(String(value)))
            return "Email tidak valid.";
        if ((field.kind === "role" || field.kind === "report-type") && value && !/^[0-9a-f-]{36}$/i.test(String(value)))
            return "Referensi harus UUID valid.";
    }
    if (resource.path === "permissions" && !/^[a-z][a-z-]*\.[a-z][a-z-]*$/.test(String(values.slug)))
        return "Slug harus module.action.";
    if (resource.path === "email-templates") {
        if (!/^[A-Z][A-Z0-9_]*$/.test(String(values.event_key)))
            return "Event key tidak valid.";
        for (const text of [String(values.subject), String(values.body_html)]) {
            for (const match of text.matchAll(/\{\{\s*([a-z_]+)\s*\}\}/g))
                if (!["report_code", "user_name", "status", "external_note"].includes(match[1]))
                    return "Placeholder tidak didukung.";
            if (/\{\{|\}\}/.test(text.replace(/\{\{\s*([a-z_]+)\s*\}\}/g, "")))
                return "Placeholder tidak valid.";
        }
    }
    return null;
}

export function hasPermission(user:User,permission:string):boolean{return user.role_name==="Super Admin"||user.permissions.includes(permission);}
