// VPN section of the landing — version 2.3.0 · 2026-11-18.
// Two groups (attach vs direct, from internal/vpn/* — the attach group is the
// adapters built on internal/vpn/appattach) and per-OS support checked against
// each adapter's platform files. No third-party logos (ADR-0023): every VPN
// gets our own monogram tile.
import { Lang } from '../../i18n'
import { pathFor } from '../../site'
import './providers.css'

interface Props {
  lang: Lang
}

// yes: works (direct group). auto: Kongtrol connects through the official
// CLI. manual: you connect in the app, Kongtrol attaches. partial: works with
// a limit (see the group note).
type Support = 'yes' | 'auto' | 'manual' | 'partial'
type OS = 'win' | 'mac' | 'linux'
type LegendItem = { s: Exclude<Support, 'yes'>; text: string }

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
  { name: 'Mullvad VPN', mark: 'Mv', kind: 'privacy', os: { win: 'auto', mac: 'partial', linux: 'auto' } },
  { name: 'Private Internet Access', mark: 'Pa', kind: 'privacy', os: { win: 'auto', mac: 'partial', linux: 'auto' } },
  { name: 'ExpressVPN', mark: 'Ev', kind: 'privacy', os: { win: 'manual', mac: 'partial', linux: 'auto' } },
  { name: 'Windscribe', mark: 'Ws', kind: 'privacy', os: { win: 'auto', mac: 'partial', linux: 'auto' } },
  { name: 'CyberGhost VPN', mark: 'Cg', kind: 'privacy', os: { win: 'manual', mac: 'partial', linux: 'partial' } },
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
      sub: 'Usa Kongtrol con estos clientes y protocolos.',
      details: 'Ver compatibilidad por sistema',
      guide: 'Consulta los requisitos en la Guía',
      attachTitle: 'Abres tu app y Kongtrol se adjunta',
      attachDesc: 'Conectas en la app oficial, como siempre. Kongtrol detecta el túnel y lo vigila. Donde el proveedor lo permite, también la conecta por ti.',
      directTitle: 'Kongtrol conecta directo',
      directDesc: 'Kongtrol levanta el túnel por ti, con tu archivo de configuración o el cliente oficial.',
      osLabel: 'Sistemas',
      state: { yes: '', auto: 'auto', manual: 'manual', partial: 'parcial' },
      stateLong: {
        yes: 'compatible',
        auto: 'Kongtrol la conecta por ti',
        manual: 'la conectas tú en su app y Kongtrol la vigila',
        partial: 'con límites',
      },
      legendLabel: 'Qué significa cada etiqueta',
      attachKey: [
        { s: 'auto', text: 'Kongtrol la conecta por ti.' },
        { s: 'manual', text: 'La conectas tú en su app, como siempre; Kongtrol la detecta y la vigila.' },
      ] as LegendItem[],
      directKey: [
        { s: 'partial', text: 'OpenVPN en macOS necesita, por ahora, ejecutarse como administrador.' },
      ] as LegendItem[],
      kinds: { privacy: 'Privacidad', enterprise: 'Empresa', open: 'Estándar abierto', network: 'Red privada' },
      note: 'Proyecto independiente. Las marcas pertenecen a sus titulares.',
    }
    : {
      title: 'Works with the VPNs you already use.',
      sub: 'Use Kongtrol with these clients and protocols.',
      details: 'Check compatibility by operating system',
      guide: 'See requirements in the guide',
      attachTitle: 'Open your app and Kongtrol attaches',
      attachDesc: 'Connect in the official app, as usual. Kongtrol detects the tunnel and watches it. Where the provider allows it, it also connects it for you.',
      directTitle: 'Kongtrol connects directly',
      directDesc: 'Kongtrol brings the tunnel up for you, from your config file or the official client.',
      osLabel: 'Systems',
      state: { yes: '', auto: 'auto', manual: 'manual', partial: 'partial' },
      stateLong: {
        yes: 'supported',
        auto: 'Kongtrol connects it for you',
        manual: 'you connect it in its app and Kongtrol watches it',
        partial: 'with limits',
      },
      legendLabel: 'What each tag means',
      attachKey: [
        { s: 'auto', text: 'Kongtrol connects it for you.' },
        { s: 'manual', text: 'You connect it in its app, as usual; Kongtrol detects it and watches it.' },
      ] as LegendItem[],
      directKey: [
        { s: 'partial', text: 'OpenVPN on macOS needs, for now, to run as administrator.' },
      ] as LegendItem[],
      kinds: { privacy: 'Privacy', enterprise: 'Enterprise', open: 'Open standard', network: 'Private network' },
      note: 'Independent project. Trademarks belong to their owners.',
    }

  // The legend sits above the cards it explains, so each tag is known before
  // it is read. Every entry shows the same chip next to plain words.
  const legend = (items: LegendItem[]) => (
    <dl className="providers-legend" aria-label={t.legendLabel}>
      {items.map(k => (
        <div key={k.s} className="providers-legend-item">
          <dt><span className={`os is-${k.s}`}><span className="os-state">{t.state[k.s]}</span></span></dt>
          <dd>{k.text}</dd>
        </div>
      ))}
    </dl>
  )

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
        <p className="provider-names">{[...ATTACH, ...DIRECT].map(p => p.name).join(' · ')}</p>

        {/* feature:providers */}
        <details className="providers-more">
          <summary>{t.details}</summary>
          <div className="providers-group" aria-labelledby="vpns-attach">
            <h3 id="vpns-attach" className="providers-group-title">{t.attachTitle}</h3>
            <p className="providers-group-desc">{t.attachDesc}</p>
            {legend(t.attachKey)}
            <ul className="provider-grid is-attach">{ATTACH.map(card)}</ul>
          </div>

          <div className="providers-group" aria-labelledby="vpns-direct">
            <h3 id="vpns-direct" className="providers-group-title">{t.directTitle}</h3>
            <p className="providers-group-desc">{t.directDesc}</p>
            {legend(t.directKey)}
            <ul className="provider-grid is-direct">{DIRECT.map(card)}</ul>
          </div>
        </details>

        <p className="providers-note">{t.note} <a href={`${pathFor('guia', lang)}#prereqs`}>{t.guide}</a></p>
      </div>
    </section>
  )
}
