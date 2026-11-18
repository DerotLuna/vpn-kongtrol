// Kongtrol UI — Actividad: signed audit log viewer · v1.0.0 · 2026-11-18
//
// GET /api/v1/audit (same reader as `kongtrol logs`). Fixes of the classic
// page (docs/dashboard-parity.md §3.5): level and date apply at once (no
// "Aplicar"), level names are translated, "Mostrar más" pages past 300, an
// unconfigured log is explained (the API answers 200 [] for it) and any
// other error says what happened instead of "not configured". Signing is
// shown per entry with a word + icon; a signed entry is "sin comprobar"
// until the daemon reports verification (see data/activity.ts).
import { useMemo, useState, type FormEvent } from 'react'
import { usePreferences } from '../../app/preferences-context'
import {
  AUDIT_LEVELS,
  AUDIT_PAGE,
  AUDIT_RANGES,
  auditToJSONL,
  downloadText,
  inRange,
  signatureState,
  useAuditLog,
  type AuditEvent,
  type AuditLevel,
  type AuditRange,
  type SignatureState,
} from '../../data/activity'
import type { Settings } from '../../data/settings'
import { toasts } from '../../data/toast'
import type { MessageKey } from '../../i18n/messages'
import { Button, IconButton } from '../ui/Button'
import { DataList, type Column } from '../ui/DataList'
import { SelectField, TextField } from '../ui/Field'
import { AlertIcon, CheckIcon, InfoIcon, RefreshIcon, ShieldIcon } from '../ui/icons'
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
  signed: { label: 'activity.sign.signed', tone: 'signed' },
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
  const icon = state === 'verified' ? <CheckIcon /> : state === 'tampered' ? <AlertIcon /> : state === 'signed' ? <ShieldIcon /> : null
  return (
    <span className="k-sig" data-state={SIGN[state].tone} title={ev.hmac || undefined}>
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
  const [range, setRange] = useState<AuditRange>('all')
  const [limit, setLimit] = useState(AUDIT_PAGE)
  const [live, setLive] = useState(true)
  const configured = settings ? settings.audit_log_path.trim() !== '' : undefined
  // Without the settings (they failed to load) the log is still worth showing.
  const showLog = configured === true || (settingsError != null && !settings)

  const query = useAuditLog({ profile, level, limit }, live && configured !== false)
  const all = query.data
  const rows = useMemo(() => (all ?? []).filter((ev) => inRange(ev, range)), [all, range])
  const profiles = useMemo(() => [...new Set((all ?? []).map((e) => e.profile).filter(Boolean))].sort(), [all])
  const filtered = profile !== '' || level !== '' || range !== 'all'

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
    setLimit(AUDIT_PAGE)
  }
  const clear = () => {
    setProfileInput('')
    setProfile('')
    setLevel('')
    setRange('all')
    setLimit(AUDIT_PAGE)
  }
  const exportRows = () => {
    const stamp = new Date().toISOString().slice(0, 10)
    const file = `kongtrol-audit-${stamp}.jsonl`
    downloadText(file, auditToJSONL(rows))
    toasts.show({ tone: 'ok', message: t('activity.audit.exported', { file, count: rows.length }) })
  }

  let body
  if (configured === false) {
    body = <EmptyState icon={<InfoIcon />} title={t('activity.audit.off.title')} body={t('activity.audit.off.body')} action={<a className="k-set__link" href="#log-settings">{t('activity.config.title')}</a>} />
  } else if (query.error && !all) {
    body = <ErrorState error={query.error} onRetry={() => void query.refetch()} />
  } else if (!all) {
    body = <Skeleton lines={6} label={t('activity.audit.loading')} />
  } else {
    body = (
      <>
        <p className="k-act__count" aria-live="polite">
          {rows.length === all.length
            ? t('activity.audit.count', { shown: rows.length })
            : t('activity.audit.countOf', { shown: rows.length, total: all.length })}
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
        {all.length >= limit && (
          <div className="k-act__more">
            <Button variant="outlined" loading={query.isFetching} onClick={() => setLimit((n) => n + AUDIT_PAGE)}>
              {t('activity.audit.more')}
            </Button>
          </div>
        )}
      </>
    )
  }

  return (
    <section className="k-act" aria-labelledby="audit-title" aria-busy={configured !== false && !all && !query.error}>
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
            <Button variant="outlined" onClick={exportRows} disabled={rows.length === 0}>
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
                if (profileInput.trim() !== profile) {
                  setProfile(profileInput.trim())
                  setLimit(AUDIT_PAGE)
                }
              }}
            />
            <datalist id="audit-profiles">
              {profiles.map((p) => (
                <option key={p} value={p} />
              ))}
            </datalist>
          </form>
          <SelectField
            label={t('activity.audit.level')}
            value={level}
            onChange={(e) => {
              setLevel(e.target.value as AuditLevel | '')
              setLimit(AUDIT_PAGE)
            }}
          >
            <option value="">{t('activity.audit.level.all')}</option>
            {AUDIT_LEVELS.map((l) => (
              <option key={l} value={l}>
                {t(LEVEL[l].label)}
              </option>
            ))}
          </SelectField>
          <SelectField
            label={t('activity.audit.range')}
            hint={t('activity.audit.range.note')}
            value={range}
            onChange={(e) => setRange(e.target.value as AuditRange)}
          >
            {AUDIT_RANGES.map((r) => (
              <option key={r} value={r}>
                {t(`activity.audit.range.${r}` as MessageKey)}
              </option>
            ))}
          </SelectField>
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
