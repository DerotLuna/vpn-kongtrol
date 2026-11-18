// Kongtrol UI — Actividad: signed audit log viewer · v1.1.0 · 2026-11-18
//
// GET /api/v1/audit (same reader as `kongtrol logs`, ADR-0042). Level,
// profile and dates filter on the daemon (from / to = local days, across
// every retained day); "Cargar más" asks for the next older page with the
// `before` cursor while X-Kongtrol-Audit-More says there is one. Level
// names are translated; an unconfigured log is explained (the API answers
// 200 [] for it) and any other error says what happened. Each entry shows
// its signature in a word + icon: "Firma válida", "Firma no válida"
// (danger: edited, or signed with another key) or "Sin firma".
import { useMemo, useState, type FormEvent } from 'react'
import { usePreferences } from '../../app/preferences-context'
import {
  AUDIT_LEVELS,
  auditToJSONL,
  badRange,
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
import { SelectField, TextField } from '../ui/Field'
import { AlertIcon, CheckIcon, InfoIcon, RefreshIcon } from '../ui/icons'
import { Banner, EmptyState, ErrorState, Skeleton } from '../ui/States'
import { StatusPill, type PillTone } from '../ui/StatusPill'
import { Switch } from '../ui/Switch'
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
  const [profileInput, setProfileInput] = useState('')
  const [profile, setProfile] = useState('')
  const [level, setLevel] = useState<AuditLevel | ''>('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [live, setLive] = useState(true)
  const configured = settings ? settings.audit_log_path.trim() !== '' : undefined
  // Without the settings (they failed to load) the log is still worth showing.
  const showLog = configured === true || (settingsError != null && !settings)
  const rangeBad = badRange(from, to)

  const query = useAuditLog(
    { profile, level, from: rangeBad ? '' : from, to: rangeBad ? '' : to },
    live && configured !== false,
    configured !== false,
  )
  const rows = useMemo(() => query.data?.pages.flatMap((p) => p.events), [query.data])
  const profiles = useMemo(() => [...new Set((rows ?? []).map((e) => e.profile).filter(Boolean))].sort(), [rows])
  const filtered = profile !== '' || level !== '' || from !== '' || to !== ''

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

  const applyProfile = (e: FormEvent) => {
    e.preventDefault()
    setProfile(profileInput.trim())
  }
  const clear = () => {
    setProfileInput('')
    setProfile('')
    setLevel('')
    setFrom('')
    setTo('')
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
          {t(query.hasNextPage ? 'activity.audit.countMore' : 'activity.audit.count', { shown: rows.length })}
        </p>
        <DataList
          caption={t('activity.audit.caption')}
          columns={columns}
          rows={rows}
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
        {query.error && <ErrorState error={query.error} onRetry={() => void query.refetch()} />}
        {query.hasNextPage && (
          <div className="k-act__more">
            <Button variant="outlined" loading={query.isFetchingNextPage} onClick={() => void query.fetchNextPage()}>
              {t('activity.audit.more')}
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
            <IconButton label={t('activity.audit.refresh')} icon={<RefreshIcon />} onClick={() => void query.refetch()} />
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
          <form onSubmit={applyProfile} className="k-audit__profile">
            <TextField
              label={t('activity.audit.profile')}
              hint={t('activity.audit.profile.hint')}
              mono
              list="audit-profiles"
              placeholder={t('activity.audit.profile.any')}
              autoComplete="off"
              spellCheck={false}
              value={profileInput}
              onChange={(e) => setProfileInput(e.target.value)}
              onBlur={() => {
                if (profileInput.trim() !== profile) setProfile(profileInput.trim())
              }}
            />
            <datalist id="audit-profiles">
              {profiles.map((p) => (
                <option key={p} value={p} />
              ))}
            </datalist>
          </form>
          <SelectField label={t('activity.audit.level')} value={level} onChange={(e) => setLevel(e.target.value as AuditLevel | '')}>
            <option value="">{t('activity.audit.level.all')}</option>
            {AUDIT_LEVELS.map((l) => (
              <option key={l} value={l}>
                {t(LEVEL[l].label)}
              </option>
            ))}
          </SelectField>
          <TextField
            type="date"
            label={t('activity.audit.from')}
            value={from}
            max={to || undefined}
            onChange={(e) => setFrom(e.target.value)}
            error={rangeBad ? { message: t('activity.audit.range.invalid'), fix: t('activity.audit.range.invalid.fix') } : null}
          />
          <TextField
            type="date"
            label={t('activity.audit.to')}
            hint={t('activity.audit.range.note')}
            value={to}
            min={from || undefined}
            onChange={(e) => setTo(e.target.value)}
          />
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
