// Perfiles (/profiles) — the VPN profiles Kongtrol manages · v1.2.0 · 2026-11-19
//
// Parity contract: docs/dashboard-parity.md §3.2 "Tab: VPN Profiles". Reads
// profiles, setup state and installed VPN apps once per visit (no polling,
// like the classic Studio); tunnel states come from the shared live feed.
// Changes write kongtrol.yaml + the OS keychain and need a restart before
// they connect: the banner offers it. With no profiles the page is the
// first-run guide.
import { useCallback, useMemo, useRef, useState } from 'react'
import { useAddSheet } from '../app/addLink'
import { usePreferences } from '../app/preferences-context'
import { Monogram, ProfilesHeader } from '../components/profiles/Bits'
import { specOf } from '../components/profiles/catalog'
import { ConfirmDelete } from '../components/profiles/ConfirmDelete'
import { backupOf, matchesFilter } from '../components/profiles/draft'
import { deleteRefusal, localizeError, type DeleteRefusal } from '../components/profiles/errors'
import { FirstRun } from '../components/profiles/FirstRun'
import { SearchIcon } from '../components/profiles/icons'
import { ProfileEditor } from '../components/profiles/ProfileEditor'
import { ProfileList, type ProfileRow } from '../components/profiles/ProfileList'
import { RestartBanner } from '../components/profiles/Restart'
import '../components/profiles/profiles.css'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { PlusIcon } from '../components/ui/icons'
import { EmptyState, ErrorState, Skeleton } from '../components/ui/States'
import type { TunnelMetrics } from '../api/types'
import {
  useDeleteProfile,
  useDetectedClients,
  useGroupsList,
  useProfilesQuery,
  useRestartPending,
  useSetupQuery,
  type AdapterType,
  type VPNProfile,
} from '../data/profiles'
import { useTunnelsQuery } from '../data/resources'

type EditorState = { mode: 'closed' } | { mode: 'edit'; profile: VPNProfile; seq: number }

// Row objects are cached per profile object (structural sharing keeps it
// identical while unchanged), so a live tick re-renders only the row whose
// tunnel or restart flag changed (DataList memoizes rows).
const rowCache = new WeakMap<VPNProfile, ProfileRow>()

function rowFor(profile: VPNProfile, tunnel: TunnelMetrics | undefined, pendingRestart: boolean): ProfileRow {
  const prev = rowCache.get(profile)
  if (prev && prev.tunnel === tunnel && prev.pendingRestart === pendingRestart) return prev
  const row = { profile, tunnel, pendingRestart }
  rowCache.set(profile, row)
  return row
}

export function ProfilesPage() {
  const { t } = usePreferences()
  const profiles = useProfilesQuery()
  const setup = useSetupQuery()
  const tunnels = useTunnelsQuery()
  const groups = useGroupsList()
  const pending = useRestartPending()
  const list = useMemo(() => profiles.data ?? [], [profiles.data])
  const empty = profiles.isSuccess && list.length === 0
  const detected = useDetectedClients()
  const [filter, setFilter] = useState('')
  const [editor, setEditor] = useState<EditorState>({ mode: 'closed' })
  const [toDelete, setToDelete] = useState<VPNProfile | null>(null)
  const seq = useRef(0)

  // The daemon's own refusal (every reference at once, or the last profile)
  // reopens the dialog with the list instead of a toast.
  // The profile being deleted is kept here: the optimistic delete already took
  // it out of `list` when the refusal arrives.
  const [refused, setRefused] = useState<{ profile: VPNProfile; refusal: DeleteRefusal } | null>(null)
  const deleting$ = useRef<VPNProfile | null>(null)
  const remove = useDeleteProfile(
    (name) => t('profiles.toast.deleted', { name }),
    (err) => localizeError(err, t),
    (err, name) => {
      const refusal = deleteRefusal(err)
      const profile = deleting$.current?.name === name ? deleting$.current : null
      if (!refusal || !profile) return false
      setRefused({ profile, refusal })
      return true
    },
  )

  const rows = useMemo(() => {
    const byName = new Map((tunnels.data ?? []).map((tn) => [tn.Name, tn]))
    return list.map((p) => rowFor(p, byName.get(p.name), pending))
  }, [list, tunnels.data, pending])
  const shown = useMemo(() => rows.filter((r) => matchesFilter(r.profile, filter)), [rows, filter])

  // Creating opens the shell's "＋ Agregar" sheet (ADR-0052).
  const { open: openAdd } = useAddSheet()
  const openCreate = useCallback((type?: AdapterType) => openAdd({ mode: 'vpn', ...(type ? { type } : {}) }), [openAdd])
  const onEdit = useCallback((profile: VPNProfile) => setEditor({ mode: 'edit', profile, seq: ++seq.current }), [])
  const onDelete = useCallback((profile: VPNProfile) => setToDelete(profile), [])
  const close = useCallback(() => setEditor({ mode: 'closed' }), [])

  // What still depends on the profile: checked here so the dialog can say what to change first.
  const blocked = useMemo(() => {
    if (refused) return refused.refusal
    if (!toDelete) return null
    if (list.length <= 1) return { message: t('profiles.err.last'), fix: t('profiles.err.last.fix') }
    const inGroups = (groups.data ?? []).filter((g) => g.profiles.includes(toDelete.name)).map((g) => g.name)
    if (inGroups.length > 0) {
      return {
        message: t('profiles.err.inGroup', { a: toDelete.name, b: inGroups.join(', ') }),
        fix: t('profiles.err.inGroup.fix', { a: toDelete.name, b: inGroups.join(', ') }),
      }
    }
    const backups = backupOf(toDelete.name, list)
    if (backups.length > 0) {
      return {
        message: t('profiles.err.isBackup', { a: toDelete.name, b: backups.join(', ') }),
        fix: t('profiles.err.isBackup.fix', { a: toDelete.name, b: backups.join(', ') }),
      }
    }
    return null
  }, [refused, toDelete, list, groups.data, t])
  const deleting = refused?.profile ?? toDelete
  const busyNames = useMemo(
    () => new Set((tunnels.data ?? []).filter((tn) => tn.Status === 'connected' || tn.Status === 'connecting').map((tn) => tn.Name)),
    [tunnels.data],
  )
  const connectedNames = useMemo(
    () => new Set((tunnels.data ?? []).filter((tn) => tn.Status === 'connected').map((tn) => tn.Name)),
    [tunnels.data],
  )

  return (
    <div className="kp-page">
      <ProfilesHeader />
      <RestartBanner />

      {profiles.isPending && (
        <Card title={t('profiles.list.title')}>
          <Skeleton lines={4} label={t('profiles.loading')} />
        </Card>
      )}

      {profiles.isError && !profiles.data && (
        <Card title={t('profiles.list.title')}>
          <ErrorState error={profiles.error} onRetry={() => void profiles.refetch()} />
        </Card>
      )}

      {empty && (
        <FirstRun
          setupMode={!!setup.data?.needs_setup}
          detected={detected.data ?? []}
          detecting={detected.isPending}
          onStart={openCreate}
        />
      )}

      {profiles.data && list.length > 0 && (
        <Card
          title={t('profiles.list.title')}
          aside={
            <Button icon={<PlusIcon />} onClick={() => openCreate()}>
              {t('profiles.action.new')}
            </Button>
          }
        >
          {list.length > 3 && (
            <div className="kp-filter">
              <SearchIcon />
              <label htmlFor="kp-filter" className="visually-hidden">
                {t('profiles.filter.label')}
              </label>
              <input
                id="kp-filter"
                type="search"
                className="k-input"
                placeholder={t('profiles.filter.placeholder')}
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
              />
            </div>
          )}
          {shown.length > 0 ? (
            <ProfileList rows={shown} onEdit={onEdit} onDelete={onDelete} />
          ) : (
            <EmptyState
              icon={<SearchIcon />}
              title={t('profiles.filter.none', { q: filter.trim() })}
              body={t('profiles.filter.noneBody')}
              action={
                <Button variant="outlined" onClick={() => setFilter('')}>
                  {t('profiles.filter.clear')}
                </Button>
              }
            />
          )}
        </Card>
      )}

      {profiles.data && (
        <p className="kp-muted kp-small kp-footnote">
          {t('profiles.note.saved')}
          {setup.data?.config_path && (
            <>
              {' '}
              {t('profiles.note.file')} <code className="mono">{setup.data.config_path}</code>
            </>
          )}
        </p>
      )}

      {editor.mode === 'edit' && (
        <ProfileEditor
          key={editor.seq}
          open
          onClose={close}
          editing={editor.profile}
          profiles={list}
          busy={busyNames.has(editor.profile.name)}
          connected={connectedNames.has(editor.profile.name)}
        />
      )}

      <ConfirmDelete
        open={!!deleting}
        title={deleting ? t('profiles.confirm.title', { name: deleting.name }) : ''}
        body={
          deleting && (
            <span className="kp-confirm">
              <Monogram spec={specOf(deleting.type)} />
              <span>{t('profiles.confirm.body')}</span>
            </span>
          )
        }
        blocked={blocked}
        onCancel={() => {
          setToDelete(null)
          setRefused(null)
        }}
        onConfirm={() => {
          deleting$.current = toDelete
          if (toDelete) remove.mutate(toDelete.name)
          setToDelete(null)
        }}
      />
    </div>
  )
}
