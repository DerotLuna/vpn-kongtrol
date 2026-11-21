import { usePreferences } from '../../app/preferences-context'
import { useProfileDefaultEgress } from '../../data/profiles'
import { Switch } from '../ui/Switch'

export function DefaultEgress({ name, busy, dirty }: { name: string; busy: boolean; dirty: boolean }) {
  const { t } = usePreferences()
  const { query, change } = useProfileDefaultEgress(name)
  const status = query.data
  if (!status || (status.reason === 'unsupported_platform' && !status.enabled)) return null

  // feature:default_egress
  const description = busy
    ? t('profiles.egress.busy')
    : dirty
      ? t('profiles.egress.dirty')
      : !status.compatible
        ? t(`profiles.egress.${status.reason ?? 'invalid_config'}`)
        : t('profiles.egress.description')
  return (
    <Switch
      checked={status.enabled}
      onChange={(enabled) => change.mutate(enabled)}
      disabled={busy || dirty || change.isPending || (!status.compatible && !status.enabled)}
      pending={change.isPending}
      label={t('profiles.egress.label')}
      description={description}
      onText={t('profiles.egress.on')}
      offText={t('profiles.egress.off')}
    />
  )
}
