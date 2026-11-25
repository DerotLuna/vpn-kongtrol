// Kongtrol UI — Actividad: signed audit log viewer · v1.3.0 · 2026-11-25
//
// GET /api/v1/audit (same reader as `kongtrol logs`, ADR-0042). Level,
// profiles (any of the chosen ones) and dates filter on the daemon (from / to = local days, across
// every retained day). Read like Tráfico › Recientes (auditView.ts): 10
// entries, then "Ver 20 más"; when the loaded entries run out it asks for
// the next older page with the `before` cursor while X-Kongtrol-Audit-More
// says there is one. "En vivo" keeps new entries behind "N nuevas ·
// Mostrar" instead of pushing the rows being read. Level
// names are translated; an unconfigured log is explained (the API answers
// 200 [] for it) and any other error says what happened. Each entry shows
// its signature in a word + icon: "Firma válida", "Firma no válida"
// (danger: edited, or signed with another key) or "Sin firma".
import { useMemo, useRef, useState } from 'react'
import { usePreferences } from '../../app/preferences-context'
import {
  AUDIT_LEVELS,
  auditToJSONL,
  downloadText,
  signatureState,
  useAuditLog,
  type AuditEvent,
  type AuditLevel,
  type SignatureState,
} from '../../data/activity'
import type { Settings } from '../../data/settings'
import { toasts } from '../../data/toast'
import type { MessageKey } from '../../i18n/messages'
import { Button, IconButton } from '../ui/Button'
import { DataList, type Column } from '../ui/DataList'
import { DateRangeField, MultiSelectField, SelectField } from '../ui/Field'
import { useProfilesQuery } from '../../data/profiles'
import { AlertIcon, CheckIcon, InfoIcon, RefreshIcon } from '../ui/icons'
import { Banner, EmptyState, ErrorState, Skeleton } from '../ui/States'
import { StatusPill, type PillTone } from '../ui/StatusPill'
import { Switch } from '../ui/Switch'
import { AUDIT_FIRST, AUDIT_STEP, useAuditView } from './auditView'
import './activity.css'

// Green Means Up: INFO is not "up", so it stays neutral (legacy painted it
// green); WARN = caution, ERROR = red, SECURITY = emphasized neutral.
const LEVEL: Record<AuditLevel, { tone: PillTone; label: MessageKey }> = {
  INFO: { tone: 'idle', label: 'activity.level.info' },
  WARN: { tone: 'caution', label: 'activity.level.warn' },
  ERROR: { tone: 'bad', label: 'activity.level.error' },
  SECURITY: { tone: 'on', label: 'activity.level.security' },
}

const SIGN: Record<SignatureState, { label: MessageKey; tone: string }> = {
  verified: { label: 'activity.sign.verified', tone: 'ok' },
  tampered: { label: 'activity.sign.tampered', tone: 'bad' },
  unsigned: { label: 'activity.sign.unsigned', tone: 'none' },
}

function LevelPill({ level }: { level: string }) {
  const { t } = usePreferences()
  const spec = LEVEL[level.toUpperCase() as AuditLevel]
  return spec ? <StatusPill tone={spec.tone}>{t(spec.label)}</StatusPill> : <StatusPill tone="unknown">{level}</StatusPill>
}

function Signature({ ev }: { ev: AuditEvent }) {
  const { t } = usePreferences()
  const state = signatureState(ev)
  const icon = state === 'verified' ? <CheckIcon /> : state === 'tampered' ? <AlertIcon /> : null
  return (
    <span className="k-sig" data-state={SIGN[state].tone} title={state === 'tampered' ? t('activity.sign.tampered.why') : undefined}>
      {icon}
      {t(SIGN[state].label)}
    </span>
  )
}

export function AuditPanel({ settings, settingsError }: { settings: Settings | undefined; settingsError: unknown }) {
  const { t, lang } = usePreferences()
  const [profiles, setProfiles] = useState<string[]>([])
  const [level, setLevel] = useState<AuditLevel | ''>('')
  const [range, setRange] = useState({ from: '', to: '' })
  const { from, to } = range
  const known = useProfilesQuery().data
  const [live, setLive] = useState(true)
  const configured = settings ? settings.audit_log_path.trim() !== '' : undefined
  // Without the settings (they failed to load) the log is still worth showing.
  const showLog = configured === true || (settingsError != null && !settings)

  const query = useAuditLog(
    { profiles, level, from, to },
    live && configured !== false,
    configured !== false,
  )
  const rows = useMemo(() => query.data?.pages.flatMap((p) => p.events), [query.data])
  // A new filter starts a new snapshot once its own answer arrives (the
  // previous filter's rows stay on screen meanwhile: placeholder data).
  const filterKey = JSON.stringify([profiles, level, from, to])
  const view = useAuditView(rows, query.isPlaceholderData ? `${filterKey}:wait` : filterKey)
  const [count, setCount] = useState(AUDIT_FIRST)
  const [countFor, setCountFor] = useState(filterKey)
  if (countFor !== filterKey) {
    // A new filter starts again from the first entries.
    setCountFor(filterKey)
    setCount(AUDIT_FIRST)
  }
  const listRef = useRef<HTMLDivElement>(null)
  const visible = view.shown.slice(0, count)
  const left = view.shown.length - visible.length
  const canMore = left > 0 || query.hasNextPage
  const focusList = () => listRef.current?.focus({ preventScroll: true })
  const showFresh = () => {
    // New entries go on top; what was on screen stays on screen below them.
    setCount((n) => n + view.fresh.length)
    view.merge()
    focusList()
  }
  const showMore = () => {
    setCount((n) => n + AUDIT_STEP)
    // The loaded entries run out: ask the daemon for the next older page.
    if (left < AUDIT_STEP && query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage()
    if (left <= AUDIT_STEP && !query.hasNextPage) focusList()
  }
  const refresh = () => void query.refetch().then(() => view.reset())
  // Configured profiles plus any name the log has mentioned on this visit (a
  // profile renamed or deleted since): kept once seen, so filtering by one
  // never hides the others from the list.
  const [seen, setSeen] = useState<readonly string[]>([])
  const fresh = (rows ?? []).map((e) => e.profile).filter((p) => p && !seen.includes(p))
  if (fresh.length > 0) setSeen([...new Set([...seen, ...fresh])])
  const names = useMemo(
    () => [...new Set([...(known ?? []).map((p) => p.name), ...seen, ...profiles].filter(Boolean))].sort(),
    [known, seen, profiles],
  )
  const filtered = profiles.length > 0 || level !== '' || from !== '' || to !== ''

  const fmt = useMemo(() => new Intl.DateTimeFormat(lang, { dateStyle: 'short', timeStyle: 'medium' }), [lang])
  const columns = useMemo<Column<AuditEvent>[]>(
    () => [
      {
        id: 'time',
        header: t('activity.audit.col.time'),
        cell: (ev) => (
          <time className="mono k-audit__time" dateTime={ev.timestamp}>
            {fmt.format(new Date(ev.timestamp))}
          </time>
        ),
      },
      { id: 'level', header: t('activity.audit.col.level'), cell: (ev) => <LevelPill level={ev.level} /> },
      { id: 'action', header: t('activity.audit.col.action'), cell: (ev) => <span className="mono">{ev.action}</span> },
      { id: 'profile', header: t('activity.audit.col.profile'), cell: (ev) => (ev.profile ? <span className="mono">{ev.profile}</span> : '—') },
      { id: 'message', header: t('activity.audit.col.message'), cell: (ev) => <span className="k-audit__msg">{ev.message}</span> },
      { id: 'sig', header: t('activity.audit.col.signature'), cell: (ev) => <Signature ev={ev} /> },
    ],
    [t, fmt],
  )

  const clear = () => {
    setProfiles([])
    setLevel('')
    setRange({ from: '', to: '' })
  }
  const exportRows = () => {
    if (!rows) return
    const stamp = new Date().toISOString().slice(0, 10)
    const file = `kongtrol-audit-${stamp}.jsonl`
    downloadText(file, auditToJSONL(rows))
    toasts.show({ tone: 'ok', message: t('activity.audit.exported', { file, count: rows.length }) })
  }
  const tampered = useMemo(() => (rows ?? []).filter((ev) => signatureState(ev) === 'tampered').length, [rows])

  let body
  if (configured === false) {
    body = <EmptyState icon={<InfoIcon />} title={t('activity.audit.off.title')} body={t('activity.audit.off.body')} action={<a className="k-set__link" href="#log-settings">{t('activity.config.title')}</a>} />
  } else if (query.error && !rows) {
    body = <ErrorState error={query.error} onRetry={() => void query.refetch()} />
  } else if (!rows) {
    body = <Skeleton lines={6} label={t('activity.audit.loading')} />
  } else {
    body = (
      <>
        {tampered > 0 && (
          <Banner tone="bad" title={t('activity.sign.tamperedCount', { n: tampered })}>
            <p>{t('activity.sign.tampered.why')}</p>
          </Banner>
        )}
        <p className="k-act__count" aria-live="polite">
          {query.hasNextPage
            ? t('activity.audit.countMore', { shown: visible.length })
            : left > 0
              ? t('activity.audit.countOf', { shown: visible.length, total: view.shown.length })
              : t('activity.audit.count', { shown: visible.length })}
        </p>
        <div role="status" className="k-audit__fresh">
          {view.fresh.length > 0 && (
            <>
              <span>{t(view.fresh.length === 1 ? 'activity.audit.fresh.one' : 'activity.audit.fresh', { n: view.fresh.length })}</span>
              <Button variant="text" size="sm" onClick={showFresh}>
                {t('activity.audit.fresh.show')}
              </Button>
            </>
          )}
        </div>
        <div ref={listRef} tabIndex={-1} className="k-audit__list" aria-label={t('activity.audit.caption')}>
        <DataList
          caption={t('activity.audit.caption')}
          columns={columns}
          rows={visible}
          rowKey={(ev) => ev.id || `${ev.timestamp}-${ev.action}-${ev.message}`}
          empty={
            <EmptyState
              icon={<InfoIcon />}
              title={t('activity.audit.empty.title')}
              body={filtered ? t('activity.audit.empty.body') : t('activity.audit.emptyAll.body')}
              action={
                filtered ? (
                  <Button variant="outlined" size="sm" onClick={clear}>
                    {t('activity.audit.clear')}
                  </Button>
                ) : undefined
              }
            />
          }
        />
        </div>
        {query.error && <ErrorState error={query.error} onRetry={() => void query.refetch()} />}
        {canMore && (
          <div className="k-act__more">
            <Button variant="outlined" loading={query.isFetchingNextPage} onClick={showMore}>
              {t('activity.audit.more', { n: query.hasNextPage ? AUDIT_STEP : Math.min(AUDIT_STEP, left) })}
            </Button>
          </div>
        )}
      </>
    )
  }

  return (
    <section className="k-act" aria-labelledby="audit-title" aria-busy={configured !== false && !rows && !query.error}>
      <header className="k-act__head">
        <div>
          <h2 id="audit-title" className="k-act__title">
            {t('activity.audit.title')}
          </h2>
          <p className="k-act__what">{t('activity.audit.what')}</p>
        </div>
        {showLog && (
          <div className="k-act__tools">
            <IconButton label={t('activity.audit.refresh')} icon={<RefreshIcon />} onClick={refresh} />
            <Button variant="outlined" onClick={exportRows} disabled={!rows?.length}>
              {t('activity.audit.export')}
            </Button>
          </div>
        )}
      </header>

      {settingsError != null && !settings && <ErrorState error={settingsError} />}

      {configured && settings && (
        <Banner
          tone={settings.audit_log_sign ? 'info' : 'caution'}
          title={settings.audit_log_sign ? t('activity.sign.state.on.title') : t('activity.sign.state.off.title')}
        >
          <p>{settings.audit_log_sign ? t('activity.sign.state.on.body') : t('activity.sign.state.off.body')}</p>
        </Banner>
      )}

      {showLog && (
        <div className="k-audit__filters" role="group" aria-label={t('activity.audit.filters')}>
          <MultiSelectField
            label={t('activity.audit.profile')}
            hint={t('activity.audit.profile.hint')}
            mono
            value={profiles}
            onChange={setProfiles}
            placeholder={t('activity.audit.profile.any')}
            searchLabel={t('activity.audit.profile.search')}
            chipsLabel={t('activity.audit.profile.chosen')}
            options={names.map((p) => ({ value: p, label: p }))}
          />
          <SelectField<AuditLevel | ''>
            label={t('activity.audit.level')}
            value={level}
            onChange={setLevel}
            options={[{ value: '', label: t('activity.audit.level.all') }, ...AUDIT_LEVELS.map((l) => ({ value: l, label: t(LEVEL[l].label) }))]}
          />
          <DateRangeField label={t('activity.audit.dates')} hint={t('activity.audit.range.note')} value={range} onChange={setRange} />
          <div className="k-audit__live">
            <Switch
              checked={live}
              onChange={setLive}
              label={t('activity.audit.live')}
              description={t('activity.audit.live.desc')}
              onText={t('activity.audit.live.on')}
              offText={t('activity.audit.live.off')}
            />
          </div>
        </div>
      )}

      {body}
    </section>
  )
}
