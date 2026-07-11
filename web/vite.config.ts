import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Dev: `npm run dev` proxies API + sync to a locally-running flagship (ADDR=:8099).
// Build: emits static assets into dist/, which the Go server embeds + serves (one image).
export default defineConfig({
  plugins: [react()],
  build: { outDir: 'dist', emptyOutDir: true },
  server: {
    proxy: {
      '/api': 'http://localhost:8099',
      '/sync': 'http://localhost:8099',
    },
  },
})
