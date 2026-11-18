// Kongtrol UI — Seguridad: opt-in public IP check · v1.0.0 · 2026-11-18
//
// AGENTS.md "Local only" + "Public IP lookup (opt-in)": off by default; the
// notice "Usa un servicio externo (<host>) que ve tu IP pública" is shown
// and confirmed BEFORE turning it on (the classic Settings switch skipped
// it: dashboard-parity §5.1 #6). One switch only: the Settings duplicate
// (leak_detection_enabled) is merged here. Reading the state never makes a
// lookup; only "Revisar ahora" (and the daemon after connect/disconnect).
import { useState } from 'react'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import {
  invalidEndpoints,
  parseEndpoints,
  usePublicIPCheck,
  usePublicIPToggle,
  useSecuritySettings,
  type SecurityState,
} from '../../data/security'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { RefreshIcon } from '../ui/icons'
import { Skeleton } from '../ui/States'
import { StatusPill } from '../ui/StatusPill'
import { Switch } from '../ui/Switch'
import { LEAK_TONE, leakPhase } from './model'
import { ConfirmDialog, SettingForm, TechDetails } from './parts'

const DEFAULT_HOST = 'api.ipify.org'

export function PublicIPCheckCard({ status }: { status: SecurityState }) {
  const { t, lang } = usePreferences()
  const toggle = usePublicIPToggle()
  const check = usePublicIPCheck()
  const settings = useSecuritySettings()
  const [asking, setAsking] = useState(false)
  const on = status.public_ip_check_enabled
  const host = status.public_ip_check_host || DEFAULT_HOST
  const phase = leakPhase(status)
  const lk = status.leak_check
  const failure = check.error ? describeError(check.error, t) : null
  const checkedAt = lk?.checked_at ? new Date(lk.checked_at) : null
  const validDate = checkedAt && !Number.isNaN(checkedAt.getTime()) && checkedAt.getFullYear() > 1

  return (
    <Card
      title={t('security.ip.title')}
      className="sec-card"
      aside={on ? <StatusPill tone={LEAK_TONE[phase]}>{t(`security.ip.pill.${phase}`)}</StatusPill> : undefined}
    >
      <span id="public-ip" className="sec-anchor" />
      <Switch
        checked={on}
        onChange={(next) => (next ? setAsking(true) : toggle.mutate(false))}
        label={t('security.ip.label')}
        description={t('security.ip.desc')}
        onText={t('security.on')}
        offText={t('security.off')}
        pending={toggle.isPending}
        disabled={toggle.isPending}
      />
      <p className="sec-notice">{t('publicip.notice', { host })}</p>

      {on ? (
        <div className="sec-ip" aria-live="polite">
          <dl className="sec-ip__grid">
            <div>
              <dt>{t('security.ip.value')}</dt>
              <dd className="sec-ip__value mono">{lk?.public_ip || '—'}</dd>
            </div>
            <div>
              <dt>{t('security.ip.checkedAt')}</dt>
              <dd className="mono">{validDate ? checkedAt.toLocaleTimeString(lang) : '—'}</dd>
            </div>
          </dl>
          {!lk && <p className="sec-meta">{t('security.ip.never')}</p>}
          {phase === 'leak' && (
            <p className="sec-alert">
              <strong>{t('security.ip.leak')}</strong> {t('security.ip.leak.fix')}
            </p>
          )}
          {phase === 'error' && (
            <p className="sec-alert">
              <strong>{t('security.ip.error', { host })}</strong> {t('security.ip.error.fix')}
            </p>
          )}
          {failure && (
            <p className="sec-alert" role="alert">
              <strong>{failure.message}</strong> {failure.fix}
            </p>
          )}
          <div>
            <Button variant="outlined" icon={<RefreshIcon />} loading={check.isPending} onClick={() => check.mutate()}>
              {t('security.ip.check')}
            </Button>
          </div>
          <p className="sec-meta">{t('security.ip.when')}</p>
        </div>
      ) : (
        <p className="sec-meta">{t('publicip.hint')}</p>
      )}

      <div className="sec-sub">
        {settings.data ? (
          <SettingForm
            key={`ep-${(settings.data.public_ip_endpoints ?? []).join('|')}`}
            multiline
            label={t('security.ip.endpoints.label')}
            hint={t('security.ip.endpoints.hint')}
            initial={(settings.data.public_ip_endpoints ?? []).join('\n')}
            placeholder="https://api.ipify.org"
            validate={(v) => {
              const bad = invalidEndpoints(parseEndpoints(v))[0]
              return bad === undefined
                ? null
                : { message: t('security.ip.endpoints.invalid', { line: bad }), fix: t('security.ip.endpoints.invalid.fix') }
            }}
            toPatch={(v) => ({ public_ip_endpoints: parseEndpoints(v) })}
          />
        ) : (
          <Skeleton lines={2} label={t('security.loading')} />
        )}
      </div>

      <TechDetails>
        <dl className="sec-dl">
          <dt>kongtrol.yaml</dt>
          <dd className="mono">security.public_ip_check.enabled: {String(on)}</dd>
          <dt>CLI</dt>
          <dd className="mono">kongtrol config public-ip on|off · kongtrol check</dd>
          {lk?.reason && (
            <>
              <dt>reason</dt>
              <dd className="mono">{lk.reason}</dd>
            </>
          )}
        </dl>
        <p className="sec-meta">{t('security.ip.expected')}</p>
      </TechDetails>

      <ConfirmDialog
        open={asking}
        danger={false}
        title={t('security.ip.consent.title')}
        action={t('security.ip.consent.action')}
        onCancel={() => setAsking(false)}
        onConfirm={() => {
          setAsking(false)
          toggle.mutate(true)
        }}
      >
        <p>
          <strong>{t('publicip.notice', { host })}.</strong>
        </p>
        <p>{t('security.ip.when')}</p>
        <p>{t('security.ip.consent.what')}</p>
      </ConfirmDialog>
    </Card>
  )
}
