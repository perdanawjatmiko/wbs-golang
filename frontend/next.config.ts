import type { NextConfig } from "next";

const nextConfig: NextConfig = {
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
