import { CSSProperties, FormEvent, useEffect, useRef, useState } from 'react'
import { OS } from '../../hooks/useOS'
import { useDetectedOS } from '../../hooks/useVisitor'
import { Lang } from '../../i18n'
import { GITHUB_REPO, RELEASE_ASSETS, ReleaseAssetKey, WINDOWS_INSTALLER_FILE } from '../../links'
import { pathFor, SITE_UPDATED, SITE_VERSION } from '../../site'
import { CTA_COPY, DownloadIcon, DownloadNote, GitHubIcon, PRIMARY_COPY, primaryDownloadHref } from '../ui/downloadCta'
import SendToComputer from '../ui/SendToComputer'

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

const BIN: Record<Exclude<OS, 'windows'>, ReleaseAssetKey> = {
  macos: 'macos-arm64',
  linux: 'linux-amd64',
}
// the file the boot log names for this system (Windows: the installer)
const fileFor = (os: OS) => os === 'windows' ? WINDOWS_INSTALLER_FILE : RELEASE_ASSETS[BIN[os]]

// Boot timeline (ms after first paint). The meaningful content (the h1, the
// plain-language line, the two buttons) is in place within ~0.6s — the h1
// itself never animates; the typed command and the tunnel rows are the
// decorative show that follows.
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

// The tunnel rows with their live uptime. The 1-second clock lives here, not
// in Hero, so each tick re-renders three rows instead of the whole hero
// (ADR-0031).
function BootRows() {
  const [elapsed, setElapsed] = useState(0)
  useEffect(() => {
    const t = window.setInterval(() => setElapsed(v => v + 1), 1000)
    return () => window.clearInterval(t)
  }, [])
  const uptime = `${Math.floor(elapsed / 60)}m ${String(elapsed % 60).padStart(2, '0')}s`
  return (
    <div className="boot-rows">
      {TUNNELS.map((t, i) => (
        <div key={t.profile} className="boot-row boot-step" style={at(BOOT.rows + i * BOOT.rowGap)}>
          <span className="boot-check" aria-hidden="true">✔</span>
          <span className="boot-profile">{t.profile}</span>
          <span className="boot-adapter">{t.adapter}</span>
          <span className="boot-ip">{t.ip}</span>
          <span className="boot-uptime">{uptime}</span>
        </div>
      ))}
    </div>
  )
}

// on macOS / Linux (and before hydration) the button scrolls down to
// #install: an arrow, not a download tray
function ArrowDownIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M8 2.5v11M3.5 9 8 13.5 12.5 9" />
    </svg>
  )
}

export default function Hero({ os, lang }: Props) {
  // The boot sequence is a CSS timeline (index.css, .boot-step): every line is
  // in the markup from the first byte — prerendered HTML included — so the
  // terminal never grows (no layout shift) and crawlers read the whole hero.
  // hydrateRoot() keeps the prerendered nodes, so their CSS animations run on
  // from first paint without restarting.
  const [shell, setShell] = useState<ShellEntry[]>([])
  const [input, setInput] = useState('')
  const [usContent, setUsContent] = useState(false)
  const logRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // the log has a fixed height and scrolls inside (ADR-0066): after every
  // command the newest line comes into view — instantly under reduced motion
  useEffect(() => {
    const el = logRef.current
    if (!el || shell.length === 0) return
    const instant = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    el.scrollTo({ top: el.scrollHeight, behavior: instant ? 'auto' : 'smooth' })
  }, [shell])

  // The primary button (PRIMARY_COPY, shared with the download section and
  // the footer) downloads the Windows installer on Windows and scrolls to
  // #install elsewhere. The label is fixed ("Descargar"); a small second line
  // names the visitor's system once it is known after hydration ("para tu
  // equipo" in the prerender). Every hint is shorter than the label, so the
  // button never changes size (no layout shift). Phones/tablets get "send to my computer", chosen by CSS
  // from <html data-device> (markup below).
  const detectedOS = useDetectedOS()
  const cta = CTA_COPY[lang]

  const es = lang === 'es'
  const copy = es
    ? {
      lastLogin: 'Last login: Mon Jul 14 09:41:07 2026 from 127.0.0.1',
      tag1: '# torre de kongtrol multi-VPN · 10 VPN · Win / macOS / Linux (según la VPN)',
      tag2: '# panel web + CLI completa',
      title: 'Todas tus VPN en un solo lugar.',
      sub: 'Conecta tus VPN como siempre. Kongtrol las detecta, las vigila, las reconecta si caen y manda cada sitio por la VPN correcta.',
      resolving: '→ resolviendo credenciales desde el keychain… ok',
      security: ['⛨ kill switch', 'ARMADO', '· DNS guard', 'ACTIVO'],
      watchdog: '♺ watchdog: sondeo cada 5s · backoff 2s → 5m',
      binHint: `→ para tu sistema: ${fileFor(os)}`,
      binHintMobile: '→ Kongtrol se instala en tu computador: envíate el enlace ↑',
      hint: 'escribe "help" y presiona Enter ↵',
      osHint: { windows: 'para Windows', macos: 'para Mac', linux: 'para Linux' } as Record<OS, string>,
      osHintGeneric: 'para tu equipo',
      facts: ['Gratis', 'Código abierto', 'Sin cuenta', 'Tus datos no salen de tu equipo'],
    }
    : {
      lastLogin: 'Last login: Mon Jul 14 09:41:07 2026 from 127.0.0.1',
      tag1: '# multi-VPN kongtrol tower · 10 VPNs · Win / macOS / Linux (varies by VPN)',
      tag2: '# web panel + full CLI',
      title: 'All your VPNs in one place.',
      sub: 'Connect your VPNs the way you always do. Kongtrol detects them, watches them, reconnects them if they drop, and sends each site through the right VPN.',
      resolving: '→ resolving credentials from the keychain… ok',
      security: ['⛨ kill switch', 'ARMED', '· DNS guard', 'ACTIVE'],
      watchdog: '♺ watchdog: poll every 5s · backoff 2s → 5m',
      binHint: `→ for your system: ${fileFor(os)}`,
      binHintMobile: '→ Kongtrol installs on your computer: send yourself the link ↑',
      hint: 'type "help" and press Enter ↵',
      osHint: { windows: 'for Windows', macos: 'for Mac', linux: 'for Linux' } as Record<OS, string>,
      osHintGeneric: 'for your OS',
      facts: ['Free', 'Open source', 'No account', 'Your data never leaves your computer'],
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
        { cls: 't-ok-line', text: es ? '✔ ayudante: instalado y respondiendo' : '✔ helper: installed and answering' },
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
    <section id="motd" className="section th-section">
      <div className="container">
        <div className="hero-contour th-contour" aria-hidden="true" />
        <div className="terminal crt th-terminal tone-ink">
          <div className="terminal-bar">
            <div className="terminal-dots" aria-hidden="true">
              <div className="terminal-dot red" />
              <div className="terminal-dot yellow" />
              <div className="terminal-dot green" />
            </div>
            <span className="terminal-title">kongtrol — terminal</span>
          </div>
          <div
            className="terminal-body th-body"
            onClick={focusInput}
          >
            {/* Reading order: banner (brand frame) → h1 → the plain sentence →
                buttons → facts. The rest of this window is decorative shell
                output: dimmed and hidden from screen readers. */}
            <div className="boot-line boot-step th-log-line" style={at(BOOT.login)} aria-hidden="true">{copy.lastLogin}</div>

            <pre className="th-banner boot-step" style={at(BOOT.banner)} aria-hidden="true">{BANNER}</pre>

            {/* the page's only h1: visible, never animated (it is the LCP) */}
            <h1 className="th-title">{copy.title}</h1>
            {/* laptops and up: two columns under the h1 — the sentence and
                buttons (th-main) beside the live log and the prompt (th-side),
                so the whole window fits one screen with no inner scroll; on
                narrow screens both wrappers are display: contents (ADR-0060) */}
            <div className="th-main">
            <p className="th-sub boot-step" style={at(BOOT.sub)}>{copy.sub}</p>

            {/* Both variants are always in the markup; CSS shows one from the
                first paint by <html data-device> (set by the boot script in
                seo.ts), so nothing swaps after hydration — no layout shift
                (ADR-0029). The hidden one is display:none: not focusable, not
                announced. The header watches #hero-cta to reveal its own
                download button. */}
            <div id="hero-cta" className="th-ctas boot-step" style={at(BOOT.ctas)}>
              <div className="th-ctas-desktop">
                <a
                  href={primaryDownloadHref(detectedOS, 'home', lang)}
                  className="th-btn primary"
                  download={detectedOS === 'windows' ? '' : undefined}
                >
                  {detectedOS === 'windows' ? <DownloadIcon /> : <ArrowDownIcon />}
                  <span className="th-btn-stack">
                    <span>{PRIMARY_COPY[lang].download}</span>
                    <small className="th-btn-os">{detectedOS ? copy.osHint[detectedOS] : copy.osHintGeneric}</small>
                  </span>
                </a>
                <a href={GITHUB_REPO} className="th-btn" target="_blank" rel="noreferrer">
                  <GitHubIcon />{cta.github}
                </a>
                <DownloadNote lang={lang} os={detectedOS} page="home" className="th-note" />
                {/* what a visitor wants to know before downloading (ADR-0079) */}
                <ul className="th-facts">
                  {copy.facts.map(f => <li key={f}>{f}</li>)}
                </ul>
              </div>
              <SendToComputer lang={lang} className="th-stc th-ctas-mobile">
                <a href={GITHUB_REPO} className="th-btn" target="_blank" rel="noreferrer">
                  <GitHubIcon />{cta.github}
                </a>
              </SendToComputer>
            </div>
            </div>

            <div className="th-side">
            {/* fixed-height scrollback (ADR-0066): it scrolls inside, the prompt
                below stays pinned; role=log announces only new output */}
            <div
              className="th-scroll"
              ref={logRef}
              role="log"
              aria-label={es ? 'salida de la terminal' : 'terminal output'}
              tabIndex={0}
            >
            <div className="th-meta">
              <div className="boot-line boot-step t-dim" style={at(BOOT.tag1)}>{copy.tag1}</div>
              <div className="boot-line boot-step t-dim" style={at(BOOT.tag2)}>{copy.tag2}</div>
            </div>

            <div className="th-log" aria-hidden="true">
            <div className="hero-cmd-line th-cmd boot-step" style={at(BOOT.cmd)}>
              <span className="t-prompt">$ </span>
              <span className="th-typed" style={at(BOOT.cmd, { '--n': CMD.length })}>
                <span className="t-cmd">{CMD}</span>
                <span className="th-typed-cursor" aria-hidden="true" />
              </span>
            </div>

            <div className="boot-line boot-step t-dim" style={at(BOOT.resolving)}>{copy.resolving}</div>

            <BootRows />

            <div className="boot-line boot-security boot-step" style={at(BOOT.security)}>
              {copy.security[0]} <strong>{copy.security[1]}</strong> {copy.security[2]} <strong>{copy.security[3]}</strong>
            </div>
            <div className="boot-line boot-step t-muted" style={at(BOOT.watchdog)}>{copy.watchdog}</div>
            {/* desktop: the file for this OS; phones can't run it (data-device, CSS) */}
            <div className="boot-line boot-step t-dim" style={at(BOOT.bin)}>
              <span className="th-bin-desktop">{copy.binHint}</span>
              <span className="th-bin-mobile">{copy.binHintMobile}</span>
            </div>
            </div>

            {/* visitor shell history — the log region announces it as it arrives */}
            <div className="th-history">
              {shell.map((entry, i) => (
                entry.kind === 'cmd'
                  ? <div key={i} className="hero-cmd-line"><span className="t-prompt">$ </span><span className="t-cmd">{entry.text}</span></div>
                  : <div key={i}>{entry.lines.map((l, j) => <div key={j} className={`boot-line ${l.cls}`}>{l.text}</div>)}</div>
              ))}
            </div>
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
      </div>
    </section>
  )
}
