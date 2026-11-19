/// <reference types="vitest/config" />
// Kongtrol UI — Vite config · v0.3.0 · 2026-11-19
//
// Build: `pnpm build` writes web/ui/dist, which web/ui_embed.go embeds into the
// Go binary and internal/api serves at / (ADR-0043).
// Dev:   `pnpm dev` (or `make ui-dev`) serves http://127.0.0.1:5173/ and
//        proxies /api (REST + WebSocket) to the running daemon.
import { mkdirSync, writeFileSync } from 'node:fs'
import { fileURLToPath, URL } from 'node:url'
import react from '@vitejs/plugin-react'
import { defineConfig, type Plugin } from 'vite'

const brandDir = fileURLToPath(new URL('../../assets/brand', import.meta.url))
const outDir = fileURLToPath(new URL('./dist', import.meta.url))
// Test table shared with the Go policy tests (classify_targets.json, ADR-0052);
// tests read it, the build never does.
const sharedTestData = fileURLToPath(new URL('../../internal/policy/testdata', import.meta.url))

// Daemon address for the dev proxy. Default matches the dashboard's default
// port (internal/config/schema.go); override when `kongtrol config dashboard
// set-port` changed it.
const daemon = process.env.KONGTROL_DAEMON_URL ?? 'http://127.0.0.1:9741'
const devHost = '127.0.0.1'
const devPort = 5173
const devOrigin = `http://${devHost}:${devPort}`

// Vite empties dist/ on every build, which would delete the tracked
// dist/.gitkeep that keeps `go build` working without Node (the embed
// pattern needs the directory to exist). Re-create it, byte-identical, so a
// UI build never leaves the git tree dirty (GoReleaser refuses dirty trees).
function keepDistMarker(): Plugin {
  return {
    name: 'kongtrol-keep-dist-marker',
    apply: 'build',
    closeBundle() {
      mkdirSync(outDir, { recursive: true })
      writeFileSync(`${outDir}/.gitkeep`, '')
    },
  }
}

export default defineConfig({
  base: '/',
  plugins: [react(), keepDistMarker()],
  resolve: {
    alias: {
      // Single-source brand assets (assets/brand, shared with landing/).
      '@brand': brandDir,
    },
  },
  build: {
    outDir,
    emptyOutDir: true,
    sourcemap: false,
  },
  server: {
    // 127.0.0.1, not localhost: the daemon's session cookie (kongtrol_session,
    // ADR-0039) is host-only, and cookies ignore the port, so opening the
    // dashboard from Kongtrol (`kongtrol dashboard`) once makes the browser
    // send it here as well.
    // The token is never read or injected by this config.
    host: devHost,
    port: devPort,
    strictPort: true,
    fs: { allow: ['.', brandDir, sharedTestData] },
    proxy: {
      '/api': {
        target: daemon,
        changeOrigin: true, // Host header = daemon (its Host check stays on)
        ws: true,
        configure(proxy) {
          // The daemon only accepts its own Origin. Rewrite it solely for
          // requests coming from this dev page; any other Origin is passed
          // through unchanged, so the daemon still rejects it.
          proxy.on('proxyReq', (proxyReq, req) => {
            if (req.headers.origin === devOrigin) proxyReq.setHeader('origin', daemon)
          })
          proxy.on('proxyReqWs', (proxyReq, req) => {
            if (req.headers.origin === devOrigin) proxyReq.setHeader('origin', daemon)
          })
        },
      },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    css: false,
  },
})
