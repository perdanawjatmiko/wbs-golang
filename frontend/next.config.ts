import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  poweredByHeader: false,
  async headers() {
    return [{source:"/:path*",headers:[
      {key:"X-Content-Type-Options",value:"nosniff"},
      {key:"X-Frame-Options",value:"DENY"},
      {key:"Referrer-Policy",value:"no-referrer"},
      {key:"Permissions-Policy",value:"camera=(), microphone=(), geolocation=()"},
      // App Router uses inline bootstrap scripts; do not invent an unsafe nonce implementation.
      // This restrictive framing/base/object policy coexists with its existing scripts and styles.
      {key:"Content-Security-Policy",value:"frame-ancestors 'none'; base-uri 'self'; object-src 'none'; form-action 'self'"},
    ]}];
  },
  /* Avoid logging URLs containing authorized contact search queries in development. */
 logging: { incomingRequests: false },
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
