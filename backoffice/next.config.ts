import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Permet de lancer une seconde instance (tests de bout en bout) sans partager le dossier de build.
  distDir: process.env.NEXT_DIST_DIR ?? ".next",
  /* config options here */
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
