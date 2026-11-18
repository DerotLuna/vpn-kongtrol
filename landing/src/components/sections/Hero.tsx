import { CSSProperties, FormEvent, useEffect, useRef, useState } from 'react'
import { OS } from '../../hooks/useOS'
import { Lang } from '../../i18n'
import { GITHUB_RELEASES, GITHUB_REPO } from '../../links'
import { pathFor, SITE_UPDATED, SITE_VERSION } from '../../site'

interface Props {
  os: OS
  lang: Lang
}

const BANNER = `██╗  ██╗ ██████╗ ███╗   ██╗ ██████╗ ████████╗██████╗  ██████╗ ██╗
██║ ██╔╝██╔═══██╗████╗  ██║██╔════╝ ╚══██╔══╝██╔══██╗██╔═══██╗██║
█████╔╝ ██║   ██║██╔██╗ ██║██║  ███╗   ██║   ██████╔╝██║   ██║██║
██╔═██╗ ██║   ██║██║╚██╗██║██║   ██║   ██║   ██╔══██╗██║   ██║██║
██║  ██╗╚██████╔╝██║ ╚████║╚██████╔╝   ██║   ██║  ██║╚██████╔╝███████╗
╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═══╝ ╚═════╝    ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚══════╝`

const CMD = 'kongtrol up --group work'

type Tunnel = { profile: string; adapter: string; ip: string }
const TUNNELS: Tunnel[] = [
  { profile: 'office', adapter: 'FortiClient', ip: '10.10.0.12' },
  { profile: 'dev-server', adapter: 'OpenVPN', ip: '172.16.4.9' },
  { profile: 'aws', adapter: 'OpenVPN', ip: '10.42.7.3' },
]

const BIN: Record<OS, string> = {
  windows: 'kongtrol-windows-amd64.exe',
  macos: 'kongtrol-darwin-arm64',
  linux: 'kongtrol-linux-amd64',
}

// Boot timeline (ms after first paint). The meaningful content (banner, the
// plain-language line, the two buttons) is in place within ~0.6s; the typed
// command and the tunnel rows are the show that follows.
const BOOT = {
  login: 0, banner: 80, tag1: 240, tag2: 300, sub: 360, ctas: 460,
  cmd: 760, resolving: 1840, rows: 2100, rowGap: 260,
  security: 2900, watchdog: 3160, bin: 3420, prompt: 3680,
}
const at = (ms: number, extra?: Record<string, string | number>) =>
  ({ '--at': `${ms}ms`, ...extra }) as CSSProperties

// one shell entry: a command the visitor typed, or output lines
type ShellEntry =
  | { kind: 'cmd'; text: string }
  | { kind: 'out'; lines: { cls: string; text: string }[] }

export default function Hero({ os, lang }: Props) {
  // The boot sequence is a CSS timeline (index.css, .boot-step): every line is
  // in the markup from the first byte — prerendered HTML included — so the
  // terminal never grows (no layout shift) and crawlers read the whole hero.
  // createRoot() replaces the prerendered nodes, which restarts their CSS
  // animations; reading how far the prerendered timeline already ran lets the
  // new nodes resume at the same frame instead of replaying from zero.
  const [bootOffset] = useState(() => {
    if (typeof document === 'undefined') return 0
    const t = document.querySelector('.th-body .boot-step')?.getAnimations?.()[0]?.currentTime
    return typeof t === 'number' ? Math.round(t) : 0
  })
  const [elapsed, setElapsed] = useState(0)
  const [shell, setShell] = useState<ShellEntry[]>([])
  const [input, setInput] = useState('')
  const [usContent, setUsContent] = useState(false)
  const bodyRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    const t = window.setInterval(() => setElapsed(v => v + 1), 1000)
    return () => window.clearInterval(t)
  }, [])

  // keep the newest line in view once the visitor starts interacting
  useEffect(() => {
    const el = bodyRef.current
    if (el && shell.length > 0) el.scrollTop = el.scrollHeight
  }, [shell])

  const es = lang === 'es'
  const copy = es
    ? {
      lastLogin: 'Last login: Mon Jul 14 09:41:07 2026 from 127.0.0.1',
      tag1: '# torre de kongtrol multi-VPN · 10 VPN · Win / macOS / Linux',
      tag2: '# ícono en la bandeja + panel web + CLI completa',
      sub: 'Conecta tus VPN como siempre. Kongtrol las detecta, las vigila, las reconecta si caen y manda cada sitio por la VPN correcta.',
      ctaDownload: '[ ⇩ Descargar en GitHub ]',
      ctaGuide: '[ man kongtrol — guía completa ]',
      resolving: '→ resolviendo credenciales desde el keychain… ok',
      security: ['⛨ kill switch', 'ARMADO', '· DNS guard', 'ACTIVO'],
      watchdog: '♺ watchdog: sondeo cada 5s · backoff 2s → 5m',
      binHint: `→ binario para tu sistema: ${BIN[os]}`,
      hint: 'escribe "help" y presiona Enter ↵',
      srTitle: 'VPN Kongtrol — todas tus VPN en un solo lugar',
    }
    : {
      lastLogin: 'Last login: Mon Jul 14 09:41:07 2026 from 127.0.0.1',
      tag1: '# multi-VPN kongtrol tower · 10 VPNs · Win / macOS / Linux',
      tag2: '# tray icon + web panel + full CLI',
      sub: 'Connect your VPNs the way you always do. Kongtrol detects them, watches them, reconnects them if they drop, and sends each site through the right VPN.',
      ctaDownload: '[ ⇩ Download on GitHub ]',
      ctaGuide: '[ man kongtrol — full guide ]',
      resolving: '→ resolving credentials from the keychain… ok',
      security: ['⛨ kill switch', 'ARMED', '· DNS guard', 'ACTIVE'],
      watchdog: '♺ watchdog: poll every 5s · backoff 2s → 5m',
      binHint: `→ binary for your system: ${BIN[os]}`,
      hint: 'type "help" and press Enter ↵',
      srTitle: 'VPN Kongtrol — all your VPNs in one place',
    }

  const fmtUptime = (offset: number) => {
    const total = Math.max(0, elapsed - offset)
    const m = Math.floor(total / 60)
    const s = total % 60
    return `${m}m ${String(s).padStart(2, '0')}s`
  }

  const runCommand = (raw: string): ShellEntry[] => {
    const cmd = raw.trim()
    const lower = cmd.toLowerCase().replace(/^kongtrol\s+/, '')
    const out = (lines: { cls: string; text: string }[]): ShellEntry[] =>
      [{ kind: 'cmd', text: cmd }, { kind: 'out', lines }]

    if (lower === '') return [{ kind: 'cmd', text: '' }]
    if (lower === 'clear' || lower === 'cls') { setShell([]); return [] }
    if (lower === 'help' || lower === '--help' || lower === '-h') {
      return out([
        { cls: 't-muted', text: es ? 'comandos disponibles:' : 'available commands:' },
        { cls: 't-dim', text: '  status          map <destino>    doctor' },
        { cls: 't-dim', text: '  up us-content   down us-content  version' },
        { cls: 't-dim', text: '  whoami          github           ' + (es ? 'guia' : 'guide') },
        { cls: 't-dim', text: '  clear           exit             konami' },
      ])
    }
    if (lower === 'status') {
      const rows = [
        ...TUNNELS.map(t => ({ cls: 't-ok-line', text: `✔ ${t.profile.padEnd(12)} ${t.adapter.padEnd(13)} ${t.ip.padEnd(13)} up` })),
        { cls: usContent ? 't-ok-line' : 't-dim', text: usContent ? `✔ ${'us-content'.padEnd(12)} ${'ProtonVPN'.padEnd(13)} ${'10.96.0.4'.padEnd(13)} up` : `○ ${'us-content'.padEnd(12)} ${'ProtonVPN'.padEnd(13)} ${'-'.padEnd(13)} down` },
        { cls: 't-muted', text: es ? `⛨ kill switch ARMADO · DNS guard ACTIVO` : `⛨ kill switch ARMED · DNS guard ACTIVE` },
      ]
      return out(rows)
    }
    if (lower.startsWith('map')) {
      const target = cmd.split(/\s+/)[1] ?? (es ? '' : '')
      if (!target) return out([{ cls: 't-dim', text: es ? 'uso: map <dominio|ip|app:exe>' : 'usage: map <domain|ip|app:exe>' }])
      const t = target.toLowerCase()
      let line
      if (t === 'amazonaws.com' || t.endsWith('.amazonaws.com')) line = `${target} → ${es ? 'regla' : 'rule'} «AWS workloads» → aws (OpenVPN)`
      else if (t.startsWith('10.')) line = `${target} → longest-prefix 10.10.0.0/16 → office (FortiClient)`
      else if (t.includes('claude')) line = `${target} → ${es ? 'regla' : 'rule'} «US content» → us-content (ProtonVPN)`
      else if (t.startsWith('app:')) line = `${target} → ${es ? 'política por app (experimental)' : 'app policy (experimental)'} → us-content`
      else line = `${target} → ${es ? 'sin regla — salida directa (sin VPN)' : 'no rule — direct egress (no VPN)'}`
      return out([{ cls: 't-muted', text: line }])
    }
    if (lower === 'up us-content' || lower === 'up') {
      setUsContent(true)
      return out([
        { cls: 't-ok-line', text: es ? '✔ us-content conectado — ProtonVPN · 10.96.0.4' : '✔ us-content connected — ProtonVPN · 10.96.0.4' },
      ])
    }
    if (lower === 'down us-content' || lower === 'down') {
      setUsContent(false)
      return out([{ cls: 't-dim', text: es ? '○ us-content desconectado — DNS restaurado' : '○ us-content disconnected — DNS restored' }])
    }
    if (lower === 'doctor') {
      return out([
        { cls: 't-ok-line', text: es ? '✔ binarios VPN encontrados (4/4)' : '✔ VPN binaries found (4/4)' },
        { cls: 't-ok-line', text: es ? '✔ credenciales en keychain (3 perfiles)' : '✔ keychain credentials (3 profiles)' },
        { cls: 't-ok-line', text: es ? '✔ permisos de administrador' : '✔ administrator permissions' },
        { cls: 't-muted', text: es ? 'todo listo · exit 0' : 'all good · exit 0' },
      ])
    }
    if (lower === 'version' || lower === '--version' || lower === '-v') {
      return out([{ cls: 't-muted', text: `kongtrol-site ${SITE_VERSION} · ${SITE_UPDATED} · Apache-2.0` }])
    }
    if (lower === 'whoami') {
      return out([{ cls: 't-muted', text: es ? 'invitado@kongtrol — root ya quisieras' : 'guest@kongtrol — you wish you were root' }])
    }
    if (lower === 'github') { window.open(GITHUB_REPO, '_blank'); return out([{ cls: 't-dim', text: `→ ${GITHUB_REPO}` }]) }
    if (lower === 'guia' || lower === 'guide' || lower === 'man kongtrol' || lower === 'man') {
      const guiaPath = pathFor('guia', lang)
      window.location.href = guiaPath
      return out([{ cls: 't-dim', text: `→ ${guiaPath}` }])
    }
    if (lower === 'exit' || lower === 'logout' || lower === 'quit' || lower === ':q') {
      return out([{ cls: 't-muted', text: es ? 'logout — es broma. de aquí no hay escape, prueba "guia".' : 'logout — kidding. there is no escape, try "guide".' }])
    }
    if (lower.startsWith('sudo')) {
      return out([{ cls: 't-muted', text: es ? 'usuario no está en sudoers. el incidente será reportado al Kong. 🦍' : 'user is not in the sudoers file. this incident will be reported to the Kong. 🦍' }])
    }
    if (lower.startsWith('rm ')) {
      return out([{ cls: 't-muted', text: es ? 'bonito intento. el kill switch también bloquea eso.' : 'nice try. the kill switch blocks that too.' }])
    }
    if (lower === 'konami') {
      return out([{ cls: 't-dim', text: '↑ ↑ ↓ ↓ ← → ← → B A' }])
    }
    if (lower === 'ls') {
      return out([{ cls: 't-dim', text: 'kongtrol.yaml  certs/  configs/  audit.log' }])
    }
    return out([
      { cls: 't-dim', text: (es ? 'comando no encontrado: ' : 'command not found: ') + cmd.split(/\s+/)[0] + (es ? ' — prueba "help"' : ' — try "help"') },
    ])
  }

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    const entries = runCommand(input)
    if (entries.length) setShell(prev => [...prev, ...entries])
    setInput('')
  }

  const focusInput = () => {
    if (window.getSelection()?.toString()) return
    inputRef.current?.focus()
  }

  return (
    <section id="motd" className="section th-section" style={{ borderTop: 'none' }}>
      <h1 className="sr-only">{copy.srTitle}</h1>
      <div className="container">
        <div className="hero-contour th-contour" aria-hidden="true" />
        <div className="terminal crt th-terminal">
          <div className="terminal-bar">
            <div className="terminal-dots" aria-hidden="true">
              <div className="terminal-dot red" />
              <div className="terminal-dot yellow" />
              <div className="terminal-dot green" />
            </div>
            <span className="terminal-title">ssh guest@kongtrol.tower</span>
          </div>
          <div
            className="terminal-body th-body"
            ref={bodyRef}
            onClick={focusInput}
            style={{ '--boot-offset': `${bootOffset}ms` } as CSSProperties}
          >
            <div className="boot-line boot-step t-dim" style={at(BOOT.login)}>{copy.lastLogin}</div>

            <pre className="th-banner boot-step" style={at(BOOT.banner)} aria-hidden="true">{BANNER}</pre>

            <div className="th-meta">
              <div className="boot-line boot-step t-dim" style={at(BOOT.tag1)}>{copy.tag1}</div>
              <div className="boot-line boot-step t-dim" style={at(BOOT.tag2)}>{copy.tag2}</div>
              <p className="th-sub boot-step" style={at(BOOT.sub)}>{copy.sub}</p>
            </div>

            <div className="th-ctas boot-step" style={at(BOOT.ctas)}>
              <a href={GITHUB_RELEASES} className="th-btn primary" target="_blank" rel="noreferrer">{copy.ctaDownload}</a>
              <a href={pathFor('guia', lang)} className="th-btn">{copy.ctaGuide}</a>
            </div>

            <div className="hero-cmd-line th-cmd boot-step" style={at(BOOT.cmd)}>
              <span className="t-prompt">$ </span>
              <span className="th-typed" style={at(BOOT.cmd, { '--n': CMD.length })}>
                <span className="t-cmd">{CMD}</span>
                <span className="th-typed-cursor" aria-hidden="true" />
              </span>
            </div>

            <div className="boot-line boot-step t-dim" style={at(BOOT.resolving)}>{copy.resolving}</div>

            <div className="boot-rows">
              {TUNNELS.map((t, i) => (
                <div key={t.profile} className="boot-row boot-step" style={at(BOOT.rows + i * BOOT.rowGap)}>
                  <span className="boot-check" aria-hidden="true">✔</span>
                  <span className="boot-profile">{t.profile}</span>
                  <span className="boot-adapter">{t.adapter}</span>
                  <span className="boot-ip">{t.ip}</span>
                  <span className="boot-uptime">{fmtUptime(0)}</span>
                </div>
              ))}
            </div>

            <div className="boot-line boot-security boot-step" style={at(BOOT.security)}>
              {copy.security[0]} <strong>{copy.security[1]}</strong> {copy.security[2]} <strong>{copy.security[3]}</strong>
            </div>
            <div className="boot-line boot-step t-muted" style={at(BOOT.watchdog)}>{copy.watchdog}</div>
            <div className="boot-line boot-step t-dim" style={at(BOOT.bin)}>{copy.binHint}</div>

            {/* visitor shell history — announced to screen readers as it arrives */}
            <div aria-live="polite">
              {shell.map((entry, i) => (
                entry.kind === 'cmd'
                  ? <div key={i} className="hero-cmd-line"><span className="t-prompt">$ </span><span className="t-cmd">{entry.text}</span></div>
                  : <div key={i}>{entry.lines.map((l, j) => <div key={j} className={`boot-line ${l.cls}`}>{l.text}</div>)}</div>
              ))}
            </div>

            <form className="th-prompt boot-step" style={at(BOOT.prompt)} onSubmit={onSubmit}>
              <span className="t-prompt" aria-hidden="true">$ </span>
              <input
                ref={inputRef}
                className="th-input"
                value={input}
                onChange={e => setInput(e.target.value)}
                placeholder={copy.hint}
                aria-label={es ? 'terminal interactiva de demostración' : 'interactive demo terminal'}
                autoComplete="off"
                autoCapitalize="off"
                spellCheck={false}
                maxLength={80}
              />
            </form>
          </div>
        </div>
      </div>
    </section>
  )
}
