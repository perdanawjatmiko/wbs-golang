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
  const appOriginRaw = process.env.APP_ORIGIN ?? "http://localhost:3000";
 let appOrigin: URL;
 try { appOrigin = new URL(appOriginRaw); } catch { throw new Error("APP_ORIGIN must be an HTTP origin"); }
 if (!["http:", "https:"].includes(appOrigin.protocol) || appOrigin.origin !== appOriginRaw || appOrigin.username || appOrigin.password) throw new Error("APP_ORIGIN must be an HTTP origin without path or credentials");
 if (process.env.NODE_ENV === "production" && process.env.APP_ORIGIN && appOrigin.protocol !== "https:" && !["localhost", "127.0.0.1"].includes(appOrigin.hostname)) throw new Error("Public production APP_ORIGIN requires HTTPS");
 return { appName, apiBaseURL: url.origin, apiTimeoutMs: 5000, appOrigin: appOrigin.origin };
}
