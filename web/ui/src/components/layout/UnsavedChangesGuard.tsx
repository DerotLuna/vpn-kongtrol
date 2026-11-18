// Kongtrol UI — "Tienes cambios sin guardar" before leaving a form · v1.0.0 · 2026-11-18
//
// Blocks in-app navigation (links, Back/Forward) while a form registered
// in app/unsaved.ts has edits, with React Router's useBlocker (data router,
// ADR-0041). The safe action comes first and keeps the edits; "Descartar"
// drops them and goes on. Changing only the query or hash of the same page
// is not leaving it, so it never asks.
import { useBlocker } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { unsavedChanges } from '../../app/unsaved'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'

export function UnsavedChangesGuard() {
  const { t } = usePreferences()
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) => currentLocation.pathname !== nextLocation.pathname && unsavedChanges.any(),
  )
  const blocked = blocker.state === 'blocked'
  const stay = () => blocker.reset?.()
  return (
    <Dialog
      open={blocked}
      onClose={stay}
      alert
      title={t('unsaved.title')}
      description={t('unsaved.body')}
      actions={
        <>
          <Button variant="filled" onClick={stay}>
            {t('unsaved.keep')}
          </Button>
          <Button
            variant="danger"
            onClick={() => {
              unsavedChanges.discardAll()
              blocker.proceed?.()
            }}
          >
            {t('unsaved.discard')}
          </Button>
        </>
      }
    />
  )
}
