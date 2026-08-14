import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'node:path'
import fs from 'node:fs'

const versionFile = path.resolve(import.meta.dirname, '../VERSION')
const appVersion = fs.readFileSync(versionFile, 'utf8').trim()
const semVer = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(?:0|[1-9]\d*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9]\d*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/

if (!semVer.test(appVersion)) {
  throw new Error(`VERSION must contain a SemVer value, got ${JSON.stringify(appVersion)}`)
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  define: {
    __APP_VERSION__: JSON.stringify(appVersion),
  },
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  server: {
    host: '127.0.0.1',
    port: 4202,
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/subscribe': 'http://127.0.0.1:8080',
    },
  },
  build: {
    outDir: '../backend/internal/webui/dist',
    emptyOutDir: false,
  },
})
