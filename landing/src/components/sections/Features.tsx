import { useEffect, useState } from 'react'
import { Lang } from '../../i18n'

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
  // once the visitor picks a destination the board stops rotating for good
  // (WCAG 2.2.2) and the readout starts being announced
  const [touched, setTouched] = useState(false)

  const copy = lang === 'es'
    ? {
      label: '$ kongtrol map claude.ai',
      title: 'Cada sitio, por la VPN correcta.',
      sub: 'Tú eliges qué direcciones y qué dominios van por cada VPN. Lo demás sale directo, sin VPN. Varias VPN a la vez, sin pelearse.',
      boardIn: 'Destino',
      boardCore: 'Reglas de Kongtrol',
      boardOut: 'Sale por',
      readoutRule: 'regla',
      readoutNoRule: 'sin regla — sale directo, sin VPN',
      hint: 'Toca un destino para ver cómo se resuelve',
      routes: [
        { dest: '10.10.0.0/16', rule: 'Red de oficina', match: 'rango de IP', tunnel: 'office', adapter: 'FortiClient' },
        { dest: '*.amazonaws.com', rule: 'AWS workloads', match: 'dominio', tunnel: 'aws', adapter: 'OpenVPN' },
        { dest: 'claude.ai', rule: 'Contenido US', match: 'dominio', tunnel: 'us-content', adapter: 'Proton VPN' },
        { dest: 'gmail.com', rule: '', match: '', tunnel: 'directo', adapter: 'sin VPN', direct: true },
      ] as Route[],
    }
    : {
      label: '$ kongtrol map claude.ai',
      title: 'Every site, through the right VPN.',
      sub: 'You choose which addresses and domains go through each VPN. Everything else goes out directly, no VPN. Several VPNs at once, no fighting.',
      boardIn: 'Destination',
      boardCore: 'Kongtrol rules',
      boardOut: 'Goes through',
      readoutRule: 'rule',
      readoutNoRule: 'no rule — goes out directly, no VPN',
      hint: 'Tap a destination to see how it resolves',
      routes: [
        { dest: '10.10.0.0/16', rule: 'Office network', match: 'IP range', tunnel: 'office', adapter: 'FortiClient' },
        { dest: '*.amazonaws.com', rule: 'AWS workloads', match: 'domain', tunnel: 'aws', adapter: 'OpenVPN' },
        { dest: 'claude.ai', rule: 'US content', match: 'domain', tunnel: 'us-content', adapter: 'Proton VPN' },
        { dest: 'gmail.com', rule: '', match: '', tunnel: 'direct', adapter: 'no VPN', direct: true },
      ] as Route[],
    }

  useEffect(() => {
    if (paused || touched) return
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
    const t = window.setInterval(() => setActive(a => (a + 1) % ROWS.length), 2600)
    return () => window.clearInterval(t)
  }, [paused, touched])

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
          className="switchboard reveal"
          onMouseEnter={() => setPaused(true)}
          onMouseLeave={() => setPaused(false)}
          onFocus={() => setPaused(true)}
          onBlur={e => { if (!e.currentTarget.contains(e.relatedTarget as Node)) setPaused(false) }}
        >
          <div className="sb-legend">
            <span className="sb-legend-item in">{copy.boardIn}</span>
            <span className="sb-legend-item core">{copy.boardCore}</span>
            <span className="sb-legend-item out">{copy.boardOut}</span>
          </div>

          <svg className="sb-svg" viewBox="0 0 640 360" role="group" aria-label={copy.boardCore}>
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
                  onClick={() => { setActive(i); setTouched(true) }}
                  aria-pressed={on}
                  role="button"
                  tabIndex={0}
                  onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setActive(i); setTouched(true) } }}
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

          <div className="sb-readout" key={active} aria-live={touched ? 'polite' : 'off'}>
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
      </div>
    </section>
  )
}
