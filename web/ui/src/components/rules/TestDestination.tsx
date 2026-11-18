// Reglas — "Probar un destino" (GUI of `kongtrol policy show`) · v1.0.0 · 2026-11-18
//
// Asks the daemon which saved rule a site, IP, URL or app uses and why
// (GET /api/v1/policies/explain), then adds what happens right now from
// the VPN's live state and the rule's on_tunnel_down. Drawn as a calm
// three-step route (destination → rule → exit), the landing's routing
// diagram in Operate form: static, no motion beyond a short fade, and a
// stacked list on narrow screens. The technical detail (reason, CLI
// equivalent) is one click away.
import { useState, type FormEvent } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { useExplain } from '../../data/rules'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { TextField } from '../ui/Field'
import { ErrorState } from '../ui/States'
import { StatusPill } from '../ui/StatusPill'
import { StepArrow } from './icons'
import { onDownOf, routeOutcome, testSuggestions, type ExplainResult, type Policy, type RouteOutcome } from './model'
import { useProfileState } from './hooks'
import { reasonText } from './preview'
import { VpnState } from './shared'

export function TestDestination({ policies }: { policies: readonly Policy[] }) {
  const { t } = usePreferences()
  const [input, setInput] = useState('')
  const [asked, setAsked] = useState('')
  const [empty, setEmpty] = useState(false)
  const explain = useExplain(asked)
  const suggestions = testSuggestions(policies)

  const ask = (value: string) => {
    const v = value.trim()
    setEmpty(!v)
    if (v) setAsked(v)
  }
  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    ask(input)
  }

  return (
    <Card title={t('rules.test.title')} className="r-test">
      <p className="muted">{t('rules.test.what')}</p>
      <form className="r-inline-form" onSubmit={onSubmit} noValidate>
        <TextField
          label={t('rules.test.input.label')}
          hint={t('rules.test.input.hint')}
          placeholder={t('rules.test.input.placeholder')}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          error={empty ? { message: t('rules.test.err.empty'), fix: t('rules.test.err.empty.fix') } : null}
          mono
          spellCheck={false}
          autoCapitalize="off"
        />
        <Button type="submit" variant="tonal" loading={explain.isFetching}>
          {t('rules.test.run')}
        </Button>
      </form>
      {suggestions.length > 0 && (
        <div className="r-suggest">
          <span className="r-suggest__label">{t('rules.test.try')}</span>
          {suggestions.map((s) => (
            <button
              key={s}
              type="button"
              className="r-chip"
              onClick={() => {
                setInput(s)
                ask(s)
              }}
            >
              {s}
            </button>
          ))}
        </div>
      )}
      <div aria-live="polite">
        {explain.isError && <ErrorState error={explain.error} onRetry={() => void explain.refetch()} />}
        {explain.data && <RouteFlow ex={explain.data} policies={policies} />}
      </div>
    </Card>
  )
}

const EXIT_KEY: Record<RouteOutcome, MessageKey> = {
  none: 'rules.flow.exit.normal',
  app: 'rules.flow.exit.app',
  via: 'rules.flow.exit.via',
  connecting: 'rules.flow.exit.via',
  noResponse: 'rules.flow.exit.via',
  normal: 'rules.flow.exit.normal',
  blocked: 'rules.flow.exit.blocked',
  connect: 'rules.flow.exit.connect',
}

export function RouteFlow({ ex, policies }: { ex: ExplainResult; policies: readonly Policy[] }) {
  const { t } = usePreferences()
  const rule = policies.find((p) => p.name === ex.RuleName)
  const state = useProfileState(ex.Via)
  const outcome = routeOutcome(ex, rule ? onDownOf(rule) : 'allow', state)
  const vars = { via: ex.Via, rule: ex.RuleName, target: ex.Target }
  const isApp = ex.Kind === 'app'

  return (
    <div className="r-flow" data-outcome={outcome} key={`${ex.Target}|${outcome}`}>
      <ol className="r-flow__steps" aria-label={t('rules.flow.label', { target: ex.Target })}>
        <li className="r-step">
          <span className="r-step__label">{t(isApp ? 'rules.flow.app' : 'rules.flow.destination')}</span>
          <span className="r-step__value mono">{ex.Target}</span>
        </li>
        <li className="r-flow__arrow" aria-hidden="true">
          <StepArrow />
        </li>
        <li className={`r-step ${ex.Matched ? '' : 'is-empty'}`}>
          <span className="r-step__label">{t('rules.flow.rule')}</span>
          {ex.Matched ? (
            <>
              <span className="r-step__value">{t('rules.quote', { name: ex.RuleName })}</span>
              <span className="r-step__why">{reasonText(ex, t)}</span>
            </>
          ) : (
            <span className="r-step__value">{t('rules.flow.noRule')}</span>
          )}
        </li>
        <li className="r-flow__arrow" aria-hidden="true">
          <StepArrow />
        </li>
        <li className="r-step r-step--exit" data-outcome={outcome}>
          <span className="r-step__label">{t('rules.flow.exit')}</span>
          {outcome === 'via' || outcome === 'connecting' || outcome === 'noResponse' ? (
            <VpnState name={ex.Via} />
          ) : outcome === 'blocked' ? (
            <StatusPill tone="bad">{t('rules.flow.exit.blocked')}</StatusPill>
          ) : (
            <span className="r-step__value">{t(EXIT_KEY[outcome], vars)}</span>
          )}
        </li>
      </ol>
      <p className="r-flow__sentence">{t(`rules.outcome.${outcome}` as MessageKey, vars)}</p>
      <details className="r-disclosure r-disclosure--tech">
        <summary>{t('rules.tech.toggle')}</summary>
        <dl className="r-tech">
          <dt>{t('rules.tech.kind')}</dt>
          <dd className="mono">{ex.Kind || '—'}</dd>
          <dt>{t('rules.tech.reason')}</dt>
          <dd>{reasonText(ex, t)}</dd>
          {rule && (
            <>
              <dt>{t('rules.tech.onDown')}</dt>
              <dd className="mono">on_tunnel_down: {onDownOf(rule)}</dd>
            </>
          )}
          <dt>{t('rules.tech.cli')}</dt>
          <dd>
            <code>kongtrol policy show {isApp ? `app:${ex.Target}` : ex.Target}</code>
          </dd>
        </dl>
      </details>
    </div>
  )
}
