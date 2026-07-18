import { Lang } from '../i18n'

interface Props {
  lang: Lang
}

export default function Security({ lang }: Props) {
  const copy = lang === 'es'
    ? {
      label: '$ kongtrol check',
      title: 'Si un túnel cae, nada se fuga.',
      sub: 'Kongtrol arma y desarma cada defensa según el estado de tus túneles — no gestionas firewalls tú.',
      defenses: [
        { name: 'Kill switch', desc: 'Firewall a nivel de OS (WFP / iptables / pf). Modo strict bloquea todo; loose permite LAN. Configurable por perfil.', spec: 'netsh · iptables · pf' },
        { name: 'DNS guard', desc: 'Fuerza el DNS por el túnel activo y restaura el original al desconectar. Con conteo de referencias entre túneles simultáneos.', spec: 'ref-counted' },
        { name: 'Leak tester', desc: 'Verifica fugas de IP y DNS cada N segundos. Al detectar una: notifica, o activa el kill switch y notifica.', spec: 'IP + DNS' },
        { name: 'Audit log firmado', desc: 'Bitácora append-only con HMAC-SHA256 por entrada. La llave de firma vive en el keychain, no en disco.', spec: 'HMAC-SHA256' },
        { name: 'Credenciales en keychain', desc: 'Windows Credential Manager, macOS Keychain, libsecret en Linux. Las contraseñas jamás tocan el YAML.', spec: '0 secretos en disco' },
      ],
      termTitle: 'kongtrol check',
      termLines: [
        { t: 'cmd', v: 'kongtrol check' },
        { t: 'ok', v: '✔ ip pública sale por el túnel esperado' },
        { t: 'ok', v: '✔ dns resuelve por 10.10.0.1 (túnel office)' },
        { t: 'ok', v: '✔ kill switch armado — 0 rutas fuera de política' },
        { t: 'ok', v: '✔ audit.log íntegro — 1,204 entradas firmadas' },
        { t: 'dim', v: 'sin fugas detectadas · exit 0' },
      ],
    }
    : {
      label: '$ kongtrol check',
      title: 'If a tunnel drops, nothing leaks.',
      sub: 'Kongtrol arms and disarms every defense based on live tunnel state — you never touch firewalls.',
      defenses: [
        { name: 'Kill switch', desc: 'OS-level firewall (WFP / iptables / pf). Strict mode blocks everything; loose allows LAN. Configurable per profile.', spec: 'netsh · iptables · pf' },
        { name: 'DNS guard', desc: 'Forces DNS through the active tunnel and restores the original on disconnect. Reference-counted across simultaneous tunnels.', spec: 'ref-counted' },
        { name: 'Leak tester', desc: 'Checks for IP and DNS leaks every N seconds. On detection: notify, or arm the kill switch and notify.', spec: 'IP + DNS' },
        { name: 'Signed audit log', desc: 'Append-only log with HMAC-SHA256 per entry. The signing key lives in the keychain, not on disk.', spec: 'HMAC-SHA256' },
        { name: 'Keychain credentials', desc: 'Windows Credential Manager, macOS Keychain, libsecret on Linux. Passwords never touch the YAML.', spec: '0 secrets on disk' },
      ],
      termTitle: 'kongtrol check',
      termLines: [
        { t: 'cmd', v: 'kongtrol check' },
        { t: 'ok', v: '✔ public ip egresses through the expected tunnel' },
        { t: 'ok', v: '✔ dns resolves via 10.10.0.1 (office tunnel)' },
        { t: 'ok', v: '✔ kill switch armed — 0 routes outside policy' },
        { t: 'ok', v: '✔ audit.log intact — 1,204 signed entries' },
        { t: 'dim', v: 'no leaks detected · exit 0' },
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
                <div className="defense-head">
                  <h3>{d.name}</h3>
                  <span className="defense-spec">{d.spec}</span>
                </div>
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
