// Reglas — edit one rule, inline (not a modal) · v1.2.0 · 2026-11-24
//
// The editor shares the quick form's destination syntax. New rules start in
// the shell's "＋ Agregar" sheet (ADR-0055). Validation
// on submit, then live; focus goes to the first problem. A refusal from
// the daemon stays next to the form (and as a toast).
import { useId, useRef, useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { ApiError } from '../../api/client'
import { usePreferences } from '../../app/preferences-context'
import { useUnsavedChanges } from '../../app/unsaved'
import { useProfilesQuery } from '../../data/profiles'
import { useSavePolicy, useTestDraft } from '../../data/rules'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { SelectField, TextAreaField, TextField, type FieldError } from '../ui/Field'
import { ErrorState } from '../ui/States'
import { CheckIcon, InfoIcon } from '../ui/icons'
import { POLICY_FIELDS, validatePolicy, type PolicyDraft, type PolicyErrors } from './editorModel'
import { TUNNEL_DOWN_CHOICES, priorityLink, splitTestInput, type Policy } from './model'
import { policyErrorText } from './preview'

interface Props {
  /** The saved name (renames supported). New rules start in the "＋ Agregar" sheet (ADR-0055). */
  original: string
  initial: PolicyDraft
  policies: readonly Policy[]
  profiles: readonly string[]
  onClose: () => void
}

export function PolicyEditor({ original, initial, policies, profiles, onClose }: Props) {
  const { t } = usePreferences()
  const save = useSavePolicy()
  const formRef = useRef<HTMLFormElement>(null)
  const [draft, setDraft] = useState<PolicyDraft>(initial)
  useUnsavedChanges(JSON.stringify(draft) !== JSON.stringify(initial))
  const [submitted, setSubmitted] = useState(false)
  const [serverError, setServerError] = useState<unknown>(null)
  const titleId = useId()

  const profileAction = useProfilesQuery().data?.find((p) => p.name === draft.via)?.on_tunnel_down ?? 'allow'
  const ctx = { names: policies.map((p) => p.name), editing: original, profiles, profileAction }
  const { errors: found, payload, hasApps } = validatePolicy(draft, ctx)
  const errors: PolicyErrors = submitted ? found : {}
  const err = (f: keyof PolicyErrors) => (errors[f] ? policyErrorText(f, errors[f], t) : null)

  const set = <K extends keyof PolicyDraft>(k: K, v: PolicyDraft[K]) => setDraft((d) => ({ ...d, [k]: v }))

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (save.isPending) return
    setSubmitted(true)
    setServerError(null)
    if (!payload) {
      const first = POLICY_FIELDS.find((f) => found[f])
      formRef.current?.querySelector<HTMLElement>(`[name="${first}"]`)?.focus()
      return
    }
    try {
      await save.mutateAsync({ original, payload })
      onClose()
    } catch (error) {
      setServerError(error)
    }
  }

  // Name taken on the daemon side (another client added it meanwhile).
  const conflict = serverError instanceof ApiError && serverError.status === 409
  const nameError = err('name') ?? (conflict ? policyErrorText('name', { code: 'duplicate', value: draft.name }, t) : null)

  return (
    <Card
      className="r-editor"
      title={
        <span id={titleId} tabIndex={-1} data-autofocus>
          {t('rules.editor.editTitle', { name: original })}
        </span>
      }
    >
      <form ref={formRef} className="r-form" noValidate onSubmit={onSubmit} aria-labelledby={titleId}>
        <div className="r-form__grid">
          <TextField
            name="name"
            label={t('rules.field.name')}
            placeholder={t('rules.field.name.placeholder')}
            value={draft.name}
            onChange={(e) => set('name', e.target.value)}
            error={nameError}
            required
            autoComplete="off"
          />
          {profiles.length === 0 ? (
            <div className="k-field">
              <span className="k-field__label">{t('rules.field.via')}</span>
              <p className="k-field__error">
                <strong>{t('rules.err.via.noProfiles')}</strong> <Link to="/profiles">{t('rules.err.via.noProfiles.fix')}</Link>
              </p>
            </div>
          ) : (
            <SelectField
              name="via"
              label={t('rules.field.via')}
              hint={t('rules.field.via.hint')}
              value={draft.via}
              onChange={(v) => set('via', v)}
              error={err('via')}
              required
              mono
              placeholder={t('rules.field.via.pick')}
              options={profiles.map((p) => ({ value: p, label: p }))}
            />
          )}
        </div>
        <TextAreaField
          name="destinations"
          label={t('add.rule.input')}
          hint={t('add.rule.input.hint')}
          placeholder={'example.com, 192.0.2.0/24\nhttps://git.example.org/'}
          value={draft.destinations}
          onChange={(e) => set('destinations', e.target.value)}
          error={err('destinations')}
          mono
          rows={4}
          spellCheck={false}
          autoCapitalize="off"
        />
        {hasApps && <p className="r-note">{t('policy.app.notApplied')}</p>}

        <fieldset className="r-choice" aria-describedby={errors.onDown ? `${titleId}-ondown-err` : undefined}>
          <legend className="k-field__label">{t('rules.field.onDown')}</legend>
          {TUNNEL_DOWN_CHOICES.map((v) => (
            <label key={v} className="r-choice__item">
              <input
                type="radio"
                name="onDown"
                value={v}
                checked={draft.onDown === v}
                onChange={() => set('onDown', v)}
              />
              <span>
                <strong>{t(`rules.onDown.${v}` as MessageKey)}</strong>
                <span className="r-choice__hint">{t(`rules.onDown.${v}.hint` as MessageKey)}</span>
              </span>
            </label>
          ))}
          {errors.onDown && (
            <p id={`${titleId}-ondown-err`} className="k-field__error">
              <strong>{policyErrorText('onDown', errors.onDown, t).message}</strong>{' '}
              <span>{policyErrorText('onDown', errors.onDown, t).fix}</span>
            </p>
          )}
        </fieldset>

        <DraftTest draft={draft} ctx={ctx} />

        <p className="r-note">
          {t('rules.editor.priority')} <Link to={draft.via ? priorityLink(draft.via) : '/profiles'}>{t('rules.editor.priority.link')}</Link>
        </p>

        {serverError !== null && !conflict && <ErrorState error={serverError} />}

        <div className="r-form__actions">
          <Button type="submit" loading={save.isPending}>
            {t('rules.action.saveChanges')}
          </Button>
          <Button variant="text" onClick={onClose}>
            {t('rules.action.cancel')}
          </Button>
        </div>
      </form>
    </Card>
  )
}

/** "Probar antes de guardar": the daemon matches the unsaved draft against one input. */
function DraftTest({ draft, ctx }: { draft: PolicyDraft; ctx: Parameters<typeof validatePolicy>[1] }) {
  const { t } = usePreferences()
  const test = useTestDraft()
  const [input, setInput] = useState('')
  const [problem, setProblem] = useState<FieldError | null>(null)

  const run = () => {
    setProblem(null)
    test.reset()
    if (!input.trim()) {
      setProblem({ message: t('rules.test.err.empty'), fix: t('rules.test.err.empty.fix') })
      return
    }
    // Only the match fields matter for a dry run; name and VPN get placeholders.
    const { payload } = validatePolicy({ ...draft, name: draft.name || 'test', via: draft.via || ctx.profiles[0] || '' }, { ...ctx, names: [] })
    if (!payload) {
      setProblem({ message: t('rules.test.err.draft'), fix: t('rules.test.err.draft.fix') })
      return
    }
    const { target, app } = splitTestInput(input)
    test.mutate({ rule: payload, target, app })
  }

  const result = test.data
  return (
    <div className="r-drafttest">
      <div className="r-inline-form">
        <TextField
          label={t('rules.test.draft.label')}
          hint={t('rules.test.input.hint')}
          placeholder={t('rules.test.input.placeholder')}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              run()
            }
          }}
          error={problem}
          mono
          spellCheck={false}
          autoCapitalize="off"
        />
        <Button variant="outlined" onClick={run} loading={test.isPending}>
          {t('rules.test.run')}
        </Button>
      </div>
      <div aria-live="polite" className="r-drafttest__result">
        {result && (
          <p className={`r-verdict ${result.matched ? 'is-match' : 'is-miss'}`}>
            {result.matched ? <CheckIcon /> : <InfoIcon />}
            <span>
              {result.matched
                ? t('rules.test.draft.match', { via: result.via || draft.via })
                : t('rules.test.draft.miss')}
            </span>
          </p>
        )}
        {test.isError && <ErrorState error={test.error} onRetry={run} />}
      </div>
    </div>
  )
}
