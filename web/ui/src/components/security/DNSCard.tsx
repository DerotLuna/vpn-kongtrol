// Kongtrol UI — Seguridad: DNS protection + split DNS · v1.1.0 · 2026-11-18
//
// DNS guard switch (optimistic; turning it off asks first, it lowers
// privacy), its state in plain words, the fallback DNS, and split DNS
// (hosts-file injection, internal/monitor/splitdns.go; on/off applies at
// daemon start: cmd/kongtrol/daemon_cmd.go).
import { useState } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { useDNSGuardToggle, useSecuritySettings, useSettingsToggle, type SecurityState } from '../../data/security'
import { Card } from '../ui/Card'
import { Skeleton } from '../ui/States'
import { Switch } from '../ui/Switch'
import { dnsView, isDuration, isIP } from './model'
import { ConfirmDialog, SettingForm, StateLine, TechDetails } from './parts'

/**
 * The DNS servers the guard applies now and which VPN each comes from
 * (security/status dns_guard_servers / _interface / _profiles, ADR-0042).
 * Nothing while the guard is idle.
 */
function DNSInUse({ status }: { status: SecurityState }) {
  const { t } = usePreferences()
  const servers = status.dns_guard_servers ?? []
  if (!status.dns_guard_enabled || servers.length === 0) return null
  const byProfile = Object.entries(status.dns_guard_profiles ?? {})
    .filter(([, list]) => list.length > 0)
    .sort(([a], [b]) => a.localeCompare(b))
  return (
    <div className="sec-dns" aria-label={t('security.dns.inUse')} role="group">
      <p className="sec-names__label">
        {status.dns_guard_interface
          ? t('security.dns.inUse.iface', { iface: status.dns_guard_interface })
          : t('security.dns.inUse')}
      </p>
      {byProfile.length > 0 ? (
        <dl className="sec-dl">
          {byProfile.map(([profile, list]) => (
            <div key={profile} className="sec-dl__pair">
              <dt className="mono">{profile}</dt>
              <dd className="mono">{list.join(', ')}</dd>
            </div>
          ))}
        </dl>
      ) : (
        <ul className="sec-chips">
          {servers.map((s) => (
            <li key={s} className="sec-chip mono">
              {s}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export function DNSCard({ status }: { status: SecurityState }) {
  const { t } = usePreferences()
  const toggle = useDNSGuardToggle()
  const settings = useSecuritySettings()
  const split = useSettingsToggle()
  const [asking, setAsking] = useState(false)
  const view = dnsView(status)
  const s = settings.data

  return (
    <Card title={t('security.dns.title')} className="sec-card">
      <span id="dns-guard" className="sec-anchor" />
      <Switch
        checked={status.dns_guard_enabled}
        onChange={(next) => (next ? toggle.mutate(true) : setAsking(true))}
        label={t('security.dns.label')}
        description={t('security.dns.desc')}
        onText={t('security.on')}
        offText={t('security.off')}
        pending={toggle.isPending}
        disabled={toggle.isPending}
      />
      <StateLine view={view} />
      <DNSInUse status={status} />

      <div className="sec-sub">
        {s ? (
          <>
            <SettingForm
              key={`fb-${s.dns_guard_fallback_dns}`}
              label={t('security.dns.fallback.label')}
              hint={t('security.dns.fallback.hint')}
              initial={s.dns_guard_fallback_dns}
              placeholder="1.1.1.1"
              validate={(v) =>
                v.trim() === '' || isIP(v)
                  ? null
                  : { message: t('security.dns.fallback.invalid'), fix: t('security.dns.fallback.invalid.fix') }
              }
              toPatch={(v) => ({ dns_guard_fallback_dns: v.trim() })}
            />
            <Switch
              checked={s.split_dns_enabled}
              onChange={(next) => split.mutate({ split_dns_enabled: next })}
              label={t('security.split.label')}
              description={
                <>
                  {t('security.split.desc')} {t('security.split.restart')}
                </>
              }
              onText={t('security.on')}
              offText={t('security.off')}
              pending={split.isPending}
              disabled={split.isPending}
            />
            {s.split_dns_enabled && (
              <SettingForm
                key={`si-${s.split_dns_interval}`}
                label={t('security.split.interval.label')}
                hint={t('security.split.interval.hint')}
                initial={s.split_dns_interval}
                placeholder="60s"
                validate={(v) =>
                  v.trim() === '' || isDuration(v)
                    ? null
                    : { message: t('security.split.interval.invalid'), fix: t('security.split.interval.invalid.fix') }
                }
                toPatch={(v) => ({ split_dns_interval: v.trim() })}
              />
            )}
          </>
        ) : (
          <Skeleton lines={3} label={t('security.loading')} />
        )}
      </div>

      <TechDetails>
        <dl className="sec-dl">
          <dt>{t('security.ks.cli.label')}</dt>
          <dd className="mono">{t(view.cli)}</dd>
          <dt>kongtrol.yaml</dt>
          <dd className="mono">
            security.dns_guard.enabled: {String(status.dns_guard_enabled)}
            <br />
            security.dns_guard.fallback_dns: {s?.dns_guard_fallback_dns || '""'}
            <br />
            monitor.split_dns.enabled: {String(s?.split_dns_enabled ?? false)}
          </dd>
          <dt>CLI</dt>
          <dd className="mono">kongtrol config dns-guard on|off|status</dd>
        </dl>
      </TechDetails>

      <ConfirmDialog
        open={asking}
        title={t('security.dns.confirm.title')}
        action={t('security.dns.confirm.action')}
        onCancel={() => setAsking(false)}
        onConfirm={() => {
          setAsking(false)
          toggle.mutate(false)
        }}
      >
        <p>{t('security.dns.confirm.body')}</p>
      </ConfirmDialog>
    </Card>
  )
}
