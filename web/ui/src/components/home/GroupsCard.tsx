// Kongtrol UI — groups quick launch (Inicio) · v1.0.0 · 2026-11-18
//
// Connect or disconnect a whole group in one click. The group's answer is
// read, not ignored (legacy bug): which VPNs started, which were already up
// and which the running Kongtrol does not know yet. Editing groups lives in
// /profiles/groups.
import { useMutationState } from '@tanstack/react-query'
import { Link } from 'react-router'
import type { Group } from '../../api/types'
import { usePreferences } from '../../app/preferences-context'
import { homeMutationKeys, useGroupAction, useGroupsList, type GroupAction } from '../../data/home'
import { Button, IconButton } from '../ui/Button'
import { Card } from '../ui/Card'
import { RefreshIcon } from '../ui/icons'
import { EmptyState, ErrorState, Skeleton } from '../ui/States'

export function GroupsCard() {
  const { t } = usePreferences()
  const query = useGroupsList()
  const action = useGroupAction()
  // Which group/action is running, from this card's mutation.
  const running = useMutationState({
    filters: { mutationKey: homeMutationKeys.groupAction, status: 'pending' },
    select: (m) => m.state.variables as { name?: string; action?: GroupAction } | undefined,
  }).filter((v): v is { name: string; action: GroupAction } => !!v?.name && !!v.action)
  const groups: Group[] = query.data ?? []

  return (
    <Card
      className="home-groups"
      title={t('home.groups.title')}
      aside={<IconButton label={t('home.groups.refresh')} icon={<RefreshIcon />} onClick={() => void query.refetch()} />}
      busy={query.isPending}
    >
      <p className="muted home-lead">{t('home.groups.lead')}</p>
      {query.isPending && <Skeleton lines={2} label={t('home.groups.loading')} />}
      {query.isError && !query.data && <ErrorState error={query.error} onRetry={() => void query.refetch()} />}
      {query.data && groups.length === 0 && (
        <EmptyState
          title={t('home.groups.empty.title')}
          body={t('home.groups.empty.body')}
          action={<Link to="/profiles/groups">{t('home.groups.empty.action')}</Link>}
        />
      )}
      {groups.length > 0 && (
        <ul className="home-groups__list">
          {groups.map((g) => {
            const busy = running.find((r) => r.name === g.name)
            const n = g.profiles.length
            return (
              <li key={g.name} className="home-group">
                <div className="home-group__text">
                  <span className="home-name">{g.name}</span>
                  <span className="home-dim small">
                    {n === 1 ? t('home.groups.count.one') : t('home.groups.count.other', { n })}
                  </span>
                </div>
                <div className="home-actions">
                  <Button
                    variant="tonal"
                    size="sm"
                    loading={busy?.action === 'connect'}
                    disabled={!!busy}
                    aria-label={t('home.groups.connect', { name: g.name })}
                    onClick={() => action.mutate({ name: g.name, action: 'connect', profiles: g.profiles })}
                  >
                    {t('home.action.connect')}
                  </Button>
                  <Button
                    variant="outlined"
                    size="sm"
                    loading={busy?.action === 'disconnect'}
                    disabled={!!busy}
                    aria-label={t('home.groups.disconnect', { name: g.name })}
                    onClick={() => action.mutate({ name: g.name, action: 'disconnect', profiles: g.profiles })}
                  >
                    {t('home.action.disconnect')}
                  </Button>
                </div>
              </li>
            )
          })}
        </ul>
      )}
    </Card>
  )
}
