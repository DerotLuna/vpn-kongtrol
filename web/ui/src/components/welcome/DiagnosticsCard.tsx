// Kongtrol UI — Diagnostics card · v0.1.1 · 2026-11-18
//
// GUI parity for `kongtrol doctor` (GET /api/v1/doctor) and
// `kongtrol export` (GET /api/v1/config/export). The daemon sends every
// check already translated with its "what to do" line; this card only
// renders. The only fix offered is installing the helper, through the same
// flow as the helper card (one OS permission prompt the user accepts —
// never a silent elevation).
import { useEffect, useRef } from 'react'
import type { DoctorState, ExportState, HelperState } from '../../api/hooks'
import type { DoctorCheck } from '../../api/types'
import { usePreferences } from '../../app/preferences-context'
import { Button } from '../ui/Button'
import { guideLink } from '../../links'
import { ErrorState } from '../ui/States'
import { ExternalLink } from '../ui/ExternalLink'
import { StatusPill, type PillTone } from '../ui/StatusPill'

// Four States Rule: fail = red, warn = caution orange, ok = green; each
// also carries its word.
const TONE: Record<DoctorCheck['status'], PillTone> = { ok: 'ok', warn: 'caution', fail: 'bad' }

interface Props {
  doctor: DoctorState
  helper: HelperState
  exporter: ExportState
}

export function DiagnosticsCard({ doctor, helper, exporter }: Props) {
  const { t } = usePreferences()
  const { report, error, running, run } = doctor

  // After a successful "Reparar", check again so the list reflects it.
  const wasInstalled = useRef(false)
  useEffect(() => {
    if (helper.installed && !wasInstalled.current) run()
    wasInstalled.current = helper.installed
  }, [helper.installed, run])

  const problems = report?.checks.filter((c) => c.status !== 'ok') ?? []
  const passed = report?.checks.filter((c) => c.status === 'ok') ?? []
  const firstFix = problems.findIndex((c) => c.fix === 'helper_install')

  let pill = null
  if (report && !running) {
    const tone: PillTone = report.failures > 0 ? 'bad' : report.warnings > 0 ? 'caution' : 'ok'
    const word = report.failures > 0 ? 'diag.pill.fail' : report.warnings > 0 ? 'diag.pill.warn' : 'diag.pill.ok'
    pill = <StatusPill tone={tone}>{t(word)}</StatusPill>
  }

  return (
    <section className="card diag" aria-labelledby="diag-title" aria-busy={running}>
      <div className="card-head">
        <h2 id="diag-title">{t('diag.title')}</h2>
        {pill}
      </div>
      <p className="muted small">{t('diag.what')}</p>

      <div role="status" aria-live="polite">
        {running && <p>{t('diag.running')}</p>}
        {!running && !report && !error && <p className="muted small">{t('diag.idle')}</p>}
        {!running && report && (
          <p>
            {problems.length === 0
              ? t('diag.allOk', { total: report.checks.length })
              : t('diag.summary', { failures: report.failures, warnings: report.warnings })}
          </p>
        )}
      </div>

      {error && <ErrorState help error={error} onRetry={run} />}

      {report && problems.length > 0 && (
        <ul className="check-list">
          {problems.map((c, i) => (
            <li key={`${c.section}|${c.label}|${i}`} className={`check ${c.status}`}>
              <div className="check-head">
                <StatusPill tone={TONE[c.status]}>{t(`diag.status.${c.status}`)}</StatusPill>
                <span className="check-label">{c.label}</span>
                <span className="check-section">{c.section}</span>
              </div>
              <p className="check-detail">{c.detail}</p>
              {c.hint && (
                <p className="check-hint">
                  <strong>{t('diag.whatToDo')}</strong> {c.hint}
                </p>
              )}
              {i === firstFix && (
                <Button variant="tonal" onClick={helper.install} disabled={helper.installing}>
                  {t('diag.fix')}
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}
      <div aria-live="polite">
        {helper.installing && <p className="notice">{t('diag.fixing')}</p>}
        {helper.installed && !helper.installing && firstFix < 0 && report && (
          <p className="notice ok">{t('diag.fixed')}</p>
        )}
      </div>

      {report && passed.length > 0 && (
        <details className="check-ok">
          <summary>{t('diag.okList', { count: passed.length })}</summary>
          <ul className="check-list compact">
            {passed.map((c, i) => (
              <li key={`${c.section}|${c.label}|${i}`} className="check ok">
                <div className="check-head">
                  <StatusPill tone="ok">{t('diag.status.ok')}</StatusPill>
                  <span className="check-label">{c.label}</span>
                </div>
                <p className="check-detail">{c.detail}</p>
              </li>
            ))}
          </ul>
        </details>
      )}

      <div className="notice-actions">
        <Button variant="tonal" onClick={run} disabled={running}>
          {report || error ? t('diag.rerun') : t('diag.run')}
        </Button>
      </div>

      <div className="diag-export">
        <p className="muted small">{t('diag.export.what')}</p>
        <Button variant="outlined" onClick={exporter.download} disabled={exporter.busy}>
          {exporter.busy ? t('diag.export.busy') : t('diag.export')}
        </Button>
        <div aria-live="polite">{exporter.done && <p className="notice ok">{t('diag.export.done')}</p>}</div>
        {exporter.error && <ErrorState help error={exporter.error} />}
      </div>

      <p className="help">
        <ExternalLink href={guideLink('trouble')}>{t('help.trouble')}</ExternalLink>
      </p>
    </section>
  )
}
