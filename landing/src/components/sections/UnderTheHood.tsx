import { Lang } from '../../i18n'

interface Props {
  lang: Lang
}

// The deep tech, kept for the people who want it and out of everyone
// else's way. Rendered as a `--verbose` dump: one row per subsystem.
export default function UnderTheHood({ lang }: Props) {
  const es = lang === 'es'
  const t = es
    ? {
      label: '$ kongtrol --verbose',
      title: 'Detalles técnicos',
      sub: 'Para quien quiere saber exactamente cómo funciona.',
      rows: [
        { k: 'Ruteo determinista', v: 'Longest-prefix match para IP y globs para dominios, con prioridad para resolver conflictos. El mismo destino siempre resuelve al mismo perfil.', m: 'LPM + prioridad' },
        { k: 'Watchdog', v: 'Sondea cada túnel cada 5 s y reconecta con backoff exponencial de 2 s hasta 5 min. Una desconexión intencional no dispara la reconexión.', m: '5s · 2s → 5m' },
        { k: 'Kill switch', v: 'Firewall del sistema: reglas de Windows Firewall (netsh), iptables en Linux, pf en macOS. Si un túnel protegido cae sin que lo apagues, bloquea sus rutas y destinos; todo el tráfico solo en perfiles de túnel completo (allow_lan deja pasar la red local).', m: 'netsh · iptables · pf' },
        { k: 'DNS guard', v: 'Con conteo de referencias entre túneles simultáneos: une los DNS de los perfiles activos y restaura el original cuando cae el último.', m: 'ref-counted' },
        { k: 'Audit log', v: 'Append-only, con HMAC-SHA256 por entrada. La llave de firma vive en el llavero del sistema, no en disco.', m: 'HMAC-SHA256' },
        { k: 'Credenciales', v: 'Windows Credential Manager, macOS Keychain o libsecret. Se leen justo antes de conectar y esa copia se pone en cero al terminar de conectar. Nunca se guardan en archivos.', m: '0 secretos en disco' },
        { k: 'Ayudante con privilegios', v: 'kongtrol service install, una sola vez. Después ni la app ni la CLI necesitan una terminal de administrador.', m: '1 permiso' },
        { k: 'Observabilidad', v: 'Métricas en vivo por WebSocket cada 5 s y al instante cuando algo cambia, API REST local y el panel web embebido en el binario. Sin servidor externo.', m: 'WS 5s · REST' },
        { k: 'CLI con paridad total', v: 'Todo lo que hace el panel lo hace la CLI: perfiles, grupos, reglas y seguridad. Lista para scripts y automatización.', m: 'init · up · status' },
      ],
    }
    : {
      label: '$ kongtrol --verbose',
      title: 'Technical details',
      sub: 'For anyone who wants to know exactly how it works.',
      rows: [
        { k: 'Deterministic routing', v: 'Longest-prefix match for IPs and globs for domains, with priority to settle conflicts. The same destination always resolves to the same profile.', m: 'LPM + priority' },
        { k: 'Watchdog', v: 'Polls every tunnel every 5 s and reconnects with exponential backoff from 2 s up to 5 min. An intentional disconnect never triggers a reconnect.', m: '5s · 2s → 5m' },
        { k: 'Kill switch', v: 'The OS firewall: Windows Firewall rules (netsh), iptables on Linux, pf on macOS. If a protected tunnel drops without you turning it off, it blocks its routes and destinations; all traffic only for full-tunnel profiles (allow_lan lets the local network through).', m: 'netsh · iptables · pf' },
        { k: 'DNS guard', v: 'Reference-counted across simultaneous tunnels: merges the DNS servers of every active profile and restores the original when the last one drops.', m: 'ref-counted' },
        { k: 'Audit log', v: 'Append-only, with HMAC-SHA256 per entry. The signing key lives in the system keychain, not on disk.', m: 'HMAC-SHA256' },
        { k: 'Credentials', v: 'Windows Credential Manager, macOS Keychain or libsecret. Read right before connecting; that copy is zeroed once connecting finishes. Never stored in files.', m: '0 secrets on disk' },
        { k: 'Privileged helper', v: 'kongtrol service install, once. After that neither the app nor the CLI needs an administrator terminal.', m: '1 prompt' },
        { k: 'Observability', v: 'Live metrics over WebSocket every 5 s and instantly on change, a local REST API, and the web panel embedded in the binary. No external server.', m: 'WS 5s · REST' },
        { k: 'Full CLI parity', v: 'Everything the panel does, the CLI does: profiles, groups, rules and security. Ready for scripts and automation.', m: 'init · up · status' },
      ],
    }

  return (
    <section id="hood" className="section hood-section tone-ink">
      <div className="container">
        <div className="section-label cmd-label">{t.label}</div>
        <h2 className="section-title">{t.title}</h2>
        <p className="section-sub">{t.sub}</p>

        <dl className="hood-list">
          {t.rows.map(r => (
            <div key={r.k} className="hood-row reveal">
              <dt>{r.k}</dt>
              <dd className="hood-desc">{r.v}</dd>
              <dd className="hood-metric">{r.m}</dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  )
}
