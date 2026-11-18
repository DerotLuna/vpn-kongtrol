// Kongtrol UI — Seguridad: kill switch · v1.1.0 · 2026-11-18
//
// What it does, whether it acts now (off / ready / protecting / blocking,
// AGENTS.md "Policy routing, kill switch"), which tunnels and rules it
// covers, the LAN exception, and the helper that applies it. Turning it
// off asks first; turning it on does not (it only adds protection).
import { useState } from 'react'
import { useHelper } from '../../api/hooks'
import { usePreferences } from '../../app/preferences-context'
import { helperUiState } from '../../data/resources'
import {
  lanAllowed,
  lanPatch,
  useKillSwitchToggle,
  useSecuritySettings,
  useSettingsToggle,
  type SecurityState,
} from '../../data/security'
import { Banner, Skeleton } from '../ui/States'
import { Card } from '../ui/Card'
import { HelperInstallButton } from '../ui/HelperInstall'
import { Switch } from '../ui/Switch'
import { BlockedList } from './BlockedList'
import { killSwitchView } from './model'
import { ConfirmDialog, StateLine, TechDetails } from './parts'

function Names({ label, names }: { label: string; names: string[] | undefined }) {
  if (!names?.length) return null
  return (
    <div className="sec-names">
      <span className="sec-names__label">{label}</span>
      <ul className="sec-chips">
        {names.map((n) => (
          <li key={n} className="sec-chip mono">
            {n}
          </li>
        ))}
      </ul>
    </div>
  )
}

const HELPER_TEXT = {
  unknown: 'security.helper.unknown',
  ready: 'security.helper.ready',
  not_needed: 'security.helper.notNeeded',
  missing: 'security.helper.missing',
  outdated: 'security.helper.outdated',
} as const

export function KillSwitchCard({ status }: { status: SecurityState }) {
  const { t } = usePreferences()
  const toggle = useKillSwitchToggle()
  const settings = useSecuritySettings()
  const lan = useSettingsToggle()
  const helper = useHelper()
  const [asking, setAsking] = useState(false)
  const view = killSwitchView(status)
  const d = status.kill_switch_status
  const helperState = helperUiState(helper.status ?? undefined)
  const helperNeeded = helperState === 'missing' || helperState === 'outdated'
  const ksError = status.kill_switch_error

  return (
    <Card title={t('security.ks.title')} className="sec-card sec-card--wide">
      <span id="kill-switch" className="sec-anchor" />
      <Switch
        checked={status.kill_switch_enabled}
        onChange={(next) => (next ? toggle.mutate(true) : setAsking(true))}
        label={t('security.ks.label')}
        description={t('security.ks.desc')}
        onText={t('security.on')}
        offText={t('security.off')}
        pending={toggle.isPending}
        disabled={toggle.isPending}
      />
      <StateLine view={view}>
        {view.pill === 'security.ks.pill.active' && <p className="sec-fix">{t('security.ks.state.active.fix')}</p>}
      </StateLine>

      {/* The daemon's own words for the last failure (message + what to do, ADR-0042). */}
      {ksError && (
        <Banner
          tone="bad"
          title={ksError.message}
          actions={helperNeeded ? <HelperInstallButton helper={helper} outdated={helperState === 'outdated'} /> : undefined}
        >
          {ksError.hint && <p>{ksError.hint}</p>}
        </Banner>
      )}

      {status.kill_switch_enabled && (
        <div className="sec-cover">
          <Names label={t('security.ks.protected')} names={d?.protected} />
          <Names label={t('security.ks.blocking')} names={d?.blocking} />
          <BlockedList enabled={status.kill_switch_enabled} />
        </div>
      )}

      <div className="sec-sub">
        {settings.data ? (
          <>
            <Switch
              checked={lanAllowed(settings.data)}
              onChange={(next) => settings.data && lan.mutate(lanPatch(next, settings.data))}
              label={t('security.ks.lan.label')}
              description={t('security.ks.lan.desc')}
              onText={t('security.on')}
              offText={t('security.off')}
              pending={lan.isPending}
              disabled={lan.isPending}
            />
            {settings.data.kill_switch_mode === 'loose' && <p className="sec-meta">{t('security.ks.lan.legacy')}</p>}
          </>
        ) : (
          <Skeleton lines={2} label={t('security.loading')} />
        )}
      </div>

      <TechDetails>
        <dl className="sec-dl">
          <dt>{t('security.ks.cli.label')}</dt>
          <dd className="mono">{t(view.cli, { n: d?.protected?.length ?? 0, names: (d?.blocking ?? []).join(', ') })}</dd>
          <dt>kongtrol.yaml</dt>
          <dd className="mono">
            security.kill_switch.enabled: {String(status.kill_switch_enabled)}
            <br />
            security.kill_switch.allow_lan: {String(settings.data?.kill_switch_allow_lan ?? false)}
            {settings.data?.kill_switch_mode ? (
              <>
                <br />
                security.kill_switch.mode: {settings.data.kill_switch_mode}
              </>
            ) : null}
          </dd>
          <dt>CLI</dt>
          <dd className="mono">kongtrol config killswitch on|off|status</dd>
          <dt>{t('security.helper.label')}</dt>
          <dd>{t(HELPER_TEXT[helperState])}</dd>
          {(ksError?.detail || d?.error) && (
            <>
              <dt>error</dt>
              <dd className="mono">
                {ksError?.error_key || d?.error_key ? `${ksError?.error_key || d?.error_key} · ` : ''}
                {ksError?.detail || d?.error}
              </dd>
            </>
          )}
        </dl>
      </TechDetails>

      <ConfirmDialog
        open={asking}
        title={t('security.ks.confirm.title')}
        action={t('security.ks.confirm.action')}
        onCancel={() => setAsking(false)}
        onConfirm={() => {
          setAsking(false)
          toggle.mutate(false)
        }}
      >
        <p>{t('security.ks.confirm.body')}</p>
      </ConfirmDialog>
    </Card>
  )
}
