// Kongtrol UI — Ajustes: Kongtrol on this computer · v1.0.1 · 2026-11-18
//
// Version, helper (shared install flow, ADR-0038 §5), config file
// (GET /api/v1/setup config_path), the panel's own address (read-only: it is
// changed with `kongtrol config dashboard set-port|set-bind`, never from the
// page it serves — AGENTS.md "machine-local preferences"), restart
// (POST /api/v1/restart) and "Salir y desconectar todo" (POST
// /api/v1/shutdown; until now only the tray offered it).
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { HelperState } from '../../api/hooks'
import { usePreferences } from '../../app/preferences-context'
import { helperUiState, useAppVersion } from '../../data/resources'
import { useRestart, useSetupStatus, useShutdown, type Settings } from '../../data/settings'
import { toasts } from '../../data/toast'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { ErrorState } from '../ui/States'
import { HelperInstallButton, HelperInstallProgress } from '../ui/HelperInstall'
import { Banner } from '../ui/States'
import { StatusPill } from '../ui/StatusPill'
import { SettingRow, SettingsSection } from './SettingsSection'

function HelperRow({ helper }: { helper: HelperState }) {
  const { t } = usePreferences()
  const state = helperUiState(helper.status ?? undefined)
  const pill = {
    unknown: <StatusPill tone="unknown">{t('helper.loading')}</StatusPill>,
    not_needed: <StatusPill tone="ok">{t('helper.pill.notNeeded')}</StatusPill>,
    ready: <StatusPill tone="ok">{t('helper.pill.ready')}</StatusPill>,
    missing: <StatusPill tone="caution">{t('helper.pill.missing')}</StatusPill>,
    outdated: <StatusPill tone="caution">{t('helper.pill.outdated')}</StatusPill>,
  }[state]
  const hint =
    state === 'missing'
      ? t('helper.needsInstall')
      : state === 'outdated'
        ? t('helper.needsUpdate')
        : helper.status?.helper_version && state === 'ready'
          ? t('helper.version', { version: helper.status.helper_version })
          : t('helper.what')
  return (
    <>
      <SettingRow label={t('settings.helper')} hint={hint}>
        {pill}
        {(state === 'missing' || state === 'outdated') && (
          <HelperInstallButton helper={helper} outdated={state === 'outdated'} />
        )}
      </SettingRow>
      <HelperInstallProgress helper={helper} />
      {helper.error && <ErrorState help error={helper.error} onRetry={helper.reload} />}
    </>
  )
}

type Asking = 'restart' | 'quit' | null

export function SystemSection({ settings, helper }: { settings: Settings | undefined; helper: HelperState }) {
  const { t } = usePreferences()
  const client = useQueryClient()
  const version = useAppVersion()
  const setup = useSetupStatus()
  const [asking, setAsking] = useState<Asking>(null)
  const restart = useRestart(() => {
    toasts.show({ tone: 'ok', message: t('settings.restart.done') })
    void client.invalidateQueries()
  })
  const quit = useShutdown()
  const configPath = setup.data?.config_path
  const address = settings ? `${settings.dashboard_bind || '127.0.0.1'}:${settings.dashboard_port}` : null

  return (
    <SettingsSection id="system" title={t('settings.system.title')} what={t('settings.system.what')}>
      <div className="k-set__rows">
        <SettingRow label={t('settings.version')}>
          <span className="mono">{version || t('settings.version.unknown')}</span>
        </SettingRow>
        <HelperRow helper={helper} />
        <SettingRow label={t('settings.config')}>
          {setup.data ? (
            configPath ? (
              <code className="k-set__path">{configPath}</code>
            ) : (
              <span className="muted">{t('settings.config.none')}</span>
            )
          ) : (
            <span className="k-set__placeholder" aria-hidden="true" />
          )}
        </SettingRow>
        <SettingRow
          label={t('settings.panel')}
          hint={
            <>
              {t('settings.panel.hint')} <code>kongtrol config dashboard set-port</code> ·{' '}
              <code>kongtrol config dashboard set-bind</code>
            </>
          }
        >
          {address ? <code className="k-set__path">{address}</code> : <span className="k-set__placeholder" aria-hidden="true" />}
        </SettingRow>
      </div>

      <div aria-live="polite">
        {restart.phase === 'running' && (
          <Banner tone="info" title={t('settings.restart.running')} />
        )}
        {restart.phase === 'timeout' && (
          <Banner tone="bad" title={t('settings.restart.timeout')}>
            <p>{t('settings.restart.timeoutFix')}</p>
          </Banner>
        )}
        {quit.phase === 'done' && <Banner tone="info" title={t('settings.quit.done')} />}
      </div>
      {restart.error && <ErrorState help error={restart.error} />}
      {quit.error && <ErrorState help error={quit.error} />}

      <div className="k-set__actions">
        <div className="k-set__action">
          <Button variant="outlined" onClick={() => setAsking('restart')} loading={restart.phase === 'running'} disabled={quit.phase === 'done'}>
            {t('settings.restart')}
          </Button>
          <p className="k-set__hint">{t('settings.restart.what')}</p>
        </div>
        <div className="k-set__action">
          <Button variant="danger" onClick={() => setAsking('quit')} loading={quit.phase === 'running'} disabled={quit.phase === 'done'}>
            {t('settings.quit')}
          </Button>
          <p className="k-set__hint">{t('settings.quit.what')}</p>
        </div>
      </div>

      <Dialog
        open={asking === 'restart'}
        alert
        onClose={() => setAsking(null)}
        title={t('settings.restart.title')}
        description={t('settings.restart.what')}
        actions={
          <>
            <Button variant="text" onClick={() => setAsking(null)}>
              {t('settings.cancel')}
            </Button>
            <Button
              variant="filled"
              onClick={() => {
                setAsking(null)
                restart.run()
              }}
            >
              {t('settings.restart.confirm')}
            </Button>
          </>
        }
      />
      <Dialog
        open={asking === 'quit'}
        alert
        onClose={() => setAsking(null)}
        title={t('settings.quit.title')}
        description={t('settings.quit.what')}
        actions={
          <>
            <Button variant="text" onClick={() => setAsking(null)}>
              {t('settings.cancel')}
            </Button>
            <Button
              variant="danger"
              onClick={() => {
                setAsking(null)
                quit.run()
              }}
            >
              {t('settings.quit.confirm')}
            </Button>
          </>
        }
      />
    </SettingsSection>
  )
}
