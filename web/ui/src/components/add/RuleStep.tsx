// Kongtrol UI — C: "Hacer que un sitio salga por una VPN" · v1.1.0 · 2026-11-24
//
// The daemon previews the first destination before and after activation.
// The entire list is validated before the rule is saved paused.
import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { useDraftRuleTest } from '../../data/add'
import { useExplain, usePoliciesQuery, useSavePolicy } from '../../data/rules'
import type { TunnelMetrics } from '../../api/types'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { SelectField, TextAreaField, TextField, type FieldError } from '../ui/Field'
import { Banner } from '../ui/States'
import { TUNNEL_DOWN_CHOICES, type OnTunnelDown, type PolicyPayload } from '../rules/model'
import { ArrowLeftIcon } from '../profiles/icons'
import { autoName as suggestName, isValidName, matchOf, nameSource, probeOf, readDestinations, type Classified, type Entry, type RuleDraft } from './model'
import { ErrorBanner } from './parts'

export type { RuleDraft }

export interface SavedRule {
  name: string
  via: string
  onDown: OnTunnelDown
  /** What "Probar" asks about. */
  probe: string
}

interface Props {
  draft: RuleDraft
  onChange: (patch: Partial<RuleDraft>) => void
  profiles: readonly string[]
  profileAction: OnTunnelDown
  tunnels: readonly TunnelMetrics[]
  /** Shown once after coming back from adding the VPN. */
  backNote?: string | undefined
  onBack: () => void
  onNeedVpn: () => void
  onDone: (rule: SavedRule) => void
}

function useDebounced<T>(value: T, ms = 400): T {
  const [v, setV] = useState(value)
  useEffect(() => {
    const id = setTimeout(() => setV(value), ms)
    return () => clearTimeout(id)
  }, [value, ms])
  return v
}

type T = ReturnType<typeof usePreferences>['t']

function problemText(c: Classified, t: T): FieldError | null {
  if (c.ok) return null
  const key = `add.err.${c.problem}` as MessageKey
  return { message: t(key, { value: c.value }), fix: t(`${key}.fix` as MessageKey, { value: c.value }) }
}

export function RuleStep({ draft, onChange, profiles, profileAction, tunnels, backNote, onBack, onNeedVpn, onDone }: Props) {
  const { t } = usePreferences()
  const policies = usePoliciesQuery()
  const save = useSavePolicy()
  const [tried, setTried] = useState(false)
  const [refusal, setRefusal] = useState<unknown>(null)
  const taken = useMemo(() => (policies.data ?? []).map((p) => p.name), [policies.data])

  const checked = readDestinations(draft.input)
  const invalid = checked.find(({ result }) => !result.ok)
  const first = checked[0]?.result
  const main = first?.ok ? first : null
  const entries = checked.flatMap(({ result }) => result.ok ? [result.entry] : [])
  const state = (name: string) => tunnels.find((tn) => tn.Name === name)?.Status
  const viaOn = state(draft.via) === 'connected'

  const autoName = main ? suggestName(t('add.rule.autoName', { site: nameSource(main.entry.value), vpn: draft.via || '…' }), taken) : ''
  const name = (draft.name || autoName).trim()

  // Inline while typing for anything typed; "empty" only after trying to save.
  let inputError = invalid ? problemText(invalid.result, t) : null
  if (inputError && invalid && checked.length > 1) {
    inputError.message = t('add.err.line', { line: invalid.line, message: inputError.message })
  } else if (tried && checked.length === 0) {
    inputError = problemText({ ok: false, problem: 'empty', value: '' }, t)
  }
  const hintText = main?.hint ? `${t(`add.hint.${main.hint}` as MessageKey, { value: main.entry.value })} ${t(`add.hint.${main.hint}.fix` as MessageKey, { value: main.entry.value })}` : null
  const viaError: FieldError | null = tried && !draft.via ? { message: t('add.err.via'), fix: t('add.err.via.fix') } : null
  const nameError: FieldError | null = !tried || !main
    ? null
    : !name
      ? { message: t('add.err.name.required'), fix: t('add.err.name.required.fix') }
      : !isValidName(name)
        ? { message: t('add.err.name.chars'), fix: t('add.err.name.chars.fix') }
        : taken.some((n) => n.toLowerCase() === name.toLowerCase())
          ? { message: t('add.err.ruleTaken', { name }), fix: t('add.err.ruleTaken.fix') }
          : null
  const appsOnly = entries.length > 0 && entries.every((e) => e.kind === 'app')
  const onDown = draft.onDown === 'inherit' ? profileAction : draft.onDown
  const downError: FieldError | null =
    tried && onDown !== 'allow' && appsOnly
      ? { message: t('rules.err.onDown.blockAppsOnly'), fix: t('rules.err.onDown.blockAppsOnly.fix') }
      : null

  const invalidOnDown = onDown !== 'allow' && appsOnly
  const payload: PolicyPayload | null =
    main && !invalid && !invalidOnDown && draft.via
      ? { name: name || 'preview', via: draft.via, on_tunnel_down: draft.onDown === 'inherit' ? '' : draft.onDown, match: matchOf(entries, draft.subs) }
      : null

  // Preview: ask once the typing settles.
  const probe = main && !invalid ? probeOf(main.entry) : null
  const settled = useDebounced(JSON.stringify({ probe, payload }))
  const { probe: dProbe, payload: dPayload } = JSON.parse(settled) as { probe: typeof probe; payload: PolicyPayload | null }
  const todayTarget = dProbe && !dProbe.app ? dProbe.target : ''
  const today = useExplain(todayTarget)
  const after = useDraftRuleTest(dPayload, dProbe?.target ?? '', dProbe?.app ?? '')

  if (profiles.length === 0) {
    return (
      <div className="k-add__step">
        <Banner
          tone="info"
          title={t('add.rule.noVpn')}
          actions={
            <Button onClick={onNeedVpn} data-autofocus="">
              {t('add.rule.noVpn.action')}
            </Button>
          }
        >
          <span>{t('add.rule.noVpn.fix')}</span>
        </Banner>
        <footer className="k-dialog__actions k-add__actions">
          <Button variant="text" className="k-add__back" icon={<ArrowLeftIcon />} onClick={onBack}>
            {t('add.back')}
          </Button>
        </footer>
      </div>
    )
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (save.isPending) return
    setTried(true)
    setRefusal(null)
    const blocked =
      !name ||
      !isValidName(name) ||
      taken.some((n) => n.toLowerCase() === name.toLowerCase())
    if (blocked || !payload || !main) {
      requestAnimationFrame(() => document.querySelector<HTMLElement>('.k-add [aria-invalid="true"]')?.focus())
      return
    }
    try {
      await save.mutateAsync({ payload: { ...payload, name, disabled: true } })
      onDone({ name, via: draft.via, onDown, probe: todayTargetOf(main.entry) })
    } catch (err) {
      setRefusal(err)
    }
  }

  const label = main?.entry.label ?? ''
  const afterText = (): string | null => {
    if (!probe || probe.app || !draft.via) return null
    if (!after.data) return after.isFetching ? t('add.rule.checking') : null
    if (!after.data.matched) return t('add.rule.after.miss', { target: label })
    if (viaOn) return t('add.rule.after.via', { target: label, via: draft.via })
    return t(onDown === 'allow' ? 'add.rule.after.offAllow' : 'add.rule.after.offBlock', { target: label, via: draft.via })
  }
  const todayText = (): string | null => {
    // A program goes wherever its destinations go (ADR-0113): no "Hoy" line.
    if (!probe || probe.app) return null
    if (!today.data) return today.isFetching ? t('add.rule.checking') : null
    return today.data.Matched
      ? t('add.rule.today.via', { target: label, via: today.data.Via, rule: today.data.RuleName })
      : t('add.rule.today.normal', { target: label })
  }
  const todayLine = todayText()
  const afterLine = afterText()

  return (
    <form className="k-add__step" onSubmit={(e) => void submit(e)} noValidate>
      {backNote && (
        <Banner tone="ok" title={backNote} />
      )}
      {refusal !== null && <ErrorBanner error={refusal} />}
      <TextAreaField
        label={t('add.rule.input')}
        hint={t('add.rule.input.hint')}
        placeholder={'example.com, 192.0.2.0/24\nhttps://git.example.org/'}
        value={draft.input}
        onChange={(e) => onChange({ input: e.target.value })}
        error={inputError}
        mono
        rows={4}
        autoComplete="off"
        spellCheck={false}
        autoCapitalize="off"
        data-autofocus=""
      />
      {hintText && <p className="k-add__small">{hintText}</p>}
      {entries.some((entry) => entry.kind === 'app') && <p className="k-add__small">{t('policy.app.notApplied')}</p>}
      {entries.some((entry) => entry.kind === 'domain' && !entry.value.startsWith('*.')) && (
        <label className="k-add-check">
          <input type="checkbox" checked={draft.subs} onChange={(e) => onChange({ subs: e.target.checked })} />
          <span>{t('add.rule.subs')}</span>
        </label>
      )}
      <SelectField
        label={t('add.rule.via')}
        value={draft.via}
        onChange={(via) => onChange({ via })}
        error={viaError}
        mono
        placeholder={t('add.rule.via.pick')}
        options={profiles.map((p) => ({ value: p, label: t(state(p) === 'connected' ? 'add.rule.via.on' : 'add.rule.via.off', { name: p }) }))}
      />
      <fieldset className="k-add-radios" aria-invalid={downError ? true : undefined}>
        <legend className="k-field__label">{t('add.rule.down')}</legend>
        {TUNNEL_DOWN_CHOICES.map((v) => (
          <label key={v} className="k-add-radio">
            <input type="radio" name="add-ondown" value={v} checked={draft.onDown === v} onChange={() => onChange({ onDown: v })} />
            <span>{t(v === 'allow' || v === 'block' ? `add.rule.down.${v}` : `rules.onDown.${v}`)}</span>
          </label>
        ))}
        {downError && (
          <p className="k-field__error">
            <strong>{downError.message}</strong> <span>{downError.fix}</span>
          </p>
        )}
      </fieldset>

      {(todayLine || afterLine) && (
        <section className="k-add-preview" aria-label={t('add.rule.preview')} aria-live="polite">
          {todayLine && (
            <p>
              <strong>{t('add.rule.today')}</strong> {todayLine}
            </p>
          )}
          {afterLine && (
            <p>
              <strong>{t('add.rule.after')}</strong> {afterLine}
            </p>
          )}
        </section>
      )}

      <details className="k-add-more" open={!!nameError || undefined}>
        <summary>{t('add.more')}</summary>
        <div className="k-add-more__body">
          <TextField
            label={t('add.rule.name')}
            hint={t('add.rule.name.hint')}
            value={draft.name || autoName}
            onChange={(e) => onChange({ name: e.target.value })}
            error={nameError}
            autoComplete="off"
          />
        </div>
      </details>

      <footer className="k-dialog__actions k-add__actions">
        <Button variant="text" className="k-add__back" icon={<ArrowLeftIcon />} onClick={onBack}>
          {t('add.back')}
        </Button>
        <Button type="submit" loading={save.isPending}>
          {t('add.rule.save')}
        </Button>
      </footer>
    </form>
  )
}

/** "Probar" asks GET /api/v1/policies/explain about the first entry. */
function todayTargetOf(e: Entry): string {
  const p = probeOf(e)
  return p.app ? `app:${p.app}` : p.target
}
