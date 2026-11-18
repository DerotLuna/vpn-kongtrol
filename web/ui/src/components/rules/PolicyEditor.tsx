// Reglas — create / edit one rule, inline (not a modal) · v1.0.0 · 2026-11-18
//
// Plain fields first (name, VPN, sites, networks, what happens when the
// VPN is off), apps behind a disclosure because the daemon only uses them
// to test and diagnose today. Starting points for people who do not write
// globs or CIDRs; raw one-per-line editing for everyone else. Validation
// on submit, then live; focus goes to the first problem. A refusal from
// the daemon stays next to the form (and as a toast).
import { useId, useRef, useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { ApiError } from '../../api/client'
import { usePreferences } from '../../app/preferences-context'
import { useSavePolicy, useTestDraft } from '../../data/rules'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { SelectField, TextAreaField, TextField, type FieldError } from '../ui/Field'
import { ErrorState } from '../ui/States'
import { CheckIcon, InfoIcon } from '../ui/icons'
import {
  hasErrors,
  ON_TUNNEL_DOWN,
  POLICY_FIELDS,
  splitTestInput,
  templateDraft,
  toPayload,
  validatePolicy,
  type Policy,
  type PolicyDraft,
  type PolicyErrors,
  type TemplateId,
} from './model'
import { policyErrorText } from './preview'

interface Props {
  /** Undefined = new rule. */
  original?: string | undefined
  initial: PolicyDraft
  policies: readonly Policy[]
  profiles: readonly string[]
  onClose: () => void
}

const TEMPLATE_IDS: readonly TemplateId[] = ['company', 'network', 'site']

export function PolicyEditor({ original, initial, policies, profiles, onClose }: Props) {
  const { t } = usePreferences()
  const save = useSavePolicy()
  const formRef = useRef<HTMLFormElement>(null)
  const [draft, setDraft] = useState<PolicyDraft>(initial)
  const [submitted, setSubmitted] = useState(false)
  const [serverError, setServerError] = useState<unknown>(null)
  const [template, setTemplate] = useState<TemplateId | null>(null)
  const titleId = useId()

  const ctx = { names: policies.map((p) => p.name), editing: original, profiles }
  const errors: PolicyErrors = submitted ? validatePolicy(draft, ctx) : {}
  const err = (f: keyof PolicyErrors) => (errors[f] ? policyErrorText(f, errors[f], t) : null)

  const set = <K extends keyof PolicyDraft>(k: K, v: PolicyDraft[K]) => setDraft((d) => ({ ...d, [k]: v }))

  const applyTemplate = (id: TemplateId) => {
    setTemplate(id)
    const names = { company: t('rules.template.domain'), site: t('rules.template.siteDomain') }
    setDraft((d) => ({
      ...d,
      ...templateDraft(id, names),
      name: d.name || t(`rules.template.${id}.name` as MessageKey),
    }))
  }

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSubmitted(true)
    setServerError(null)
    const found = validatePolicy(draft, ctx)
    if (hasErrors(found)) {
      const first = POLICY_FIELDS.find((f) => found[f])
      formRef.current?.querySelector<HTMLElement>(`[name="${first}"]`)?.focus()
      return
    }
    try {
      await save.mutateAsync({ original, payload: toPayload(draft) })
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
          {original ? t('rules.editor.editTitle', { name: original }) : t('rules.editor.newTitle')}
        </span>
      }
    >
      {!original && (
        <div className="r-templates">
          <p className="r-templates__lead">{t('rules.template.lead')}</p>
          <div className="r-templates__list">
            {TEMPLATE_IDS.map((id) => (
              <button
                key={id}
                type="button"
                className="r-template"
                aria-pressed={template === id}
                onClick={() => applyTemplate(id)}
              >
                <strong>{t(`rules.template.${id}` as MessageKey)}</strong>
                <span>{t(`rules.template.${id}.what` as MessageKey)}</span>
              </button>
            ))}
          </div>
          {template && (
            <p className="r-callout" role="status">
              <InfoIcon />
              <span>{t(`rules.template.${template}.edit` as MessageKey)}</span>
            </p>
          )}
        </div>
      )}

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
              onChange={(e) => set('via', e.target.value)}
              error={err('via')}
              required
            >
              <option value="">{t('rules.field.via.pick')}</option>
              {profiles.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </SelectField>
          )}
          <TextAreaField
            name="domains"
            label={t('rules.field.domains')}
            hint={t('rules.field.domains.hint')}
            placeholder={`${t('rules.template.domain')}\n*.${t('rules.template.domain')}`}
            value={draft.domains}
            onChange={(e) => set('domains', e.target.value)}
            error={err('domains')}
            mono
            rows={4}
            spellCheck={false}
            autoCapitalize="off"
          />
          <TextAreaField
            name="ipRanges"
            label={t('rules.field.ipRanges')}
            hint={t('rules.field.ipRanges.hint')}
            placeholder={'10.0.0.0/8\n192.0.2.10'}
            value={draft.ipRanges}
            onChange={(e) => set('ipRanges', e.target.value)}
            error={err('ipRanges')}
            mono
            rows={4}
            spellCheck={false}
            autoCapitalize="off"
          />
        </div>

        <fieldset className="r-choice" aria-describedby={errors.onDown ? `${titleId}-ondown-err` : undefined}>
          <legend className="k-field__label">{t('rules.field.onDown')}</legend>
          {ON_TUNNEL_DOWN.map((v) => (
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

        <details className="r-disclosure" open={draft.apps.trim() !== '' || undefined}>
          <summary>{t('rules.field.apps.toggle')}</summary>
          <p className="r-disclosure__note">{t('rules.field.apps.note')}</p>
          <TextAreaField
            name="apps"
            label={t('rules.field.apps')}
            hint={t('rules.field.apps.hint')}
            placeholder={'chrome.exe\nteams*'}
            value={draft.apps}
            onChange={(e) => set('apps', e.target.value)}
            mono
            rows={3}
            spellCheck={false}
            autoCapitalize="off"
          />
        </details>

        <DraftTest draft={draft} ctx={ctx} />

        <p className="r-note">
          {t('rules.editor.priority')} <Link to="/profiles">{t('rules.editor.priority.link')}</Link>
        </p>

        {serverError !== null && !conflict && <ErrorState error={serverError} />}

        <div className="r-form__actions">
          <Button type="submit" loading={save.isPending}>
            {original ? t('rules.action.saveChanges') : t('rules.action.create')}
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
    const found = validatePolicy({ ...draft, name: draft.name || 'test', via: draft.via || ctx.profiles[0] || '' }, { ...ctx, names: [] })
    if (hasErrors(found)) {
      setProblem({ message: t('rules.test.err.draft'), fix: t('rules.test.err.draft.fix') })
      return
    }
    const payload = toPayload({ ...draft, name: draft.name || 'test', via: draft.via || ctx.profiles[0] || '' })
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
