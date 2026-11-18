// Kongtrol UI — protection summary (Inicio) · v1.0.0 · 2026-11-18
//
// Same states as `kongtrol status` and the classic shell badges
// (shellSecurityBadges), as words + pill shapes: kill switch, DNS
// protection and the opt-in public IP check, with the public IP, the last
// check time and one "Revisar" (only while the lookup is on; nothing leaves
// the machine otherwise). Changing any of them lives in /security.
import { Link } from 'react-router'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { usePublicIPCheck, type SecuritySummary } from '../../data/home'
import { useSecurityQuery } from '../../data/resources'
import { securityView, type SecurityLine } from './model'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { ErrorState, Skeleton } from '../ui/States'
import { StatusPill } from '../ui/StatusPill'
import { formatClock } from './format'

function Row({ title, line }: { title: string; line: SecurityLine }) {
  const { t } = usePreferences()
  return (
    <div className="home-sec__row">
      <dt>{title}</dt>
      <dd>
        <StatusPill tone={line.tone}>{t(line.label)}</StatusPill>
        {line.detail && <p className="home-sec__detail">{t(line.detail.key, line.detail.vars)}</p>}
      </dd>
    </div>
  )
}

export function SecurityCard() {
  const { t, lang } = usePreferences()
  const query = useSecurityQuery()
  const check = usePublicIPCheck()
  const st = query.data as SecuritySummary | undefined
  const view = st ? securityView(st) : null
  const on = !!st?.public_ip_check_enabled
  const checkError = check.isError ? describeError(check.error, t) : null

  return (
    <Card
      className="home-sec"
      title={t('home.sec.title')}
      aside={<Link to="/security">{t('home.sec.manage')}</Link>}
      busy={query.isPending}
    >
      {query.isPending && <Skeleton lines={4} label={t('home.sec.loading')} />}
      {query.isError && !st && <ErrorState error={query.error} onRetry={() => void query.refetch()} />}
      {st && view && (
        <>
          <dl className="home-sec__list">
            <Row title={t('home.sec.killSwitch')} line={view.killSwitch} />
            <Row title={t('home.sec.dnsGuard')} line={view.dnsGuard} />
            <Row title={t('home.sec.leak')} line={view.leak} />
          </dl>
          <dl className="home-facts">
            <div>
              <dt>{t('home.sec.publicIp')}</dt>
              <dd className="mono">{(on && st.leak_check?.public_ip) || '—'}</dd>
            </div>
            <div>
              <dt>{t('home.sec.lastChecked')}</dt>
              <dd className="mono">{(on && formatClock(st.leak_check?.checked_at, lang)) || '—'}</dd>
            </div>
          </dl>
          {on && (
            <Button variant="outlined" size="sm" className="home-sec__check" loading={check.isPending} onClick={() => check.mutate()}>
              {check.isPending ? t('home.sec.checking') : t('home.sec.check')}
            </Button>
          )}
          <div aria-live="polite">
            {checkError && !check.isPending && (
              <p className="notice bad">
                <strong>{checkError.message}</strong>
                <span>{checkError.fix}</span>
              </p>
            )}
          </div>
        </>
      )}
    </Card>
  )
}
