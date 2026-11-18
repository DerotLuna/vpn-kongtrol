import { Lang } from '../../i18n'

interface Props {
  lang: Lang
}

export default function Security({ lang }: Props) {
  const copy = lang === 'es'
    ? {
      label: '$ kongtrol ip',
      title: 'Si un túnel cae, nada se fuga.',
      sub: 'Kongtrol activa cada protección según el estado real de tus VPN. No tienes que tocar el firewall.',
      defenses: [
        { name: 'Kill switch', desc: 'Si una VPN cae, bloquea el tráfico para que nada salga sin protección. Lo activas o lo apagas desde el panel o con kongtrol config killswitch.' },
        { name: 'Protección de DNS', desc: 'Tus consultas DNS van por la VPN activa y vuelven solas a la normalidad al desconectar.' },
        { name: 'Detector de fugas (opcional)', desc: 'Si lo activas, consulta tu IP pública al conectar o desconectar y cuando lo pides, y te avisa si no es la IP que configuraste como esperada para esa VPN. Viene apagado: usa un servicio externo que ve tu IP.' },
        { name: 'Registro firmado', desc: 'Cada evento queda anotado y firmado. Si alguien lo altera, se nota.' },
        { name: 'Contraseñas en el llavero', desc: 'Se guardan en el llavero de tu sistema, nunca en archivos de configuración.' },
      ],
      termTitle: 'kongtrol ip',
      termLines: [
        { t: 'cmd', v: 'kongtrol ip' },
        { t: 'ok', v: '✔ No se detectaron fugas  IP pública: 203.0.113.7' },
        { t: 'dim', v: 'Usa un servicio externo (api.ipify.org) que ve tu IP pública' },
      ],
    }
    : {
      label: '$ kongtrol ip',
      title: 'If a tunnel drops, nothing leaks.',
      sub: 'Kongtrol turns each protection on and off based on the real state of your VPNs. You never touch the firewall.',
      defenses: [
        { name: 'Kill switch', desc: 'If a VPN drops, it blocks traffic so nothing leaves unprotected. Turn it on or off from the panel or with kongtrol config killswitch.' },
        { name: 'DNS protection', desc: 'Your DNS lookups go through the active VPN and return to normal on their own when you disconnect.' },
        { name: 'Leak detector (optional)', desc: 'If you turn it on, it looks up your public IP when you connect or disconnect and when you ask, and warns you if it is not the IP you set as expected for that VPN. Off by default: it uses an external service that sees your IP.' },
        { name: 'Signed log', desc: 'Every event is recorded and signed. If anyone tampers with it, it shows.' },
        { name: 'Passwords in the keychain', desc: 'Stored in your system keychain, never in configuration files.' },
      ],
      termTitle: 'kongtrol ip',
      termLines: [
        { t: 'cmd', v: 'kongtrol ip' },
        { t: 'ok', v: '✔ No leak detected  Public IP: 203.0.113.7' },
        { t: 'dim', v: 'Uses an external service (api.ipify.org) that sees your public IP' },
      ],
    }

  return (
    <section id="check" className="section security-section">
      <div className="container">
        <div className="section-label cmd-label">{copy.label}</div>
        <h2 className="section-title">{copy.title}</h2>
        <p className="section-sub">{copy.sub}</p>

        <div className="security-grid">
          <div className="defense-list">
            {copy.defenses.map(d => (
              <article key={d.name} className="defense reveal">
                <h3>{d.name}</h3>
                <p>{d.desc}</p>
              </article>
            ))}
          </div>

          <div className="security-terminal reveal">
            <div className="terminal crt">
              <div className="terminal-bar">
                <div className="terminal-dots">
                  <div className="terminal-dot red" />
                  <div className="terminal-dot yellow" />
                  <div className="terminal-dot green" />
                </div>
                <span className="terminal-title">{copy.termTitle}</span>
              </div>
              <div className="terminal-body" style={{ minHeight: 0 }}>
                {copy.termLines.map((l, i) => (
                  l.t === 'cmd'
                    ? <div key={i} className="hero-cmd-line"><span className="t-prompt">$ </span><span className="t-cmd">{l.v}</span></div>
                    : <div key={i} className={`boot-line ${l.t === 'ok' ? 't-ok-line' : 't-dim'}`}>{l.v}</div>
                ))}
                <div className="hero-cmd-line" style={{ marginTop: 8 }}>
                  <span className="t-prompt">$ </span><span className="t-cursor" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
