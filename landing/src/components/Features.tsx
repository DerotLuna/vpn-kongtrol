import { useEffect, useState } from 'react'
import { Lang } from '../i18n'

interface Props {
  lang: Lang
}

type Route = {
  dest: string
  rule: string
  match: string
  tunnel: string
  adapter: string
  direct?: boolean
}

// Fixed row centers inside the 640×360 SVG viewBox — no DOM measurement needed.
const ROWS = [64, 141, 219, 296]

export default function Features({ lang }: Props) {
  const [active, setActive] = useState(0)
  const [paused, setPaused] = useState(false)

  const copy = lang === 'es'
    ? {
      label: '$ kongtrol map claude.ai',
      title: 'Un destino. Una regla. El túnel correcto.',
      sub: 'Kongtrol decide en tiempo real por qué túnel sale cada paquete — sin peleas de rutas.',
      boardIn: 'Destino',
      boardCore: 'Motor de políticas',
      boardOut: 'Túnel resuelto',
      readoutRule: 'regla',
      readoutNoRule: 'sin regla — salida directa (sin VPN)',
      hint: 'Toca un destino para ver cómo se resuelve',
      routes: [
        { dest: '10.10.0.0/16', rule: 'Red de oficina', match: 'IP · longest-prefix', tunnel: 'office', adapter: 'FortiClient' },
        { dest: '*.amazonaws.com', rule: 'AWS workloads', match: 'dominio · glob', tunnel: 'aws', adapter: 'OpenVPN' },
        { dest: 'claude.ai', rule: 'Contenido US', match: 'dominio · glob', tunnel: 'us-content', adapter: 'ProtonVPN' },
        { dest: 'gmail.com', rule: '', match: '', tunnel: 'directo', adapter: 'sin VPN', direct: true },
      ] as Route[],
      pillarsTitle: 'Lo que corre detrás',
      pillars: [
        { k: '01', name: 'Ruteo determinista', desc: 'Longest-prefix match + prioridad. Cada dominio e IP resuelve al perfil exacto, siempre igual.', metric: 'LPM + PRIORIDAD' },
        { k: '02', name: 'Seguridad sincronizada', desc: 'Kill switch y DNS guard se ajustan al estado real de los túneles activos. Sin fugas al caer.', metric: 'KILL SWITCH · DNS GUARD' },
        { k: '03', name: 'Autorreparación', desc: 'El watchdog sondea cada 5 s y reconecta con backoff exponencial de 2 s hasta 5 min.', metric: 'WATCHDOG 5s · 2s→5m' },
        { k: '04', name: 'Observabilidad viva', desc: 'Métricas por WebSocket cada segundo, API REST y bitácora de auditoría firmada con HMAC.', metric: 'WS 1s · REST · HMAC' },
      ],
      adaptersTitle: 'Ocho clientes. Un solo contrato de kongtrol.',
      adaptersSub: 'No reemplaza tus VPN — las orquesta con política, seguridad y observabilidad.',
    }
    : {
      label: '$ kongtrol map claude.ai',
      title: 'One destination. One rule. The right tunnel.',
      sub: 'Kongtrol decides in real time which tunnel each packet takes — no route fights.',
      boardIn: 'Destination',
      boardCore: 'Policy engine',
      boardOut: 'Resolved tunnel',
      readoutRule: 'rule',
      readoutNoRule: 'no rule — direct egress (no VPN)',
      hint: 'Tap a destination to see how it resolves',
      routes: [
        { dest: '10.10.0.0/16', rule: 'Office network', match: 'IP · longest-prefix', tunnel: 'office', adapter: 'FortiClient' },
        { dest: '*.amazonaws.com', rule: 'AWS workloads', match: 'domain · glob', tunnel: 'aws', adapter: 'OpenVPN' },
        { dest: 'claude.ai', rule: 'US content', match: 'domain · glob', tunnel: 'us-content', adapter: 'ProtonVPN' },
        { dest: 'gmail.com', rule: '', match: '', tunnel: 'direct', adapter: 'no VPN', direct: true },
      ] as Route[],
      pillarsTitle: 'What runs underneath',
      pillars: [
        { k: '01', name: 'Deterministic routing', desc: 'Longest-prefix match plus priority. Every domain and IP resolves to the exact profile, every time.', metric: 'LPM + PRIORITY' },
        { k: '02', name: 'Synchronized security', desc: 'Kill switch and DNS guard track live tunnel state. No leaks the moment a tunnel drops.', metric: 'KILL SWITCH · DNS GUARD' },
        { k: '03', name: 'Self-healing', desc: 'The watchdog polls every 5 s and reconnects with exponential backoff from 2 s up to 5 min.', metric: 'WATCHDOG 5s · 2s→5m' },
        { k: '04', name: 'Live observability', desc: 'Per-second WebSocket metrics, a REST API, and an HMAC-signed audit log.', metric: 'WS 1s · REST · HMAC' },
      ],
      adaptersTitle: 'Eight clients. One kongtrol contract.',
      adaptersSub: 'Not a replacement for your VPN clients — Kongtrol orchestrates them with policy, security, and observability.',
    }

  const ADAPTERS = [
    { name: 'OpenVPN', tag: 'Stable' },
    { name: 'WireGuard', tag: 'Recommended' },
    { name: 'FortiClient', tag: 'Enterprise' },
    { name: 'GlobalProtect', tag: 'Enterprise' },
    { name: 'Cisco AnyConnect', tag: 'Enterprise' },
    { name: 'Tailscale', tag: 'Mesh' },
    { name: 'ProtonVPN', tag: 'Privacy' },
    { name: 'Cloudflare WARP', tag: 'Edge' },
  ]

  useEffect(() => {
    if (paused) return
    const t = window.setInterval(() => setActive(a => (a + 1) % ROWS.length), 2600)
    return () => window.clearInterval(t)
  }, [paused])

  const routes = copy.routes
  const cur = routes[active]

  return (
    <section id="map" className="section features-section">
      <div className="container">
        <div className="section-label cmd-label">{copy.label}</div>
        <h2 className="section-title">{copy.title}</h2>
        <p className="section-sub">{copy.sub}</p>

        {/* ── SIGNATURE: live routing switchboard ── */}
        <div
          className="switchboard"
          onMouseEnter={() => setPaused(true)}
          onMouseLeave={() => setPaused(false)}
        >
          <div className="sb-legend">
            <span className="sb-legend-item in">{copy.boardIn}</span>
            <span className="sb-legend-item core">{copy.boardCore}</span>
            <span className="sb-legend-item out">{copy.boardOut}</span>
          </div>

          <svg className="sb-svg" viewBox="0 0 640 360" role="img" aria-label={copy.boardCore}>
            {/* wires — behind nodes */}
            {routes.map((r, i) => {
              const y = ROWS[i]
              const on = i === active
              return (
                <g key={`w-${i}`} className={`sb-wire-group${on ? ' active' : ''}${r.direct ? ' direct' : ''}`}>
                  <path className="sb-wire" d={`M156 ${y} C 214 ${y}, 214 180, 270 180`} />
                  <path className="sb-wire" d={`M370 180 C 426 180, 426 ${y}, 484 ${y}`} />
                </g>
              )
            })}

            {/* core — the Kong mark has one continuous ambient pulse, unaffected by route changes */}
            <g className="sb-core" transform="translate(320 180)">
              <circle className="sb-glow" r="50" />
              <circle className="sb-ring" r="56" />
              <image className="sb-logo" href={`${import.meta.env.BASE_URL}logo-kong.svg`} x="-42" y="-42" width="84" height="84" />
            </g>

            {/* destination nodes (left) */}
            {routes.map((r, i) => {
              const y = ROWS[i]
              const on = i === active
              return (
                <g
                  key={`in-${i}`}
                  className={`sb-node in${on ? ' active' : ''}`}
                  onClick={() => setActive(i)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setActive(i) } }}
                >
                  <rect x="6" y={y - 18} width="150" height="36" rx="6" />
                  <text x="20" y={y + 4}>{r.dest}</text>
                </g>
              )
            })}

            {/* tunnel nodes (right) */}
            {routes.map((r, i) => {
              const y = ROWS[i]
              const on = i === active
              return (
                <g key={`out-${i}`} className={`sb-node out${on ? ' active' : ''}${r.direct ? ' direct' : ''}`}>
                  <rect x="484" y={y - 20} width="150" height="40" rx="6" />
                  <text className="sb-out-name" x="498" y={y - 2}>{r.tunnel}</text>
                  <text className="sb-out-sub" x="498" y={y + 13}>{r.adapter}</text>
                </g>
              )
            })}
          </svg>

          <div className="sb-readout" key={active} aria-live="polite">
            <span className="sb-readout-dest">{cur.dest}</span>
            <svg className="sb-arrow" width="16" height="10" viewBox="0 0 16 10" aria-hidden="true"><path d="M0 5h13M9 1l4 4-4 4" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/></svg>
            {cur.direct ? (
              <span className="sb-readout-mid">{copy.readoutNoRule}</span>
            ) : (
              <span className="sb-readout-mid">
                {copy.readoutRule} «<b>{cur.rule}</b>» · {cur.match}
              </span>
            )}
            <svg className="sb-arrow" width="16" height="10" viewBox="0 0 16 10" aria-hidden="true"><path d="M0 5h13M9 1l4 4-4 4" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/></svg>
            <span className={`sb-readout-tunnel${cur.direct ? ' direct' : ''}`}>{cur.tunnel}</span>
          </div>
          <p className="sb-hint">{copy.hint}</p>
        </div>

        {/* ── Capability pillars ── */}
        <div className="pillars-head">{copy.pillarsTitle}</div>
        <div className="pillars">
          {copy.pillars.map(p => (
            <article key={p.k} className="pillar reveal">
              <span className="pillar-glyph" aria-hidden="true" />
              <span className="pillar-k">{p.k}</span>
              <h3>{p.name}</h3>
              <p>{p.desc}</p>
              <span className="pillar-metric">{p.metric}</span>
            </article>
          ))}
        </div>

        {/* ── Adapters ── */}
        <div className="adapters">
          <div className="adapters-head">
            <h3>{copy.adaptersTitle}</h3>
            <p>{copy.adaptersSub}</p>
          </div>
          <div className="adapter-grid">
            {ADAPTERS.map(a => (
              <div key={a.name} className="adapter-chip reveal">
                <span className="adapter-name">{a.name}</span>
                <span className="adapter-tag">{a.tag}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
