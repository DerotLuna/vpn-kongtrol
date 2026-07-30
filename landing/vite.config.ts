import { resolve } from 'node:path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Multi-page app: landing (/) + setup guide (/guia) + terms (/terminos)
// Served from a GitHub Pages project page at /vpn-kongtrol/.
export default defineConfig({
  base: '/vpn-kongtrol/',
  plugins: [react()],
  build: {
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html'),
        guia: resolve(__dirname, 'guia.html'),
        terminos: resolve(__dirname, 'terminos.html'),
      },
    },
  },
})
