import { Lang } from '../../i18n'

interface Props {
  lang: Lang
}

type Provider = {
  name: string
  mark: string // two-letter monogram — our own mark, not the provider's logo
  kind: 'privacy' | 'enterprise' | 'open' | 'network'
  attach?: boolean
}

// Attach-capable first: these are the apps people already click "Connect" in.
const PROVIDERS: Provider[] = [
  { name: 'NordVPN', mark: 'Nv', kind: 'privacy', attach: true },
  { name: 'Proton VPN', mark: 'Pv', kind: 'privacy', attach: true },
  { name: 'Surfshark', mark: 'Ss', kind: 'privacy', attach: true },
  { name: 'FortiClient', mark: 'Fc', kind: 'enterprise', attach: true },
  { name: 'WireGuard', mark: 'Wg', kind: 'open' },
  { name: 'OpenVPN', mark: 'Ov', kind: 'open' },
  { name: 'Tailscale', mark: 'Ts', kind: 'network' },
  { name: 'Cloudflare WARP', mark: 'Cw', kind: 'network' },
  { name: 'Cisco AnyConnect', mark: 'Ca', kind: 'enterprise' },
  { name: 'GlobalProtect', mark: 'Gp', kind: 'enterprise' },
]

export default function Providers({ lang }: Props) {
  const es = lang === 'es'
  const t = es
    ? {
      title: 'Funciona con las VPN que ya usas.',
      sub: '10 VPN compatibles. No reemplaza tus apps: las pone a trabajar juntas.',
      attach: 'Conectas en su app y Kongtrol se adjunta',
      kinds: { privacy: 'Privacidad', enterprise: 'Empresa', open: 'Estándar abierto', network: 'Red privada' },
      note: 'Los nombres son marcas de sus respectivos dueños. Kongtrol es un proyecto independiente, sin afiliación con ellos.',
    }
    : {
      title: 'Works with the VPNs you already use.',
      sub: '10 supported VPNs. It does not replace your apps: it makes them work together.',
      attach: 'Connect in its app and Kongtrol attaches',
      kinds: { privacy: 'Privacy', enterprise: 'Enterprise', open: 'Open standard', network: 'Private network' },
      note: 'Names are trademarks of their respective owners. Kongtrol is an independent project, not affiliated with them.',
    }

  return (
    <section id="vpns" className="section providers-section">
      <div className="container">
        <h2 className="section-title">{t.title}</h2>
        <p className="section-sub">{t.sub}</p>

        <p className="providers-legend"><i className="attach-pip" aria-hidden="true" />{t.attach}</p>

        <ul className="provider-grid">
          {PROVIDERS.map(p => (
            <li key={p.name} className={`provider${p.attach ? ' is-attach' : ''}`}>
              <span className="provider-mark" aria-hidden="true">{p.mark}</span>
              <span className="provider-text">
                <span className="provider-name">{p.name}</span>
                <span className="provider-kind">{t.kinds[p.kind]}</span>
              </span>
              {p.attach && (
                <span className="attach-pip" role="img" aria-label={t.attach} title={t.attach} />
              )}
            </li>
          ))}
        </ul>

        <p className="providers-note">{t.note}</p>
      </div>
    </section>
  )
}
