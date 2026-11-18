// Kongtrol UI — latency trend sparkline (Inicio) · v1.0.0 · 2026-11-18
//
// SVG, not canvas: crisp at any zoom, follows the theme through
// currentColor (no redraw on theme change) and carries its own text
// alternative (legacy canvas had none). Min-max scaled like charts.js.
// Subscribes to the latency store itself, so a new sample redraws only this
// tunnel's line, not the table row.
import { memo } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { useLatencySamples } from '../../data/home'
import { sparkPoints } from './model'

const W = 72
const H = 22
const PAD = 2

export const LatencySparkline = memo(function LatencySparkline({ name }: { name: string }) {
  const { t } = usePreferences()
  const samples = useLatencySamples(name)
  if (samples.length < 2) {
    return <span className="home-spark home-spark--empty">{t('home.latency.collecting')}</span>
  }
  const label = t('home.latency.trend', {
    n: samples.length,
    min: Math.min(...samples),
    max: Math.max(...samples),
    now: samples.at(-1) ?? 0,
  })
  const points = sparkPoints(samples, W, H, PAD)
  const last = points.split(' ').at(-1)?.split(',') ?? ['0', '0']
  return (
    <svg className="home-spark" viewBox={`0 0 ${W} ${H}`} width={W} height={H} role="img" aria-label={label}>
      <title>{label}</title>
      <polyline points={points} fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" strokeLinecap="round" />
      <circle cx={last[0]} cy={last[1]} r="2" fill="currentColor" />
    </svg>
  )
})
