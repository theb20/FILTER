import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'

// CSP injectée uniquement au build : en développement, Vite/React Refresh
// injectent des scripts inline. Idéalement, la CSP est aussi posée en
// en-tête HTTP par l'hébergeur (une balise meta ne couvre pas frame-ancestors).
const CSP = [
  "default-src 'self'",
  "script-src 'self'",
  "style-src 'self' 'unsafe-inline'", // styles inline : index.html + attributs style React
  "img-src 'self' data: https:",
  "media-src 'self'",
  "font-src 'self' data:",
  "connect-src 'self'",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
].join('; ')

const cspPlugin = () => ({
  name: 'inject-csp',
  apply: 'build',
  transformIndexHtml(html) {
    return html.replace(
      '</title>',
      `</title>\n    <meta http-equiv="Content-Security-Policy" content="${CSP}" />`,
    )
  },
})

export default defineConfig({
  plugins: [
    react({
      babel: {
        plugins: [['babel-plugin-react-compiler']],
      },
    }),
    cspPlugin(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  server: {
    host: 'localhost', // plus d'exposition sur le réseau local
    port: 3001,
    open: true,
    strictPort: false,
  },
})
