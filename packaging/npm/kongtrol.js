#!/usr/bin/env node
// Launcher for the Kongtrol CLI installed from npm — version 1.0.0 · 2026-11-18.
// Runs the prebuilt binary from the @kongtrol/cli-<os>-<cpu> package that the
// package manager installed for this machine (see packaging/npm/build.mjs).
'use strict'
const { spawnSync } = require('node:child_process')
const { chmodSync } = require('node:fs')

const es = (Intl.DateTimeFormat().resolvedOptions().locale || '').toLowerCase().startsWith('es')
const pkg = `@kongtrol/cli-${process.platform}-${process.arch}`
const exe = process.platform === 'win32' ? 'kongtrol.exe' : 'kongtrol'

let bin
try {
  bin = require.resolve(`${pkg}/bin/${exe}`)
} catch {
  console.error(es
    ? `Kongtrol no tiene binario para ${process.platform}-${process.arch} (falta ${pkg}).\n` +
      'Qué hacer: reinstala sin omitir dependencias opcionales (npm i -g kongtrol, sin --no-optional) o descarga Kongtrol desde https://kongtrol.app'
    : `Kongtrol has no binary for ${process.platform}-${process.arch} (${pkg} is missing).\n` +
      'What to do: reinstall without skipping optional dependencies (npm i -g kongtrol, no --no-optional) or download Kongtrol from https://kongtrol.app')
  process.exit(1)
}

function run() {
  return spawnSync(bin, process.argv.slice(2), { stdio: 'inherit', windowsHide: false })
}

let res = run()
if (res.error && res.error.code === 'EACCES' && process.platform !== 'win32') {
  // Some package managers drop the executable bit when unpacking.
  try { chmodSync(bin, 0o755) } catch { /* reported below */ }
  res = run()
}
if (res.error) {
  console.error(es
    ? `No se pudo ejecutar Kongtrol: ${res.error.message}\nQué hacer: reinstala con npm i -g kongtrol`
    : `Could not run Kongtrol: ${res.error.message}\nWhat to do: reinstall with npm i -g kongtrol`)
  process.exit(1)
}
if (res.signal) process.kill(process.pid, res.signal)
process.exit(res.status ?? 1)
