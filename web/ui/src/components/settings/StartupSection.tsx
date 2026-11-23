// Kongtrol UI — Ajustes › Al iniciar · v1.0.0 · 2026-11-23
//
// Two switches that apply at once (ADR-0098), the same as
// `kongtrol config autostart` and `kongtrol config restore-tunnels`.
import { usePreferences } from '../../app/preferences-context'
import { useSaveStartup, useStartupQuery } from '../../data/startup'
import { ErrorState, Skeleton } from '../ui/States'
import { Switch } from '../ui/Switch'
import { SettingsSection } from './SettingsSection'

// feature:restore_at_start
export function StartupSection() {
  const { t } = usePreferences()
  const query = useStartupQuery()
  const save = useSaveStartup()
  const st = query.data
  const pending = save.isPending

  return (
    <SettingsSection id="startup" title={t('settings.startup.title')} what={t('settings.startup.what')} busy={!st && !query.error}>
      {query.error ? (
        <ErrorState error={query.error} onRetry={() => void query.refetch()} />
      ) : !st ? (
        <Skeleton lines={2} label={t('settings.loading')} />
      ) : (
        <div className="k-set__fields">
          <Switch
            checked={st.start_at_login}
            onChange={(on) => save.mutate({ start_at_login: on })}
            disabled={pending || (!st.start_at_login_supported && !st.start_at_login)}
            pending={pending}
            label={t('settings.startup.login')}
            description={st.start_at_login_supported ? t('settings.startup.login.hint') : t('settings.startup.login.noTray')}
            onText={t('settings.startup.on')}
            offText={t('settings.startup.off')}
          />
          <Switch
            checked={st.restore_tunnels}
            onChange={(on) => save.mutate({ restore_tunnels: on })}
            disabled={pending}
            pending={pending}
            label={t('settings.startup.restore')}
            description={t('settings.startup.restore.hint')}
            onText={t('settings.startup.on')}
            offText={t('settings.startup.off')}
          />
        </div>
      )}
    </SettingsSection>
  )
}
