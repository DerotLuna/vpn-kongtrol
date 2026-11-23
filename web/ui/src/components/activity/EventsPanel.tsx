// Kongtrol UI — Actividad: daemon notices timeline · v1.1.0 · 2026-11-19
//
// The notices the tray and the classic panel show as toasts (GET
// /api/v1/events), kept as a readable history: newest first, grouped by
// day, each in plain language from its i18n key (never pre-rendered by the
// daemon). A notice about traffic for a tunnel that is off offers "Conectar"
// in place, through the same optimistic tunnel action as Inicio; every
// notice about a profile offers "＋" (send a site through it, ADR-0055).
import { memo, useCallback, useMemo } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { eventText, useEventsHistory, type DaemonEvent } from '../../data/activity'
import { useTunnelAction } from '../../data/resources'
import { AddRuleIcon } from '../add/entries'
import { OfferActions } from '../home/OfferActions'
import { Button } from '../ui/Button'
import { AlertIcon, InfoIcon, PlugIcon } from '../ui/icons'
import { EmptyState, ErrorState, Skeleton } from '../ui/States'
import { StatusPill } from '../ui/StatusPill'
import './activity.css'

function dayKey(iso: string): string {
  const d = new Date(iso)
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
}

const EventRow = memo(function EventRow({
  ev,
  onConnect,
  onDisconnect,
  busy,
}: {
  ev: DaemonEvent
  onConnect: (name: string) => void
  onDisconnect: (name: string) => void
  busy: boolean
}) {
  const { t, lang } = usePreferences()
  const warn = ev.level === 'WARN'
  const time = new Intl.DateTimeFormat(lang, { timeStyle: 'short' }).format(new Date(ev.time))
  return (
    <li className="k-ev" data-level={warn ? 'warn' : 'info'}>
      <span className="k-ev__icon" aria-hidden="true">
        {warn ? <AlertIcon /> : <InfoIcon />}
      </span>
      <div className="k-ev__body">
        <p className="k-ev__msg">{eventText(ev, t)}</p>
        <p className="k-ev__meta">
          <time dateTime={ev.time} className="mono">
            {time}
          </time>
          <StatusPill tone={warn ? 'caution' : 'idle'}>{warn ? t('activity.level.warn') : t('activity.level.info')}</StatusPill>
          {ev.profile && <span className="mono k-ev__profile">{ev.profile}</span>}
        </p>
      </div>
      {ev.profile && (
        <div className="k-ev__actions">
          {ev.can_connect && (
            <Button variant="outlined" size="sm" icon={<PlugIcon />} loading={busy} onClick={() => onConnect(ev.profile!)}>
              {ev.key.startsWith('watchdog.event.')
                ? t('activity.event.watchdog.event.action_retry', { 0: ev.profile })
                : t('activity.event.policy.event.action_connect', { 0: ev.profile })}
            </Button>
          )}
          <OfferActions event={ev} onDisconnect={onDisconnect} />
          <AddRuleIcon via={ev.profile} />
        </div>
      )}
    </li>
  )
})

export function EventsPanel() {
  const { t, lang } = usePreferences()
  const { data, error, refetch, dataUpdatedAt } = useEventsHistory()
  const action = useTunnelAction()
  const connecting = action.isPending ? action.variables?.name : undefined

  const groups = useMemo(() => {
    const out: { key: string; label: string; items: DaemonEvent[] }[] = []
    // "Hoy" / "Ayer" relative to the last read (renders stay pure).
    const now = new Date(dataUpdatedAt)
    const today = dayKey(now.toISOString())
    const yesterday = dayKey(new Date(now.getTime() - 86_400_000).toISOString())
    const fmt = new Intl.DateTimeFormat(lang, { dateStyle: 'full' })
    for (const ev of data?.events ?? []) {
      const k = dayKey(ev.time)
      let g = out.at(-1)
      if (!g || g.key !== k) {
        const label = k === today ? t('activity.events.today') : k === yesterday ? t('activity.events.yesterday') : fmt.format(new Date(ev.time))
        g = { key: k, label, items: [] }
        out.push(g)
      }
      g.items.push(ev)
    }
    return out
  }, [data, dataUpdatedAt, lang, t])

  const { mutate } = action
  const onConnect = useCallback((name: string) => mutate({ name, action: 'connect' }), [mutate])
  const onDisconnect = useCallback((name: string) => mutate({ name, action: 'disconnect' }), [mutate])

  return (
    <section className="k-act" aria-labelledby="events-title" aria-busy={!data && !error}>
      <header className="k-act__head">
        <div>
          <h2 id="events-title" className="k-act__title">
            {t('activity.events.title')}
          </h2>
          <p className="k-act__what">{t('activity.events.what')}</p>
        </div>
        <span className="k-live" aria-hidden="true">
          <span className="k-live__dot" />
          {t('activity.audit.live')}
        </span>
      </header>

      {error && !data ? (
        <ErrorState error={error} onRetry={() => void refetch()} />
      ) : !data ? (
        <Skeleton lines={4} label={t('activity.events.loading')} />
      ) : data.events.length === 0 ? (
        <EmptyState icon={<InfoIcon />} title={t('activity.events.empty.title')} body={t('activity.events.empty.body')} />
      ) : (
        <div className="k-ev-groups" aria-label={t('activity.events.list')} role="region">
          {groups.map((g) => (
            <section key={g.key} aria-label={g.label}>
              <h3 className="k-ev-day">{g.label}</h3>
              <ul className="k-ev-list">
                {g.items.map((ev) => (
                  <EventRow key={ev.seq} ev={ev} onConnect={onConnect} onDisconnect={onDisconnect} busy={connecting === ev.profile} />
                ))}
              </ul>
            </section>
          ))}
        </div>
      )}
      <p className="k-act__note">{t('activity.events.memory')}</p>
    </section>
  )
}
