import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Permet de lancer une seconde instance (tests de bout en bout) sans partager le dossier de build.
  distDir: process.env.NEXT_DIST_DIR ?? ".next",
  /* config options here */
  // Les images téléversées sont servies par l'API ; le navigateur les charge depuis la même origine que le backoffice.
  async rewrites() {
    return [{ source: "/v1/files/:name", destination: `${process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/v1/files/:name` }];
  },
  async redirects() {
    return [{ source: "/", destination: "/login", permanent: false }];
  },
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
