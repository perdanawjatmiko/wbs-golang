import "server-only";

export function serverConfig() {
  const appName = process.env.NEXT_PUBLIC_APP_NAME ?? "WBS";
  if (!appName.trim()) throw new Error("NEXT_PUBLIC_APP_NAME must not be blank");
  const raw = process.env.API_BASE_URL ?? "http://127.0.0.1:8080";
  let url: URL;
  try { url = new URL(raw); } catch { throw new Error("API_BASE_URL must be a valid HTTP origin"); }
  if (!["http:", "https:"].includes(url.protocol) || url.username || url.password || url.search || url.hash || url.pathname !== "/") {
    throw new Error("API_BASE_URL must be an HTTP origin without credentials or paths");
  }
  // HTTP is permitted for local or private server-to-server development connections.
  return { appName, apiBaseURL: url.origin, apiTimeoutMs: 5000 };
}
