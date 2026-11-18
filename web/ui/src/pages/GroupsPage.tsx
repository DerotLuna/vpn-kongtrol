// Grupos (/profiles/groups) — connect several VPNs at once · v1.0.0 · 2026-11-18
//
// Parity contract: docs/dashboard-parity.md §3.2 "Tab: Groups". Group CRUD
// (optimistic, rolled back on refusal) and connect / disconnect / reload a
// whole group. Every answer is read, not assumed (legacy bug §5.1 item 3):
// profiles the running daemon could not start are named, with "Reiniciar
// ahora" when a restart is what they need.
import { useCallback, useMemo, useState } from 'react'
import { Link } from 'react-router'
import type { Group, TunnelMetrics } from '../api/types'
import { describeError } from '../api/errors'
import { usePreferences } from '../app/preferences-context'
import { ProfilesHeader } from '../components/profiles/Bits'
import { ConfirmDelete } from '../components/profiles/ConfirmDelete'
import { localizeError } from '../components/profiles/errors'
import { GroupEditor } from '../components/profiles/GroupEditor'
import { EditIcon, PowerIcon, TrashIcon } from '../components/profiles/icons'
import { RestartBanner } from '../components/profiles/Restart'
import { useRestartAction } from '../components/profiles/useRestartAction'
import '../components/profiles/profiles.css'
import { Button, IconButton } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { DataList, type Column } from '../components/ui/DataList'
import { PlugIcon, PlusIcon, RefreshIcon } from '../components/ui/icons'
import { EmptyState, ErrorState, Skeleton } from '../components/ui/States'
import { StatusPill } from '../components/ui/StatusPill'
import {
  useDeleteGroup,
  useGroupAction,
  useGroupsList,
  useProfilesQuery,
  useSaveGroup,
  type GroupAction,
  type GroupActionResult,
} from '../data/profiles'
import { useTunnelsQuery } from '../data/resources'
import { toasts } from '../data/toast'

interface Row {
  group: Group
  connected: number
}

type EditorState = { mode: 'closed' } | { mode: 'create'; seq: number } | { mode: 'edit'; group: Group; seq: number }

export function GroupsPage() {
  const { t } = usePreferences()
  const groups = useGroupsList()
  const profiles = useProfilesQuery()
  const tunnels = useTunnelsQuery()
  const action = useGroupAction()
  const { restart } = useRestartAction()
  const [editor, setEditor] = useState<EditorState>({ mode: 'closed' })
  const [toDelete, setToDelete] = useState<Group | null>(null)
  const [busy, setBusy] = useState<string>('')

  const localize = useCallback((err: unknown) => localizeError(err, t), [t])
  const save = useSaveGroup(
    (v) => t(v.original ? 'groups.toast.updated' : 'groups.toast.created', { name: v.original ?? v.name }),
    localize,
  )
  const remove = useDeleteGroup((name) => t('groups.toast.deleted', { name }), localize)

  const restartAction = useMemo(() => ({ label: t('profiles.restart.button'), run: () => void restart() }), [restart, t])

  const report = useCallback(
    (name: string, members: string[], res: GroupActionResult) => {
      if (res.action === 'connect') {
        const { started = [], already_connected: already = [] } = res.result
        const skipped = members.filter((p) => !started.includes(p) && !already.includes(p))
        if (skipped.length > 0) {
          toasts.show({
            tone: 'bad',
            message: t('groups.toast.skipped', { name, profiles: skipped.join(', ') }),
            detail: t('groups.toast.skipped.fix'),
            action: restartAction,
          })
        } else if (started.length === 0) {
          toasts.show({ tone: 'info', message: t('groups.toast.already', { name }) })
        } else {
          toasts.show({ tone: 'info', message: t('groups.toast.connecting', { name, n: started.length }) })
        }
        return
      }
      if (res.action === 'disconnect') {
        toasts.show({ tone: 'ok', message: t('groups.toast.disconnecting', { name }) })
        return
      }
      const r = res.result
      if (r.status === 'restart_required') {
        toasts.show({
          tone: 'bad',
          message: t('groups.toast.restartRequired', { name }),
          detail: t('groups.toast.restartRequired.fix', { profiles: (r.missing_profiles ?? []).join(', ') }),
          action: restartAction,
        })
      } else if ((r.restarted ?? []).length === 0) {
        toasts.show({ tone: 'info', message: t('groups.toast.nothingToReload', { name }) })
      } else {
        toasts.show({ tone: 'info', message: t('groups.toast.reloading', { name, n: (r.restarted ?? []).length }) })
      }
    },
    [restartAction, t],
  )

  const run = useCallback(
    (group: Group, kind: GroupAction) => {
      setBusy(`${group.name}:${kind}`)
      action.mutate(
        { name: group.name, action: kind },
        {
          onSuccess: (res) => report(group.name, group.profiles, res),
          onError: (err) => {
            const failed = kind === 'connect' ? 'groups.alert.connectFailed' : kind === 'disconnect' ? 'groups.alert.disconnectFailed' : 'groups.alert.reloadFailed'
            const { message, fix } = describeError(localizeError(err, t), t)
            toasts.show({ tone: 'bad', message: t(failed, { name: group.name }), detail: `${message} · ${fix}` })
          },
          onSettled: () => setBusy(''),
        },
      )
    },
    [action, report, t],
  )

  const rows = useMemo<Row[]>(() => {
    const up = new Set((tunnels.data ?? []).filter((tn: TunnelMetrics) => tn.Status === 'connected').map((tn) => tn.Name))
    return (groups.data ?? []).map((group) => ({ group, connected: group.profiles.filter((p) => up.has(p)).length }))
  }, [groups.data, tunnels.data])

  const columns = useMemo<Column<Row>[]>(
    () => [
      { id: 'name', header: t('groups.col.name'), cell: ({ group }) => <span className="mono kp-vpn__name">{group.name}</span> },
      {
        id: 'profiles',
        header: t('groups.col.profiles'),
        cell: ({ group }) => (
          <ul className="kp-names" aria-label={t('groups.col.profiles')}>
            {group.profiles.map((p) => (
              <li key={p} className="mono">
                {p}
              </li>
            ))}
          </ul>
        ),
      },
      {
        id: 'state',
        header: t('groups.col.state'),
        cell: ({ group, connected }) =>
          connected === 0 ? (
            <StatusPill tone="idle">{t('groups.state.none')}</StatusPill>
          ) : (
            <StatusPill tone={connected === group.profiles.length ? 'ok' : 'on'}>
              {t('groups.state.some', { n: connected, total: group.profiles.length })}
            </StatusPill>
          ),
      },
      {
        id: 'actions',
        header: t('profiles.col.actions'),
        hideHeader: true,
        cell: ({ group }) => (
          <span className="kp-actions kp-actions--group">
            <Button variant="tonal" icon={<PlugIcon />} loading={busy === `${group.name}:connect`} onClick={() => run(group, 'connect')}>
              {t('groups.action.connect')}
            </Button>
            <Button variant="outlined" icon={<PowerIcon />} loading={busy === `${group.name}:disconnect`} onClick={() => run(group, 'disconnect')}>
              {t('groups.action.disconnect')}
            </Button>
            <IconButton
              label={t('groups.action.reloadName', { name: group.name })}
              icon={<RefreshIcon />}
              disabled={busy === `${group.name}:reload`}
              onClick={() => run(group, 'reload')}
            />
            <IconButton label={t('profiles.action.editName', { name: group.name })} icon={<EditIcon />} onClick={() => setEditor({ mode: 'edit', group, seq: Date.now() })} />
            <IconButton
              label={t('profiles.action.deleteName', { name: group.name })}
              icon={<TrashIcon />}
              className="kp-danger-icon"
              onClick={() => setToDelete(group)}
            />
          </span>
        ),
      },
    ],
    [busy, run, t],
  )

  const profileList = profiles.data ?? []
  const noProfiles = profiles.isSuccess && profileList.length === 0

  return (
    <div className="kp-page">
      <ProfilesHeader />
      <RestartBanner />

      <Card
        title={t('groups.list.title')}
        aside={
          !noProfiles &&
          groups.data &&
          groups.data.length > 0 && (
            <Button icon={<PlusIcon />} onClick={() => setEditor({ mode: 'create', seq: Date.now() })}>
              {t('groups.action.new')}
            </Button>
          )
        }
        busy={groups.isPending}
      >
        <p className="kp-muted kp-lead">{t('groups.lead')}</p>
        {groups.isPending && <Skeleton lines={3} label={t('groups.loading')} />}
        {groups.isError && !groups.data && <ErrorState error={groups.error} onRetry={() => void groups.refetch()} />}
        {groups.data && groups.data.length === 0 && (
          <EmptyState
            title={t(noProfiles ? 'groups.empty.noProfiles' : 'groups.empty.title')}
            body={t(noProfiles ? 'groups.empty.noProfiles.body' : 'groups.empty.body')}
            action={
              noProfiles ? (
                <Link className="k-btn k-btn--filled k-btn--md" to="/profiles">
                  <span className="k-btn__label">{t('groups.empty.goProfiles')}</span>
                </Link>
              ) : (
                <Button icon={<PlusIcon />} onClick={() => setEditor({ mode: 'create', seq: Date.now() })}>
                  {t('groups.action.new')}
                </Button>
              )
            }
          />
        )}
        {groups.data && groups.data.length > 0 && (
          <DataList caption={t('groups.list.caption')} columns={columns} rows={rows} rowKey={(r) => r.group.name} />
        )}
      </Card>

      <details className="kp-more kp-tip">
        <summary>{t('groups.reload.what')}</summary>
        <p className="kp-muted">{t('groups.reload.tip')}</p>
      </details>

      {editor.mode !== 'closed' && (
        <GroupEditor
          key={editor.seq}
          open
          onClose={() => setEditor({ mode: 'closed' })}
          editing={editor.mode === 'edit' ? editor.group : undefined}
          groups={groups.data ?? []}
          profiles={profileList}
          onSave={(vars) => save.mutate(vars)}
        />
      )}

      <ConfirmDelete
        open={!!toDelete}
        title={toDelete ? t('groups.confirm.title', { name: toDelete.name }) : ''}
        body={t('groups.confirm.body')}
        onCancel={() => setToDelete(null)}
        onConfirm={() => {
          if (toDelete) remove.mutate(toDelete.name)
          setToDelete(null)
        }}
      />
    </div>
  )
}
