// Kongtrol UI — confirm before deleting a profile or group · v1.1.0 · 2026-11-18
//
// An app dialog (legacy used the browser's confirm() with untranslated
// buttons, docs/dashboard-parity.md §5.2 item 17). Cancel comes first and
// is the safe default. When something still depends on the item (a group,
// another profile's backups) the dialog says what to change first and
// offers no delete, instead of letting the daemon refuse afterwards.
import type { ReactNode } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'

interface Props {
  open: boolean
  title: string
  body: ReactNode
  /** Set when deleting is not possible yet: the reason + what to do (+ where it is used). */
  blocked?: { message: string; fix: string; refs?: { label: string; names: string[] }[] } | null
  onCancel: () => void
  onConfirm: () => void
}

export function ConfirmDelete({ open, title, body, blocked, onCancel, onConfirm }: Props) {
  const { t } = usePreferences()
  return (
    <Dialog
      open={open}
      onClose={onCancel}
      alert
      title={title}
      description={blocked ? <strong>{blocked.message}</strong> : body}
      actions={
        blocked ? (
          <Button variant="outlined" onClick={onCancel}>
            {t('profiles.action.understood')}
          </Button>
        ) : (
          <>
            <Button variant="outlined" onClick={onCancel}>
              {t('profiles.action.cancel')}
            </Button>
            <Button variant="danger" onClick={onConfirm}>
              {t('profiles.action.delete')}
            </Button>
          </>
        )
      }
    >
      {!!blocked?.refs?.length && (
        <dl className="kp-refs" aria-label={t('profiles.err.inUse.where')}>
          {blocked.refs.map((r) => (
            <div key={r.label} className="kp-refs__row">
              <dt>{r.label}</dt>
              <dd>
                <ul className="kp-refs__names">
                  {r.names.map((n) => (
                    <li key={n} className="mono">
                      {n}
                    </li>
                  ))}
                </ul>
              </dd>
            </div>
          ))}
        </dl>
      )}
      {blocked && <p className="kp-muted">{blocked.fix}</p>}
    </Dialog>
  )
}
