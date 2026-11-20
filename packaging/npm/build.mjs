// Builds the npm packages for the Kongtrol CLI — version 1.0.0 · 2026-11-18.
//
//   node packaging/npm/build.mjs <version> <goreleaser dist dir> <out dir>
//
// Pattern (esbuild / Biome): one package per os+cpu holding the prebuilt
// binary (`os`/`cpu` fields, so npm/pnpm/yarn/bun install only the matching
// one) and a main package `kongtrol` that lists them all as
// optionalDependencies and runs the right binary from a tiny launcher. No
// postinstall script, no download at install time. Binaries are the raw
// release binaries (kongtrol-<os>-<arch>[.exe], cli-binary archive in
// .goreleaser.yaml): Go static builds, so no glibc/musl variants are needed.
// Decision and sources: docs/adr/0025-desktop-release-channels-first-run.md.
import { chmodSync, copyFileSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const [version, distDir, outDir] = process.argv.slice(2)
if (!version || !distDir || !outDir) {
  console.error('usage: node packaging/npm/build.mjs <version> <dist dir> <out dir>')
  process.exit(2)
}

const here = dirname(fileURLToPath(import.meta.url))
const root = resolve(here, '..', '..')
const scope = '@kongtrol'
const repository = { type: 'git', url: 'git+https://github.com/DerotLuna/vpn-kongtrol.git' }
const common = {
  version,
  license: 'Apache-2.0',
  author: 'Rubén Luna (https://github.com/DerotLuna)',
  homepage: 'https://kongtrol.app',
  repository,
}

// npm platform/arch names → Go GOOS/GOARCH.
const targets = [
  { os: 'linux', cpu: 'x64', goos: 'linux', goarch: 'amd64' },
  { os: 'linux', cpu: 'arm64', goos: 'linux', goarch: 'arm64' },
  { os: 'darwin', cpu: 'x64', goos: 'darwin', goarch: 'amd64' },
  { os: 'darwin', cpu: 'arm64', goos: 'darwin', goarch: 'arm64' },
  { os: 'win32', cpu: 'x64', goos: 'windows', goarch: 'amd64' },
  { os: 'win32', cpu: 'arm64', goos: 'windows', goarch: 'arm64' },
]

const optionalDependencies = {}
for (const t of targets) {
  const name = `${scope}/cli-${t.os}-${t.cpu}`
  const exe = t.os === 'win32' ? 'kongtrol.exe' : 'kongtrol'
  const src = join(distDir, `kongtrol-${t.goos}-${t.goarch}${t.os === 'win32' ? '.exe' : ''}`)
  const dir = join(outDir, `cli-${t.os}-${t.cpu}`)
  mkdirSync(join(dir, 'bin'), { recursive: true })
  copyFileSync(src, join(dir, 'bin', exe))
  chmodSync(join(dir, 'bin', exe), 0o755)
  writeFileSync(join(dir, 'package.json'), JSON.stringify({
    name,
    ...common,
    description: `Kongtrol CLI binary for ${t.os}-${t.cpu} (installed by the "kongtrol" package).`,
    os: [t.os],
    cpu: [t.cpu],
    files: ['bin'],
    preferUnplugged: true,
  }, null, 2) + '\n')
  copyFileSync(join(root, 'LICENSE'), join(dir, 'LICENSE'))
  optionalDependencies[name] = version
}

const mainDir = join(outDir, 'kongtrol')
mkdirSync(join(mainDir, 'bin'), { recursive: true })
copyFileSync(join(here, 'kongtrol.js'), join(mainDir, 'bin', 'kongtrol.js'))
copyFileSync(join(root, 'LICENSE'), join(mainDir, 'LICENSE'))
copyFileSync(join(here, 'README.md'), join(mainDir, 'README.md'))
writeFileSync(join(mainDir, 'package.json'), JSON.stringify({
  name: 'kongtrol',
  ...common,
  description: 'Kongtrol: connect and control several VPNs from one place (CLI + local dashboard).',
  keywords: ['vpn', 'wireguard', 'openvpn', 'cli'],
  bin: { kongtrol: 'bin/kongtrol.js' },
  files: ['bin', 'README.md'],
  engines: { node: '>=18' },
  optionalDependencies,
}, null, 2) + '\n')

const pkg = JSON.parse(readFileSync(join(mainDir, 'package.json'), 'utf8'))
console.log(`Built ${pkg.name}@${pkg.version} + ${targets.length} platform packages in ${outDir}`)
