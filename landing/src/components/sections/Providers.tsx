// VPN section of the landing — version 2.2.0 · 2026-11-18.
// Two groups (attach vs direct, from internal/vpn/* — the attach group is the
// adapters built on internal/vpn/appattach) and per-OS support checked against
// each adapter's platform files. No third-party logos (ADR-0023): every VPN
// gets our own monogram tile.
import { Lang } from '../../i18n'
import './providers.css'

interface Props {
  lang: Lang
}

// yes: works (direct group). auto: Kongtrol connects through the official
// CLI. manual: you connect in the app, Kongtrol attaches. partial: works with
// a limit (see the group note).
type Support = 'yes' | 'auto' | 'manual' | 'partial'
type OS = 'win' | 'mac' | 'linux'

type Provider = {
  name: string
  mark: string // two-letter monogram — our own mark, not the provider's logo
  kind: 'privacy' | 'enterprise' | 'open' | 'network'
  os: Partial<Record<OS, Support>> // missing OS = not supported
}

// Attach group: provider adapters on internal/vpn/appattach.
// NordVPN: official CLI on Windows + Linux (platform_windows.go / platform_linux.go),
// attach only on macOS (platform_other.go). Proton: CLI on Linux only
// (platform_linux.go). Surfshark: no CLI, AutoConnect nil (cli.go).
// FortiClient: Windows strategies per ADR-0020. macOS/Linux are still "no
// verificado" in docs/forticlient-support.md (the technical truth); the owner
// decided to list them as plain chips, with no qualifier, and take problems
// as GitHub issues.
const ATTACH: Provider[] = [
  { name: 'NordVPN', mark: 'Nv', kind: 'privacy', os: { win: 'auto', mac: 'manual', linux: 'auto' } },
  { name: 'Proton VPN', mark: 'Pv', kind: 'privacy', os: { win: 'manual', mac: 'manual', linux: 'auto' } },
  { name: 'Surfshark', mark: 'Ss', kind: 'privacy', os: { win: 'manual', mac: 'manual', linux: 'manual' } },
  { name: 'FortiClient', mark: 'Fc', kind: 'enterprise', os: { win: 'auto', mac: 'yes', linux: 'yes' } },
]

// Direct group: Kongtrol brings the tunnel up itself (WireGuard, OpenVPN) or
// drives the vendor CLI. OpenVPN on macOS without admin needs a bundled signed
// openvpn (helper refuses Homebrew binaries — AGENTS.md). GlobalProtect has no
// Linux client (globalprotect/cli.go ErrLinux).
const DIRECT: Provider[] = [
  { name: 'WireGuard', mark: 'Wg', kind: 'open', os: { win: 'yes', mac: 'yes', linux: 'yes' } },
  { name: 'OpenVPN', mark: 'Ov', kind: 'open', os: { win: 'yes', mac: 'partial', linux: 'yes' } },
  { name: 'Tailscale', mark: 'Ts', kind: 'network', os: { win: 'yes', mac: 'yes', linux: 'yes' } },
  { name: 'Cloudflare WARP', mark: 'Cw', kind: 'network', os: { win: 'yes', mac: 'yes', linux: 'yes' } },
  { name: 'Cisco AnyConnect', mark: 'Ca', kind: 'enterprise', os: { win: 'yes', mac: 'yes', linux: 'yes' } },
  { name: 'GlobalProtect', mark: 'Gp', kind: 'enterprise', os: { win: 'yes', mac: 'yes' } },
]

const OS_ORDER: OS[] = ['win', 'mac', 'linux']
const OS_NAME: Record<OS, string> = { win: 'Windows', mac: 'macOS', linux: 'Linux' }

export default function Providers({ lang }: Props) {
  const es = lang === 'es'
  const t = es
    ? {
      title: 'Funciona con las VPN que ya usas.',
      sub: '10 VPN compatibles. No reemplaza tus apps: las pone a trabajar juntas.',
      attachTitle: 'Abres tu app y Kongtrol se adjunta',
      attachDesc: 'Conectas en la app oficial, como siempre. Kongtrol detecta el túnel y lo vigila. Donde hay línea de comandos oficial, también conecta solo.',
      directTitle: 'Kongtrol conecta directo',
      directDesc: 'Kongtrol levanta el túnel por ti, con tu archivo de configuración o el cliente oficial.',
      osLabel: 'Sistemas',
      state: { yes: '', auto: 'auto', manual: 'manual', partial: 'parcial' },
      stateLong: {
        yes: 'compatible',
        auto: 'Kongtrol conecta solo',
        manual: 'conectas en la app y Kongtrol se adjunta',
        partial: 'con límites',
      },
      attachKey: 'auto: Kongtrol conecta solo · manual: conectas en la app y Kongtrol se adjunta.',
      directKey: 'parcial: OpenVPN en macOS necesita, por ahora, ejecutarse como administrador.',
      kinds: { privacy: 'Privacidad', enterprise: 'Empresa', open: 'Estándar abierto', network: 'Red privada' },
      note: 'Los nombres son marcas de sus respectivos dueños. Kongtrol es un proyecto independiente, sin afiliación con ellos.',
    }
    : {
      title: 'Works with the VPNs you already use.',
      sub: '10 supported VPNs. It does not replace your apps: it makes them work together.',
      attachTitle: 'Open your app and Kongtrol attaches',
      attachDesc: 'Connect in the official app, as usual. Kongtrol detects the tunnel and watches it. Where there is an official command line, it also connects on its own.',
      directTitle: 'Kongtrol connects directly',
      directDesc: 'Kongtrol brings the tunnel up for you, from your config file or the official client.',
      osLabel: 'Systems',
      state: { yes: '', auto: 'auto', manual: 'manual', partial: 'partial' },
      stateLong: {
        yes: 'supported',
        auto: 'Kongtrol connects on its own',
        manual: 'connect in the app and Kongtrol attaches',
        partial: 'with limits',
      },
      attachKey: 'auto: Kongtrol connects on its own · manual: connect in the app and Kongtrol attaches.',
      directKey: 'partial: OpenVPN on macOS needs, for now, to run as administrator.',
      kinds: { privacy: 'Privacy', enterprise: 'Enterprise', open: 'Open standard', network: 'Private network' },
      note: 'Names are trademarks of their respective owners. Kongtrol is an independent project, not affiliated with them.',
    }

  const card = (p: Provider) => (
    <li key={p.name} className="provider">
      <span className="provider-mark" aria-hidden="true">
        {p.mark}
      </span>
      <span className="provider-text">
        <span className="provider-name">{p.name}</span>
        <span className="provider-kind">{t.kinds[p.kind]}</span>
      </span>
      <ul className="provider-os" aria-label={t.osLabel}>
        {OS_ORDER.filter(o => p.os[o]).map(o => {
          const s = p.os[o] as Support
          return (
            <li key={o} className={`os is-${s}`}>
              {OS_NAME[o]}
              {s !== 'yes' && <span className="os-state" aria-hidden="true">{t.state[s]}</span>}
              {s !== 'yes' && <span className="sr-only">: {t.stateLong[s]}</span>}
            </li>
          )
        })}
      </ul>
    </li>
  )

  return (
    <section id="vpns" className="section providers-section">
      <div className="container">
        <h2 className="section-title">{t.title}</h2>
        <p className="section-sub">{t.sub}</p>

        {/* feature:providers */}
        <div className="providers-group" aria-labelledby="vpns-attach">
          <h3 id="vpns-attach" className="providers-group-title">{t.attachTitle}</h3>
          <p className="providers-group-desc">{t.attachDesc}</p>
          <ul className="provider-grid is-attach">{ATTACH.map(card)}</ul>
          <p className="providers-key">{t.attachKey}</p>
        </div>

        <div className="providers-group" aria-labelledby="vpns-direct">
          <h3 id="vpns-direct" className="providers-group-title">{t.directTitle}</h3>
          <p className="providers-group-desc">{t.directDesc}</p>
          <ul className="provider-grid is-direct">{DIRECT.map(card)}</ul>
          <p className="providers-key">{t.directKey}</p>
        </div>

        <p className="providers-note">{t.note}</p>
      </div>
    </section>
  )
}
