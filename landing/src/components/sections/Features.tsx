import { useEffect, useState } from 'react'
import { Lang } from '../../i18n'
import './routeboard.css'

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
// Core geometry: center (320,180), ring r 56. Ports are small dots ON the
// ring (r 3) at x 264 (IN) and 376 (OUT); wires stop at each dot's outer edge.
const CORE_R = 56
const PORT_DOT = 3
const PORT_L = 320 - CORE_R - PORT_DOT
const PORT_R = 320 + CORE_R + PORT_DOT

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
      portIn: 'tus apps',
      portOut: 'VPN o internet',
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
      portIn: 'your apps',
      portOut: 'VPN or internet',
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
  // the visitor picked a destination (click, tap, Enter or Space)
  const pick = (i: number) => { setActive(i); setTouched(true) }

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
            {/* wires fade into the core ring as they reach its ports */}
            <defs>
              <linearGradient id="sb-fade-g" gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="640" y2="0">
                <stop offset="0" stopColor="#fff" />
                <stop offset={220 / 640} stopColor="#fff" />
                <stop offset={PORT_L / 640} stopColor="#fff" stopOpacity="0.25" />
                <stop offset={PORT_R / 640} stopColor="#fff" stopOpacity="0.25" />
                <stop offset={420 / 640} stopColor="#fff" />
                <stop offset="1" stopColor="#fff" />
              </linearGradient>
              <mask id="sb-fade" maskUnits="userSpaceOnUse" x="0" y="0" width="640" height="360">
                <rect width="640" height="360" fill="url(#sb-fade-g)" />
              </mask>
            </defs>

            {/* wires — behind nodes */}
            <g mask="url(#sb-fade)">
              {routes.map((r, i) => {
                const y = ROWS[i]
                const on = i === active
                return (
                  <g key={`w-${i}`} className={`sb-wire-group${on ? ' active' : ''}${r.direct ? ' direct' : ''}`}>
                    <path className="sb-wire" d={`M156 ${y} C 212 ${y}, 212 180, ${PORT_L} 180`} />
                    <path className="sb-wire" d={`M${PORT_R} 180 C 428 180, 428 ${y}, 484 ${y}`} />
                  </g>
                )
              })}
            </g>

            {/* core — ring, ports, pulse and logo share one center. The pulse is an
                ambient ring inside the core ring, unaffected by route changes. */}
            <g className={`sb-core${cur.direct ? ' direct' : ''}`} transform="translate(320 180)">
              <circle className="sb-pulse" r="48" />
              <circle className="sb-ring" r={CORE_R} />
              <image className="sb-logo" href={`${import.meta.env.BASE_URL}logo-kong.svg`} x="-40" y="-40" width="80" height="80" />
              <circle className="sb-port in" cx={-CORE_R} r={PORT_DOT} />
              <circle className="sb-port out" cx={CORE_R} r={PORT_DOT} />
              <text className="sb-port-label" x={-CORE_R} y={CORE_R + 26} textAnchor="middle">{copy.portIn}</text>
              <text className="sb-port-label" x={CORE_R} y={CORE_R + 26} textAnchor="middle">{copy.portOut}</text>
            </g>

            {/* destination nodes (left) */}
            {routes.map((r, i) => {
              const y = ROWS[i]
              const on = i === active
              return (
                <g
                  key={`in-${i}`}
                  className={`sb-node in${on ? ' active' : ''}`}
                  onClick={() => pick(i)}
                  aria-pressed={on}
                  role="button"
                  tabIndex={0}
                  onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); pick(i) } }}
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

          {/* Narrow screens: the scaled SVG would shrink its labels below 11px,
              so the same board is laid out in HTML — one row per destination,
              real buttons, same state, readout and live region. routeboard.css
              swaps the two by media query (never by JS), so the prerender and
              the first client render match; the hidden one is display:none, out
              of the tab order and the accessibility tree. */}
          <div className="sb-list" role="group" aria-label={copy.boardCore}>
            {routes.map((r, i) => {
              const on = i === active
              return (
                <div key={`row-${i}`} className={`sb-row${on ? ' active' : ''}${r.direct ? ' direct' : ''}`}>
                  <button type="button" className="sb-row-dest" aria-pressed={on} onClick={() => pick(i)}>
                    {r.dest}
                  </button>
                  <span className="sb-row-wire" aria-hidden="true"><i /></span>
                  <span className="sb-row-out">
                    <b>{r.tunnel}</b>
                    <small>{r.adapter}</small>
                  </span>
                </div>
              )
            })}
          </div>

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
