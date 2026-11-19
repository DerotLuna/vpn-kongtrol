// Kongtrol UI — groups quick launch (Inicio) · v1.1.0 · 2026-11-19
//
// Connect or disconnect a whole group in one click. The group's answer is
// read, not ignored (legacy bug): which VPNs started, which were already up
// and which the running Kongtrol does not know yet. Each group shows one
// main action (Conectar in amber while a member is down, else Desconectar
// outlined) plus Desconectar as an icon button when part of it is up — the
// same layout as /profiles/groups — and stays in progress until every
// member reached its end state (data/transition.ts). Editing groups lives
// in /profiles/groups.
import { useMutationState } from '@tanstack/react-query'
import { Link } from 'react-router'
import type { Group, TunnelMetrics } from '../../api/types'
import { usePreferences } from '../../app/preferences-context'
import { homeMutationKeys, useGroupAction, useGroupsList, type GroupAction } from '../../data/home'
import { useTunnelsQuery } from '../../data/resources'
import { usePendingAny } from '../../data/transition'
import { PowerIcon } from '../profiles/icons'
import { Button, IconButton } from '../ui/Button'
import { Card } from '../ui/Card'
import { PlugIcon, RefreshIcon } from '../ui/icons'
import { EmptyState, ErrorState, Skeleton } from '../ui/States'

export function GroupsCard() {
  const { t } = usePreferences()
  const query = useGroupsList()
  const action = useGroupAction()
  const tunnels = useTunnelsQuery().data
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
      aside={<IconButton size="sm" variant="outlined" label={t('home.groups.refresh')} icon={<RefreshIcon />} onClick={() => void query.refetch()} />}
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
          {groups.map((g) => (
            <GroupRow
              key={g.name}
              group={g}
              tunnels={tunnels}
              busy={running.find((r) => r.name === g.name)?.action}
              onRun={(a) => action.mutate({ name: g.name, action: a, profiles: g.profiles })}
            />
          ))}
        </ul>
      )}
    </Card>
  )
}

function GroupRow({
  group,
  tunnels,
  busy,
  onRun,
}: {
  group: Group
  tunnels: TunnelMetrics[] | undefined
  busy: GroupAction | undefined
  onRun: (a: GroupAction) => void
}) {
  const { t } = usePreferences()
  const moving = usePendingAny(group.profiles)
  const n = group.profiles.length
  const up = group.profiles.filter((p) => tunnels?.some((tn) => tn.Name === p && tn.Status === 'connected')).length
  const main: GroupAction = up < n ? 'connect' : 'disconnect'
  const mainBusy = busy === main || (moving && busy === undefined)
  const word =
    main === 'connect'
      ? t(mainBusy ? 'home.action.connecting' : 'home.action.connect')
      : t(mainBusy ? 'home.action.disconnecting' : 'home.action.disconnect')
  return (
    <li className="home-group">
      <div className="home-group__text">
        <span className="home-name">{group.name}</span>
        <span className="home-dim small">{n === 1 ? t('home.groups.count.one') : t('home.groups.count.other', { n })}</span>
      </div>
      <div className="home-actions">
        <Button
          variant={main === 'connect' ? 'filled' : 'outlined'}
          size="sm"
          className="home-main-act"
          icon={main === 'connect' ? <PlugIcon /> : <PowerIcon />}
          loading={mainBusy}
          disabled={!!busy && !mainBusy}
          aria-label={`${word} ${group.name}`}
          onClick={() => onRun(main)}
        >
          {word}
        </Button>
        {main === 'connect' && up > 0 && (
          <IconButton
            size="sm"
            variant="outlined"
            label={t('home.groups.disconnect', { name: group.name })}
            icon={<PowerIcon />}
            loading={busy === 'disconnect'}
            disabled={!!busy}
            onClick={() => onRun('disconnect')}
          />
        )}
      </div>
    </li>
  )
}
