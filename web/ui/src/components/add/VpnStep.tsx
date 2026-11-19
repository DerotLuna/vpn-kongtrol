// Kongtrol UI — A: "Conectar una VPN que ya uso" · v1.0.0 · 2026-11-19
//
// VPN apps found on this computer come first, each with one "Usar";
// "Otra VPN" opens the same type picker as before. The form asks only what
// the profile needs (a name, and the gateway for corporate clients); the
// rest sits under "Más opciones" and stays editable later in Perfiles.
// The name starts as the app's name, made unique ("NordVPN 2") and valid
// for config.ValidateName. WireGuard / OpenVPN need their file: picking
// them goes to the import step.
import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { ApiError } from '../../api/client'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { useDetectedClients, type AdapterType, type VPNProfile } from '../../data/profiles'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { TextField, type FieldError } from '../ui/Field'
import { StatusPill } from '../ui/StatusPill'
import { Switch } from '../ui/Switch'
import { Monogram } from '../profiles/Bits'
import { isKnownType, specOf } from '../profiles/catalog'
import { checkDraft, emptyDraft, fieldForErrorKey, toRequest, type Draft, type DraftField } from '../profiles/draft'
import { localizeError } from '../profiles/errors'
import { ArrowLeftIcon } from '../profiles/icons'
import { TypePicker } from '../profiles/TypePicker'
import { useSaveAndActivate } from './activate'
import { cleanName, isValidName, uniqueName } from './model'
import { ActivateFooter, ErrorBanner } from './parts'

interface Props {
  profiles: readonly VPNProfile[]
  /** Start on the form for this type (deep link `type=`, Perfiles first-run tiles). */
  initialType?: AdapterType | undefined
  onBack: () => void
  /** WireGuard / OpenVPN: continue with their file. */
  onNeedsFile: (type: 'wireguard' | 'openvpn') => void
  onDone: (name: string) => void
  onDirty: (dirty: boolean) => void
  focusFirst: () => void
}

type Sub = { k: 'list' } | { k: 'pick' } | { k: 'form'; type: AdapterType }

const FILE_TYPES = new Set<AdapterType>(['wireguard', 'openvpn'])

export function VpnStep({ profiles, initialType, onBack, onNeedsFile, onDone, onDirty, focusFirst }: Props) {
  const { t } = usePreferences()
  const detected = useDetectedClients()
  const names = useMemo(() => profiles.map((p) => p.name), [profiles])
  const [sub, setSub] = useState<Sub>(initialType ? { k: 'form', type: initialType } : { k: 'list' })
  const [draft, setDraft] = useState<Draft>(() => draftFor(initialType, undefined, names))
  const [pristine, setPristine] = useState(draft)
  const [tried, setTried] = useState(false)
  const [serverErrors, setServerErrors] = useState<Partial<Record<DraftField, FieldError>>>({})
  const [formError, setFormError] = useState<unknown>(null)
  const act = useSaveAndActivate()
  const mounted = useRef(false)

  const dirty = sub.k === 'form' && JSON.stringify(draft) !== JSON.stringify(pristine)
  useEffect(() => onDirty(dirty), [dirty, onDirty])

  // Each sub-step starts at its first control (the first mount is the sheet's);
  // the found-apps list moves focus to its first "Usar" once the scan ends.
  useEffect(() => {
    if (mounted.current) focusFirst()
    mounted.current = true
  }, [sub.k, detected.isPending, focusFirst])

  const startForm = (type: AdapterType, label?: string) => {
    if (FILE_TYPES.has(type)) {
      onNeedsFile(type as 'wireguard' | 'openvpn')
      return
    }
    const next = draftFor(type, label, names)
    setDraft(next)
    setPristine(next)
    setTried(false)
    setServerErrors({})
    setFormError(null)
    setSub({ k: 'form', type })
  }

  if (sub.k === 'list') {
    const known = (detected.data ?? []).filter((d) => isKnownType(d.adapter_type))
    const unknown = (detected.data ?? []).filter((d) => !isKnownType(d.adapter_type))
    const seen = new Set<string>()
    // One row per type; the app whose tunnel is up right now comes first (it is the one in use).
    const apps = [...known]
      .sort((a, b) => Number(b.active === true) - Number(a.active === true))
      .filter((d) => !seen.has(d.adapter_type) && !!seen.add(d.adapter_type))
    return (
      <div className="k-add__step">
        <h3 className="k-add__h">{t('add.vpn.found.title')}</h3>
        <div aria-live="polite" aria-busy={detected.isPending || undefined}>
          {detected.isPending ? (
            <p className="k-add__muted">{t('add.vpn.detecting')}</p>
          ) : apps.length === 0 ? (
            <p className="k-add__muted">{t('add.vpn.none')}</p>
          ) : (
            <ul className="k-add-apps">
              {apps.map((d, i) => {
                const spec = specOf(d.adapter_type)
                const existing = profiles.find((p) => p.type === spec.type)
                return (
                  <li key={d.adapter_type} className="k-add-app">
                    <Monogram spec={spec} />
                    <span className="k-add-app__text">
                      <span className="k-add-app__name">{d.label || spec.label}</span>
                      {d.active === true && <StatusPill tone="ok">{t('add.vpn.activeNow')}</StatusPill>}
                      {existing && <span className="k-add__muted">{t('add.vpn.already', { name: existing.name })}</span>}
                    </span>
                    <Button
                      variant={i === 0 ? 'filled' : 'tonal'}
                      size="sm"
                      aria-label={t('add.vpn.useApp', { app: d.label || spec.label })}
                      data-autofocus={i === 0 || undefined}
                      onClick={() => startForm(spec.type, d.label || spec.label)}
                    >
                      {t('add.vpn.use')}
                    </Button>
                  </li>
                )
              })}
            </ul>
          )}
          {unknown.length > 0 && (
            <p className="k-add__muted k-add__small">{t('add.vpn.unsupported', { apps: unknown.map((d) => d.label).join(', ') })}</p>
          )}
        </div>
        <footer className="k-dialog__actions k-add__actions">
          <Button variant="outlined" icon={<ArrowLeftIcon />} onClick={onBack}>
            {t('add.back')}
          </Button>
          <Button variant="tonal" onClick={() => setSub({ k: 'pick' })} data-autofocus={(!detected.isPending && apps.length === 0) || undefined}>
            {t('add.vpn.other')}
          </Button>
        </footer>
      </div>
    )
  }

  if (sub.k === 'pick') {
    return (
      <div className="k-add__step k-add__pick">
        <p className="k-add__muted">{t('add.vpn.pickLead')}</p>
        <TypePicker detected={detected.data ?? []} onPick={(type) => startForm(type)} />
        <footer className="k-dialog__actions k-add__actions">
          <Button variant="outlined" icon={<ArrowLeftIcon />} onClick={() => setSub({ k: 'list' })} data-autofocus="">
            {t('add.back')}
          </Button>
        </footer>
      </div>
    )
  }

  const spec = specOf(sub.type)
  const problems = tried ? problemsOf(draft, profiles) : {}
  const err = (f: DraftField): FieldError | null => {
    const p = problems[f]
    if (p) return { message: t(p.key, p.vars), fix: t(`${p.key}.fix` as MessageKey, p.vars) }
    return serverErrors[f] ?? null
  }
  const text = (f: DraftField) => ({
    value: draft[f] as string,
    onChange: (e: { target: { value: string } }) => {
      setDraft((d) => ({ ...d, [f]: e.target.value }))
      setServerErrors((s) => ({ ...s, [f]: undefined }))
    },
    error: err(f),
  })
  const hasAuth = spec.auth !== 'none'
  const moreFields: DraftField[] = ['username', 'password', 'country', 'city', 'host']
  const moreProblem = moreFields.some((f) => f !== 'host' && err(f))

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (act.busy) return
    setTried(true)
    setFormError(null)
    if (Object.keys(problemsOf(draft, profiles)).length > 0) {
      requestAnimationFrame(() => document.querySelector<HTMLElement>('.k-add [aria-invalid="true"]')?.focus())
      return
    }
    try {
      const ok = await act.run(toRequest(draft, 'create'))
      onDirty(false)
      if (ok) onDone(draft.name.trim())
    } catch (error) {
      const local = localizeError(error, t)
      const field = local instanceof ApiError ? fieldForErrorKey(local.key) : undefined
      if (field) {
        const { message, fix } = describeError(local, t)
        setServerErrors({ [field]: { message, fix } })
      } else setFormError(local)
    }
  }

  return (
    <form className="k-add__step" onSubmit={(e) => void submit(e)} noValidate>
      <div className="k-add__typehead">
        <Monogram spec={spec} size="lg" />
        <div>
          <p className="k-add__typename">{spec.label}</p>
          <p className="k-add__muted k-add__small">{t('add.vpn.formLead')}</p>
        </div>
      </div>
      {formError !== null && <ErrorBanner error={formError} />}
      <TextField
        label={t('add.field.name')}
        hint={t('add.field.name.hint')}
        required
        autoComplete="off"
        spellCheck={false}
        data-autofocus=""
        {...text('name')}
      />
      {spec.host === 'required' && (
        <TextField
          label={t('profiles.field.host')}
          hint={t('profiles.hint.host')}
          placeholder={t('profiles.placeholder.host')}
          required
          mono
          spellCheck={false}
          {...text('host')}
        />
      )}
      {spec.auth === 'none' && <p className="k-add__muted k-add__small">{t('profiles.signin.app', { app: spec.label })}</p>}
      {(hasAuth || spec.location || spec.autoConnect || spec.host === 'optional') && (
        <details className="k-add-more" open={moreProblem || undefined}>
          <summary>{t('add.more')}</summary>
          <div className="k-add-more__body">
            {hasAuth && (
              <>
                <TextField label={t('profiles.field.username')} autoComplete="off" spellCheck={false} {...text('username')} />
                <TextField
                  label={t('profiles.field.password')}
                  hint={t('profiles.hint.password')}
                  type="password"
                  autoComplete="new-password"
                  {...text('password')}
                />
              </>
            )}
            {spec.host === 'optional' && <TextField label={t('profiles.field.exitNode')} mono spellCheck={false} {...text('host')} />}
            {spec.location && (
              <>
                <TextField label={t('profiles.field.country')} placeholder={t('profiles.placeholder.country')} {...text('country')} />
                <TextField label={t('profiles.field.city')} {...text('city')} />
              </>
            )}
            {spec.autoConnect && (
              <Switch
                checked={draft.autoConnect}
                onChange={(v) => setDraft((d) => ({ ...d, autoConnect: v }))}
                label={t('profiles.field.autoConnect')}
                description={t('profiles.hint.autoConnect')}
                onText={t('profiles.yes')}
                offText={t('profiles.no')}
              />
            )}
          </div>
        </details>
      )}
      <ActivateFooter
        phase={act.phase}
        onBack={() => (initialType ? onBack() : setSub({ k: 'list' }))}
        onRetry={() =>
          void act.retryRestart().then((ok) => {
            if (ok) onDone(draft.name.trim())
          })
        }
      />
    </form>
  )
}

function draftFor(type: AdapterType | undefined, label: string | undefined, taken: readonly string[]): Draft {
  const d = emptyDraft(type ?? 'nordvpn')
  const base = cleanName(label ?? (type ? specOf(type).label : ''))
  return { ...d, name: base ? uniqueName(base, taken) : '' }
}

/** checkDraft (the form's own checks) plus the daemon's name rule. */
function problemsOf(draft: Draft, profiles: readonly VPNProfile[]) {
  const out = checkDraft(draft, { profiles })
  // The quick form has no file, failover or priority: those checks belong to Perfiles.
  delete out.config
  delete out.failover
  delete out.priority
  const name = draft.name.trim()
  if (name && !out.name && !isValidName(name)) out.name = { key: 'add.err.name.chars' as const }
  if (!name) out.name = { key: 'add.err.name.required' as const }
  return out
}
