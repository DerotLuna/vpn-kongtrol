// Kongtrol UI — one-series line chart (SVG) · v1.0.0 · 2026-11-18
//
// Single series, so the title names it and no legend is needed (dataviz:
// small multiples instead of two colors that fail the normal-vision floor).
// Steel line + soft area, recessive grid, y labels for 0 and the shared
// max, x labels "hace N min" / "ahora". The text alternative is the
// figcaption summary (always visible) plus a table elsewhere; hover and
// the arrow keys move a crosshair with a readout (WCAG 1.4.13: the readout
// stays while hovered/focused). No animation: points only move on new data.
import { memo, useId, useState, type KeyboardEvent, type PointerEvent } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { formatClock } from '../../data/traffic'

export interface MiniChartProps {
  title: string
  times: readonly number[]
  values: readonly number[]
  /** Upper bound of the y scale (shared between charts that compare). */
  max: number
  format: (v: number) => string
  summary: string
}

const W = 320
const H = 80

function MiniChartImpl({ title, times, values, max, format, summary }: MiniChartProps) {
  const { t, lang } = usePreferences()
  const [active, setActive] = useState<number | null>(null)
  const descId = useId()
  const n = values.length
  const t0 = times[0] ?? 0
  const t1 = times[n - 1] ?? t0
  const span = Math.max(t1 - t0, 1)
  const top = max > 0 ? max : 1
  const xs = times.map((at) => ((at - t0) / span) * W)
  const ys = values.map((v) => H - (Math.min(v, top) / top) * (H - 4))
  const line = xs.map((x, i) => `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${ys[i]!.toFixed(1)}`).join(' ')
  const area = n > 1 ? `${line} L${W},${H} L0,${H} Z` : ''
  const minutes = Math.round(span / 60_000)
  const startLabel = minutes >= 1 ? t('traffic.chart.ago', { n: minutes }) : t('traffic.chart.agoSec', { n: Math.round(span / 1000) })

  const nearest = (frac: number) => {
    const target = t0 + frac * span
    let best = 0
    for (let i = 1; i < n; i++) if (Math.abs(times[i]! - target) < Math.abs(times[best]! - target)) best = i
    return best
  }
  const onMove = (e: PointerEvent<HTMLDivElement>) => {
    const r = e.currentTarget.getBoundingClientRect()
    if (r.width > 0) setActive(nearest(Math.min(1, Math.max(0, (e.clientX - r.left) / r.width))))
  }
  const onKey = (e: KeyboardEvent<HTMLDivElement>) => {
    const last = n - 1
    const cur = active ?? last
    const next =
      e.key === 'ArrowLeft' ? Math.max(0, cur - 1)
      : e.key === 'ArrowRight' ? Math.min(last, cur + 1)
      : e.key === 'Home' ? 0
      : e.key === 'End' ? last
      : e.key === 'Escape' ? null
      : undefined
    if (next === undefined) return
    e.preventDefault()
    setActive(next)
  }
  const idx = active !== null && active < n ? active : null
  const left = idx !== null ? (xs[idx]! / W) * 100 : 0

  return (
    <figure className="tr-chart">
      <div className="tr-chart__head">
        <span className="tr-chart__title">{title}</span>
        <span className="tr-chart__now">{format(values[n - 1] ?? 0)}</span>
      </div>
      <div
        className="tr-chart__plot"
        tabIndex={0}
        role="slider"
        aria-label={title}
        aria-describedby={descId}
        aria-valuemin={0}
        aria-valuemax={Math.max(0, n - 1)}
        aria-valuenow={idx ?? n - 1}
        aria-valuetext={t('traffic.chart.point', {
          time: formatClock(times[idx ?? n - 1] ?? t1, lang),
          value: format(values[idx ?? n - 1] ?? 0),
        })}
        onPointerMove={onMove}
        onPointerLeave={() => setActive(null)}
        onKeyDown={onKey}
        onBlur={() => setActive(null)}
      >
        <span className="tr-chart__y tr-chart__y--max" aria-hidden="true">{format(top)}</span>
        <span className="tr-chart__y tr-chart__y--zero" aria-hidden="true">0</span>
        <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" aria-hidden="true" focusable="false">
          <line className="tr-chart__grid" x1="0" x2={W} y1="4" y2="4" vectorEffect="non-scaling-stroke" />
          <line className="tr-chart__grid" x1="0" x2={W} y1={H / 2 + 2} y2={H / 2 + 2} vectorEffect="non-scaling-stroke" />
          <line className="tr-chart__base" x1="0" x2={W} y1={H - 0.5} y2={H - 0.5} vectorEffect="non-scaling-stroke" />
          {area && <path className="tr-chart__area" d={area} />}
          <path className="tr-chart__line" d={line} vectorEffect="non-scaling-stroke" />
        </svg>
        {idx !== null && (
          <>
            <span className="tr-chart__cross" style={{ left: `${left}%` }} aria-hidden="true" />
            <span
              className="tr-chart__dot"
              style={{ left: `${left}%`, top: `${(ys[idx]! / H) * 100}%` }}
              aria-hidden="true"
            />
            <span
              className={`tr-chart__tip ${left > 60 ? 'tr-chart__tip--left' : ''}`}
              style={{ left: `${left}%` }}
              aria-hidden="true"
            >
              <strong>{format(values[idx]!)}</strong>
              <span>{formatClock(times[idx]!, lang)}</span>
            </span>
          </>
        )}
      </div>
      <div className="tr-chart__x" aria-hidden="true">
        <span>{startLabel}</span>
        <span>{t('traffic.chart.now')}</span>
      </div>
      <figcaption id={descId} className="tr-chart__summary">
        {summary} <span className="visually-hidden">{t('traffic.chart.keys')}</span>
      </figcaption>
    </figure>
  )
}

export const MiniChart = memo(MiniChartImpl)
