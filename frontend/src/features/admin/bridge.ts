// Explicit route allowlist; never accept an arbitrary upstream URL.
export function bridgePath(parts: string[], method: string): string | null {
    if (parts[0] === "auth" && parts.length === 2) {
        if ((parts[1] === "me" && method === "GET") || (["login", "logout"].includes(parts[1]) && method === "POST"))
            return parts.join("/");
    }
    if (parts[0] === "public" && parts.length === 2 && ["report-types", "faqs"].includes(parts[1]) && method === "GET")
        return parts.join("/");
    if (parts[0] === "admin" && parts[1] === "dashboard" && parts.length === 2 && method === "GET")
        return parts.join("/");
    if (parts[0] === "admin" && parts[1] === "reports") {
        if (parts.length === 2 && method === "GET")
            return parts.join("/");
        if (parts.length === 3 && parts[2] === "options" && method === "GET")
            return parts.join("/");
        if (parts.length >= 3 && /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i.test(parts[2])) {
            if (parts.length === 3 && method === "GET")
                return parts.join("/");
            if (parts.length === 4 && ((parts[3] === "attachment" && method === "GET") || (parts[3] === "status" && method === "PATCH") || (parts[3] === "notes" && method === "POST")))
                return parts.join("/");
        }
        return null;
    }
    const resources = ["users", "roles", "permissions", "report-types", "faqs", "email-templates"];
    if (parts[0] !== "admin" || !resources.includes(parts[1]))
        return null;
    if (parts.length === 2 && ["GET", "POST"].includes(method))
        return parts.join("/");
    if (parts.length === 3 && /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i.test(parts[2]) && ["GET", "PATCH", "DELETE"].includes(method))
        return parts.join("/");
    return null;
}
