// Kongtrol UI — "Nueva VPN" / "Editar" sheet · v1.0.0 · 2026-11-18
//
// A sheet (bottom on phones, centered on wide screens) because creating a
// VPN is a short focused task with its own steps: 1) which VPN, 2) its
// details. Editing opens on step 2. Checks run on "Guardar" and then on
// every change; the daemon's own validation error lands on its field when
// we know which one, otherwise above the form — always with what to do.
// Nothing is saved until the daemon accepts, so a refusal never leaves a
// half-saved profile on screen.
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { ApiError } from '../../api/client'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { useSaveProfile, type AdapterType, type DetectedClient, type VPNProfile } from '../../data/profiles'
import { toasts } from '../../data/toast'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import type { FieldError } from '../ui/Field'
import { Banner } from '../ui/States'
import { checkDraft, emptyDraft, fieldForErrorKey, fromProfile, toRequest, withType, type Draft, type DraftField, type Problems } from './draft'
import { localizeError } from './errors'
import { ProfileForm } from './ProfileForm'
import { useRestartAction } from './useRestartAction'
import { TypePicker } from './TypePicker'

interface Props {
  open: boolean
  onClose: () => void
  /** Profile to edit; undefined = create. */
  editing?: VPNProfile | undefined
  /** Create with this type already chosen (skips step 1). */
  initialType?: AdapterType | undefined
  profiles: readonly VPNProfile[]
  detected: readonly DetectedClient[]
  /** First run: after the first profile, restart on our own and go to Inicio. */
  firstRun: boolean
}

export function ProfileEditor({ open, onClose, editing, initialType, profiles, detected, firstRun }: Props) {
  const { t } = usePreferences()
  const save = useSaveProfile()
  const { restart } = useRestartAction()
  const [step, setStep] = useState<'pick' | 'form'>(editing || initialType ? 'form' : 'pick')
  const [draft, setDraft] = useState<Draft>(() => (editing ? fromProfile(editing) : emptyDraft(initialType ?? 'openvpn')))
  const [tried, setTried] = useState(false)
  const [serverErrors, setServerErrors] = useState<Partial<Record<DraftField, FieldError>>>({})
  const [formError, setFormError] = useState<unknown>(null)
  const formRef = useRef<HTMLFormElement>(null)

  const problems: Problems = tried ? checkDraft(draft, { profiles, editing: editing?.name }) : {}

  const patch = (p: Partial<Draft>) => {
    setDraft((d) => ({ ...d, ...p }))
    const touched = Object.keys(p) as DraftField[]
    setServerErrors((e) => {
      const next = { ...e }
      for (const f of touched) delete next[f]
      return next
    })
  }

  // Step 2 starts at its first field; going back starts at the tiles.
  useEffect(() => {
    if (!open) return
    const el = formRef.current?.querySelector<HTMLElement>(step === 'form' ? 'select, input' : '.kp-tile')
    if (step === 'form' && !editing && !initialType) el?.focus()
    if (step === 'pick' && formRef.current?.dataset.back) el?.focus()
  }, [step, open, editing, initialType])

  const pickType = (type: AdapterType) => {
    setDraft((d) => withType(d, type))
    setStep('form')
  }

  const submit = (e: FormEvent) => {
    e.preventDefault()
    setTried(true)
    setFormError(null)
    const found = checkDraft(draft, { profiles, editing: editing?.name })
    if (Object.keys(found).length > 0) {
      // Move to the first problem (WCAG 3.3.1); the error is linked to it.
      requestAnimationFrame(() => formRef.current?.querySelector<HTMLElement>('[aria-invalid="true"], .kp-failover select, .kp-failover button')?.focus())
      return
    }
    const name = draft.name.trim()
    save.mutate(
      { original: editing?.name, body: toRequest(draft, editing ? 'update' : 'create') },
      {
        onSuccess: () => {
          onClose()
          if (firstRun && !editing) {
            void restart({ thenHome: true, progress: t('profiles.setup.saved') })
            return
          }
          toasts.show({
            tone: 'ok',
            message: t(editing ? 'profiles.toast.updated' : 'profiles.toast.created', { name }),
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

  const title = editing
    ? t('profiles.editor.editTitle', { name: editing.name })
    : firstRun
      ? t('profiles.editor.firstTitle')
      : t('profiles.editor.newTitle')
  const stepText = editing ? undefined : t('profiles.editor.step', { n: step === 'pick' ? 1 : 2, total: 2 })
  const formErr = formError ? describeError(formError, t) : null

  return (
    <Dialog
      open={open}
      onClose={onClose}
      variant="sheet"
      title={title}
      description={
        <>
          {stepText && <span className="kp-step">{stepText} · </span>}
          {step === 'pick' ? t('profiles.editor.pickLead') : editing ? t('profiles.editor.editLead') : t('profiles.editor.formLead')}
        </>
      }
    >
      <form ref={formRef} className="kp-editor" onSubmit={submit} noValidate>
        {step === 'pick' ? (
          <TypePicker detected={detected} onPick={pickType} />
        ) : (
          <>
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
              onPickAnother={
                editing
                  ? undefined
                  : () => {
                      if (formRef.current) formRef.current.dataset.back = '1'
                      setStep('pick')
                    }
              }
            />
          </>
        )}
        <footer className="k-dialog__actions kp-editor__actions">
          <Button variant="outlined" onClick={onClose}>
            {t('profiles.action.cancel')}
          </Button>
          {step === 'form' && (
            <Button type="submit" loading={save.isPending}>
              {editing ? t('profiles.action.save') : firstRun ? t('profiles.action.createFirst') : t('profiles.action.create')}
            </Button>
          )}
        </footer>
      </form>
    </Dialog>
  )
}
