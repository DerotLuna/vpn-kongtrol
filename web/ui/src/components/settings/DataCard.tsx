import { useEffect, useRef, useState } from 'react'
import type { PurgeState } from '../../api/lifecycle-hooks'
import { usePreferences } from '../../app/preferences-context'
import { ErrorNotice } from '../ui/ErrorNotice'

// feature:uninstall
/**
 * "Borrar todos mis datos" (ADR-0034): a danger action behind a
 * confirmation that lists exactly what is deleted and says the app stays
 * installed. Cancel is the default focus. Uninstalling the app itself is
 * the system's job (or `kongtrol uninstall`), so it is only explained here.
 */
export function DataCard({ state }: { state: PurgeState }) {
  const { t } = usePreferences()
  const { busy, done, error, purge } = state
  const [asking, setAsking] = useState(false)
  const cancelRef = useRef<HTMLButtonElement>(null)
  const openerRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (asking) cancelRef.current?.focus()
  }, [asking])

  const close = () => {
    setAsking(false)
    openerRef.current?.focus()
  }

  return (
    <section className="card danger-zone" aria-labelledby="data-title">
      <div className="card-head">
        <h2 id="data-title">{t('data.title')}</h2>
      </div>
      <p>{t('data.what')}</p>
      <p className="muted small">{t('data.uninstallHint')}</p>
      <div aria-live="polite">{done && <p className="notice ok">{t('data.done')}</p>}</div>
      {!done && !asking && (
        <div className="notice-actions">
          <button ref={openerRef} type="button" className="btn danger" onClick={() => setAsking(true)} disabled={busy}>
            {busy ? t('data.busy') : t('data.button')}
          </button>
        </div>
      )}
      {asking && !done && (
        <div
          className="notice bad"
          role="alertdialog"
          aria-modal="false"
          aria-labelledby="data-dialog-title"
          aria-describedby="data-dialog-body"
          onKeyDown={(e) => {
            if (e.key === 'Escape') close()
          }}
        >
          <strong id="data-dialog-title">{t('data.dialog.title')}</strong>
          <div id="data-dialog-body">
            <p>{t('data.dialog.body')}</p>
            <ul>
              <li>{t('data.dialog.item.profiles')}</li>
              <li>{t('data.dialog.item.passwords')}</li>
              <li>{t('data.dialog.item.prefs')}</li>
            </ul>
            <p>{t('data.dialog.keep')}</p>
          </div>
          <div className="notice-actions">
            <button ref={cancelRef} type="button" className="btn ghost" onClick={close} disabled={busy}>
              {t('data.dialog.cancel')}
            </button>
            <button
              type="button"
              className="btn danger"
              onClick={() => {
                purge()
                setAsking(false)
              }}
              disabled={busy}
            >
              {t('data.dialog.confirm')}
            </button>
          </div>
        </div>
      )}
      {error && <ErrorNotice error={error} />}
    </section>
  )
}
