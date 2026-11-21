import { CSSProperties, useEffect, useRef, useState } from 'react'
import { useInView } from '../../hooks/useInView'
import { Lang } from '../../i18n'
import './routeboard.css'

interface Props {
  lang: Lang
}

type Route = {
  dest: string
  /** what the destination is, in everyday words — kept short so it fits the
      150-unit card at 11px in the SVG */
  human: string
  tunnel: string
  adapter: string
  direct?: boolean
}

// Patch panel ("switchboard", routeboard.css v2). One example at a time: a
// packet leaves a destination, enters the panel through its socket, follows
// its cable to the exit socket and lands on the exit; then the rule label
// shows. CYCLE must cover the CSS timeline (packet --pp-travel, landing at
// --pp-land); scripts/readme-media.mjs records 4 × CYCLE.
const CYCLE = 3400

// Fixed geometry in the 640×360 viewBox (desktop) — no DOM measurement.
// Cards: 6–156 (left) and 484–634 (right). Panel: 236–404. Sockets sit ON the
// panel edges at each row, so every wire leaves its card edge horizontally.
const ROWS = [64, 141, 219, 296]
const CARD_L = 156
const CARD_R = 484
const PANEL_L = 236
const PANEL_R = 404
const JACK = 6 // half-width of a socket housing
// Destination i leaves through exit row OUT[i]: the exits are not in the same
// order as the destinations, so the cables cross inside the panel like real
// patch cords. Exits (right column) are listed by row: EXIT_ORDER[row] = route.
const OUT = [1, 3, 0, 2]
const EXIT_ORDER = [2, 0, 3, 1]

const cable = (i: number) => {
  const y1 = ROWS[i]
  const y2 = ROWS[OUT[i]]
  return `M${PANEL_L + JACK} ${y1} C 320 ${y1}, 320 ${y2}, ${PANEL_R - JACK} ${y2}`
}
// the whole route the packet travels: card edge → socket → cable → socket → card edge
const route = (i: number) => {
  const y1 = ROWS[i]
  const y2 = ROWS[OUT[i]]
  return `M${CARD_L} ${y1} H${PANEL_L + JACK} C 320 ${y1}, 320 ${y2}, ${PANEL_R - JACK} ${y2} H${CARD_R}`
}

// Narrow screens: a 300×120 strip between the two lists — sockets on its top
// edge (destinations, in order) and bottom edge (exits, in EXIT_ORDER).
const STRIP_X = [48, 116, 184, 252]
const STRIP_T = 16
const STRIP_B = 104
const stripRoute = (i: number) => {
  const x1 = STRIP_X[i]
  const x2 = STRIP_X[OUT[i]]
  return `M${x1} 0 V${STRIP_T} C ${x1} 60, ${x2} 60, ${x2} ${STRIP_B} V120`
}
const stripCable = (i: number) => {
  const x1 = STRIP_X[i]
  const x2 = STRIP_X[OUT[i]]
  return `M${x1} ${STRIP_T + JACK} C ${x1} 60, ${x2} 60, ${x2} ${STRIP_B - JACK}`
}

export default function Features({ lang }: Props) {
  const [active, setActive] = useState(0)
  // bumps on every new example, so the keyed packet / trace / label remount
  // and their CSS timeline starts again even when the same route repeats
  const [run, setRun] = useState(0)
  const [paused, setPaused] = useState(false)
  // Announce only a destination the visitor picked, never the automatic demo.
  const [touched, setTouched] = useState(false)
  // rotates and animates only while on screen (ADR-0031)
  const boardRef = useRef<HTMLDivElement>(null)
  const live = useInView(boardRef)
  // .is-live is toggled on the node, not through className: useReveal adds
  // .is-in to this .reveal element by hand, and a className re-render would drop it
  useEffect(() => {
    boardRef.current?.classList.toggle('is-live', live && !paused)
  }, [live, paused])

  const copy = lang === 'es'
    ? {
      label: '$ kongtrol policy show claude.ai',
      title: 'Cada sitio, por la VPN correcta.',
      sub: 'Elige qué sitios y redes van por cada VPN. Por ejemplo, el sistema de tu empresa por la VPN del trabajo y otro sitio por tu VPN personal.',
      boardIn: 'Sitio o red',
      boardOut: 'Conexión que usa',
      boardName: 'Mapa de rutas: qué destino sale por qué VPN',
      hint: 'Elige un sitio o una red para ver su recorrido.',
      pause: 'Pausar demo',
      resume: 'Continuar demo',
      routes: [
        { dest: '10.10.0.0/16', human: 'sistema de tu empresa', tunnel: 'office', adapter: 'FortiClient' },
        { dest: '*.amazonaws.com', human: 'servidores en la nube', tunnel: 'aws', adapter: 'OpenVPN' },
        { dest: 'claude.ai', human: 'un sitio en EE. UU.', tunnel: 'us-content', adapter: 'Proton VPN' },
        { dest: 'gmail.com', human: 'tu correo de siempre', tunnel: 'directo', adapter: 'sin VPN', direct: true },
      ] as Route[],
    }
    : {
      label: '$ kongtrol policy show claude.ai',
      title: 'Every site, through the right VPN.',
      sub: 'Choose which sites and networks use each VPN. For example, send your company system through the work VPN and another site through your personal VPN.',
      boardIn: 'Site or network',
      boardOut: 'Connection it uses',
      boardName: 'Route map: which destination goes out through which VPN',
      hint: 'Choose a site or network to see its route.',
      pause: 'Pause demo',
      resume: 'Resume demo',
      routes: [
        { dest: '10.10.0.0/16', human: 'your company system', tunnel: 'office', adapter: 'FortiClient' },
        { dest: '*.amazonaws.com', human: 'cloud servers', tunnel: 'aws', adapter: 'OpenVPN' },
        { dest: 'claude.ai', human: 'a site in the US', tunnel: 'us-content', adapter: 'Proton VPN' },
        { dest: 'gmail.com', human: 'your everyday email', tunnel: 'direct', adapter: 'no VPN', direct: true },
      ] as Route[],
    }

  useEffect(() => {
    if (paused || !live) return
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
    let timer: number | undefined
    const updateTimer = () => {
      window.clearInterval(timer)
      if (reducedMotion.matches) return
      timer = window.setInterval(() => {
        setActive(a => (a + 1) % ROWS.length)
        setRun(n => n + 1)
      }, CYCLE)
    }
    updateTimer()
    reducedMotion.addEventListener('change', updateTimer)
    return () => {
      reducedMotion.removeEventListener('change', updateTimer)
      window.clearInterval(timer)
    }
  }, [paused, live])

  const routes = copy.routes
  const cur = routes[active]
  // the visitor picked a destination (click, tap, Enter or Space)
  const pick = (i: number) => {
    setActive(i)
    setRun(n => n + 1)
    setPaused(true)
    setTouched(true)
  }
  const toggleAnimation = () => {
    if (paused) setTouched(false)
    setPaused(!paused)
  }
  // the line-art original as the panel badge (DESIGN.md Logo, ADR-0060)
  const mark = `${import.meta.env.BASE_URL}logo-kong-on-dark.svg`
  const cls = (i: number, extra = '') =>
    `${extra}${i === active ? ' active' : ''}${routes[i].direct ? ' direct' : ''}`

  return (
    <section id="map" className="section features-section tone-ink edge edge-b edge-b-drift">
      <div className="container">
        <div className="section-label cmd-label">{copy.label}</div>
        <h2 className="section-title">{copy.title}</h2>
        <p className="section-sub">{copy.sub}</p>

        {/* ── SIGNATURE: routing patch panel ── */}
        <div
          ref={boardRef}
          className="switchboard reveal"
          style={{ '--pp-cycle': `${CYCLE}ms` } as CSSProperties}
        >
          <div className="sb-legend" aria-hidden="true">
            <span>{copy.boardIn}</span>
            <span>{copy.boardOut}</span>
          </div>

          <svg className="sb-svg" viewBox="0 0 640 360" role="group" aria-label={copy.boardName}>
            {/* wires from each card edge to its socket (behind everything) */}
            {routes.map((_, i) => (
              <g key={`w-${i}`}>
                <path className="pp-wire" d={`M${CARD_L} ${ROWS[i]} H${PANEL_L - JACK}`} />
                <path className="pp-wire" d={`M${PANEL_R + JACK} ${ROWS[OUT[i]]} H${CARD_R}`} />
              </g>
            ))}

            {/* the panel: body, four corner screws, the line-art mark as its
                badge, centred above the cables (they cross lower down) */}
            <g className="pp-panel">
              <rect className="pp-body" x={PANEL_L} y="22" width={PANEL_R - PANEL_L} height="316" rx="10" />
              <circle className="pp-screw" cx={PANEL_L + 12} cy="34" r="2.5" />
              <circle className="pp-screw" cx={PANEL_R - 12} cy="34" r="2.5" />
              <circle className="pp-screw" cx={PANEL_L + 12} cy="326" r="2.5" />
              <circle className="pp-screw" cx={PANEL_R - 12} cy="326" r="2.5" />
              <image className="pp-badge" href={mark} x={(PANEL_L + PANEL_R) / 2 - 17} y="25" width="34" height="34" />
              {routes.map((_, i) => (
                <path key={`c-${i}`} className={cls(i, 'pp-cable')} d={cable(i)} />
              ))}
            </g>

            {/* the active route lights up behind the packet; under reduced
                motion every route is drawn lit and static (routeboard.css) */}
            {routes.map((_, i) => (
              <path
                key={`t-${i}-${i === active ? run : 'x'}`}
                className={cls(i, 'pp-trace')}
                d={route(i)}
                pathLength={100}
              />
            ))}

            {/* sockets: a jack housing with its opening, on both panel edges */}
            {ROWS.map((y, row) => {
              const inRoute = row
              const outRoute = EXIT_ORDER[row]
              return (
                <g key={`j-${row}`}>
                  <g className={cls(inRoute, 'pp-jack in')}>
                    <rect x={PANEL_L - JACK} y={y - 8} width={JACK * 2} height="16" rx="2" />
                    <rect className="pp-jack-hole" x={PANEL_L - 2} y={y - 4} width="4" height="8" rx="1" />
                  </g>
                  <g className={cls(outRoute, 'pp-jack out')}>
                    <rect x={PANEL_R - JACK} y={y - 8} width={JACK * 2} height="16" rx="2" />
                    <rect className="pp-jack-hole" x={PANEL_R - 2} y={y - 4} width="4" height="8" rx="1" />
                  </g>
                </g>
              )
            })}

            {/* the packet: a round dash that travels the active route */}
            <path key={`p-${run}`} className={`pp-packet${cur.direct ? ' direct' : ''}`} d={route(active)} pathLength={100} />

            {/* destination cards (left) */}
            {routes.map((r, i) => {
              const y = ROWS[i]
              return (
                <g
                  key={`in-${i}`}
                  className={cls(i, 'sb-node in')}
                  onClick={() => pick(i)}
                  aria-pressed={i === active}
                  role="button"
                  tabIndex={0}
                  onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); pick(i) } }}
                >
                  <rect x="6" y={y - 20} width={CARD_L - 6} height="40" rx="6" />
                  <text x="20" y={y - 2}>{r.dest}</text>
                  <text className="sb-in-human" x="20" y={y + 13}>{r.human}</text>
                </g>
              )
            })}

            {/* exit cards (right), listed in panel order */}
            {EXIT_ORDER.map((i, row) => {
              const y = ROWS[row]
              const r = routes[i]
              return (
                <g key={`out-${i}`} className={cls(i, 'sb-node out')}>
                  <rect x={CARD_R} y={y - 20} width="150" height="40" rx="6" />
                  <text className="sb-out-name" x={CARD_R + 14} y={y - 2}>{r.tunnel}</text>
                  <text className="sb-out-sub" x={CARD_R + 14} y={y + 13}>{r.adapter}</text>
                </g>
              )
            })}
          </svg>

          {/* Narrow screens: the scaled SVG would shrink its labels below 11px,
              so the board stacks: destinations (real buttons), a small panel
              strip with no text, then the exits. routeboard.css swaps the two
              layouts by media query (never by JS), so the prerender and the
              first client render match; the hidden one is display:none, out of
              the tab order and the accessibility tree. */}
          <div className="sb-stack" role="group" aria-label={copy.boardName}>
            <p className="sb-stack-head" aria-hidden="true">{copy.boardIn}</p>
            <div className="sb-stack-list">
              {routes.map((r, i) => (
                <button key={`d-${i}`} type="button" className={cls(i, 'sb-chip in')} aria-pressed={i === active} onClick={() => pick(i)}>
                  <span>{r.dest}</span>
                  <small>{r.human}</small>
                </button>
              ))}
            </div>
            <svg className="sb-strip" viewBox="0 0 300 120" aria-hidden="true">
              <rect className="pp-body" x="4" y={STRIP_T} width="292" height={STRIP_B - STRIP_T} rx="10" />
              <image className="pp-badge" href={mark} x="9" y={STRIP_T + 30} width="28" height="28" />
              {routes.map((_, i) => (
                <path key={`sc-${i}`} className={cls(i, 'pp-cable')} d={stripCable(i)} />
              ))}
              {routes.map((_, i) => (
                <path key={`st-${i}-${i === active ? run : 'x'}`} className={cls(i, 'pp-trace')} d={stripRoute(i)} pathLength={100} />
              ))}
              {STRIP_X.map((x, k) => (
                <g key={`sj-${k}`}>
                  <g className={cls(k, 'pp-jack in')}>
                    <rect x={x - 8} y={STRIP_T - JACK} width="16" height={JACK * 2} rx="2" />
                    <rect className="pp-jack-hole" x={x - 4} y={STRIP_T - 2} width="8" height="4" rx="1" />
                  </g>
                  <g className={cls(EXIT_ORDER[k], 'pp-jack out')}>
                    <rect x={x - 8} y={STRIP_B - JACK} width="16" height={JACK * 2} rx="2" />
                    <rect className="pp-jack-hole" x={x - 4} y={STRIP_B - 2} width="8" height="4" rx="1" />
                  </g>
                </g>
              ))}
              <path key={`sp-${run}`} className={`pp-packet${cur.direct ? ' direct' : ''}`} d={stripRoute(active)} pathLength={100} />
            </svg>
            <p className="sb-stack-head" aria-hidden="true">{copy.boardOut}</p>
            <div className="sb-stack-list exits">
              {EXIT_ORDER.map(i => (
                <span key={`e-${i}`} className={cls(i, 'sb-chip out')}>
                  <b>{routes[i].tunnel}</b>
                  <small>{routes[i].adapter}</small>
                </span>
              ))}
            </div>
          </div>

          {/* the rule, shown once the packet lands */}
          <p className="pp-rule" aria-live={touched ? 'polite' : 'off'}>
            <span key={`r-${run}`} className={cur.direct ? 'direct' : undefined}>
              {cur.dest} <span aria-hidden="true">→</span> {cur.tunnel}
            </span>
          </p>
          <div className="sb-controls">
            <p className="sb-hint">{copy.hint}</p>
            <button type="button" className="sb-motion" onClick={toggleAnimation}>
              {paused ? copy.resume : copy.pause}
            </button>
          </div>
        </div>
      </div>
    </section>
  )
}
