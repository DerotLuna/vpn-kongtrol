// Kongtrol UI — "Editar <perfil>" sheet · v2.0.0 · 2026-11-19
//
// A sheet (bottom on phones, centered on wide screens) with the full guided
// form. Creating a profile moved to the shell's "＋ Agregar" sheet
// (components/add, ADR-0055); this one only edits. Checks run on "Guardar"
// and then on every change; the daemon's own validation error lands on its
// field when we know which one, otherwise above the form — always with what
// to do. Nothing is saved until the daemon accepts.
//
// A connected profile saves with "Guardar y reconectar": PUT
// /api/v1/vpns/{name}?reconnect=true, so only this VPN cuts for a few
// seconds and comes back with the new settings (owner decision, ADR-0055).
// An older daemon answers `reconnect_required` instead: the toast then
// offers "Reconectar" (the tunnel reload Inicio uses).
import { useRef, useState, type FormEvent } from 'react'
import { ApiError } from '../../api/client'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { useUnsavedChanges } from '../../app/unsaved'
import { useTunnelReload } from '../../data/home'
import { useSaveProfile, type VPNProfile } from '../../data/profiles'
import { toasts } from '../../data/toast'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import type { FieldError } from '../ui/Field'
import { Banner } from '../ui/States'
import { checkDraft, fieldForErrorKey, fromProfile, toRequest, withType, type Draft, type DraftField, type Problems } from './draft'
import { localizeError } from './errors'
import { ProfileForm } from './ProfileForm'

interface Props {
  open: boolean
  onClose: () => void
  editing: VPNProfile
  profiles: readonly VPNProfile[]
  /** The tunnel is up or connecting (renaming is refused then). */
  busy?: boolean
  /** The tunnel is connected: saving reconnects it. */
  connected?: boolean
}

export function ProfileEditor({ open, onClose, editing, profiles, busy = false, connected = false }: Props) {
  const { t } = usePreferences()
  const save = useSaveProfile()
  const reload = useTunnelReload()
  const [draft, setDraft] = useState<Draft>(() => fromProfile(editing))
  // What the sheet opened with: any difference is an unsaved edit.
  const [pristine] = useState(draft)
  useUnsavedChanges(JSON.stringify(draft) !== JSON.stringify(pristine))
  const [tried, setTried] = useState(false)
  const [serverErrors, setServerErrors] = useState<Partial<Record<DraftField, FieldError>>>({})
  const [formError, setFormError] = useState<unknown>(null)
  const formRef = useRef<HTMLFormElement>(null)

  const problems: Problems = tried ? checkDraft(draft, { profiles, editing: editing.name }) : {}

  const patch = (p: Partial<Draft>) => {
    setDraft((d) => ({ ...d, ...p }))
    const touched = Object.keys(p) as DraftField[]
    setServerErrors((e) => {
      const next = { ...e }
      for (const f of touched) delete next[f]
      return next
    })
  }

  const submit = (e: FormEvent) => {
    e.preventDefault()
    setTried(true)
    setFormError(null)
    const found = checkDraft(draft, { profiles, editing: editing.name })
    if (Object.keys(found).length > 0) {
      // Move to the first problem (WCAG 3.3.1); the error is linked to it.
      requestAnimationFrame(() => formRef.current?.querySelector<HTMLElement>('[aria-invalid="true"], .kp-failover select, .kp-failover button')?.focus())
      return
    }
    const name = draft.name.trim()
    save.mutate(
      { original: editing.name, body: toRequest(draft, 'update'), reconnect: connected },
      {
        onSuccess: (res) => {
          onClose()
          if (res?.status === 'renamed') {
            toasts.show({
              tone: 'ok',
              message: t('profiles.toast.renamed', { from: res.previous ?? editing.name, to: res.profile }),
              detail: t('profiles.toast.renamed.detail'),
              duration: 9_000,
            })
            return
          }
          if (res?.reconnected) {
            toasts.show({ tone: 'ok', message: t('profiles.toast.reconnected', { name }) })
            return
          }
          if (res?.reconnect_error) {
            toasts.show({
              tone: 'bad',
              message: t('profiles.toast.reconnectFailed', { name }),
              detail: `${res.reconnect_error} · ${t('profiles.toast.reconnectFailed.fix')}`,
              action: { label: t('profiles.action.reconnect'), run: () => reload(name) },
            })
            return
          }
          if (res?.reconnect_required) {
            toasts.show({
              tone: 'info',
              message: t('profiles.toast.updated', { name }),
              detail: t('profiles.toast.reconnectToUse'),
              action: { label: t('profiles.action.reconnect'), run: () => reload(name) },
              duration: 10_000,
            })
            return
          }
          toasts.show({
            tone: 'ok',
            message: t('profiles.toast.updated', { name }),
            detail: t('profiles.toast.restartToUse'),
            duration: 7_000,
          })
        },
        onError: (err) => {
          const local = localizeError(err, t)
          const field = local instanceof ApiError ? fieldForErrorKey(local.key) : undefined
          if (field) {
            const { message, fix } = describeError(local, t)
            setServerErrors({ [field]: { message, fix } })
            requestAnimationFrame(() => formRef.current?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus())
          } else {
            setFormError(local)
          }
        },
      },
    )
  }

  const formErr = formError ? describeError(formError, t) : null
  const saveLabel = connected
    ? save.isPending
      ? t('profiles.action.reconnecting')
      : t('profiles.action.saveReconnect')
    : t('profiles.action.save')

  return (
    <Dialog
      open={open}
      onClose={onClose}
      variant="sheet"
      title={t('profiles.editor.editTitle', { name: editing.name })}
      description={t('profiles.editor.editLead')}
    >
      <form ref={formRef} className="kp-editor" onSubmit={submit} noValidate>
        {formErr && (
          <Banner tone="bad" title={formErr.message} urgent>
            <span>{formErr.fix}</span>
          </Banner>
        )}
        <ProfileForm
          draft={draft}
          onChange={patch}
          onTypeChange={(type) => setDraft((d) => withType(d, type))}
          problems={problems}
          serverErrors={serverErrors}
          profiles={profiles}
          editing={editing}
          busy={busy}
        />
        {connected && <p className="kp-muted kp-small kp-editor__cut">{t('profiles.editor.reconnectNote')}</p>}
        <footer className="k-dialog__actions kp-editor__actions">
          <Button variant="outlined" onClick={onClose}>
            {t('profiles.action.cancel')}
          </Button>
          <Button type="submit" loading={save.isPending}>
            {saveLabel}
          </Button>
        </footer>
      </form>
    </Dialog>
  )
}
