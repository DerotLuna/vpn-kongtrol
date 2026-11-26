// Reglas (/rules) — which sites and networks go through each VPN · v1.3.0 · 2026-11-24
//
// Parity with the classic Studio "Policies" tab (docs/dashboard-parity.md),
// plus: success toasts, on_tunnel_down per row, overlaps explained, a
// "Probar un destino" route view (`kongtrol policy show`), the read-only
// "Rutas que trae cada VPN" (ADR-0107, `kongtrol routes vpn`) and starting
// points for people who don't write globs or CIDRs. The editor opens
// inline above the list (not a modal); delete asks first.
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useAddSheet } from '../app/addLink'
import { usePreferences } from '../app/preferences-context'
import { useSearchFocus } from '../app/searchFocus'
import { PolicyEditor } from '../components/rules/PolicyEditor'
import { Overlaps, PolicyList } from '../components/rules/PolicyList'
import { ConfirmDelete, RulesHeader } from '../components/rules/shared'
import { TestDestination } from '../components/rules/TestDestination'
import { VpnRoutes } from '../components/rules/VpnRoutes'
import { FileIcon, RouteIcon } from '../components/rules/icons'
import { draftFromPolicy, type PolicyDraft } from '../components/rules/editorModel'
import type { Policy } from '../components/rules/model'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { EmptyState, ErrorState, Skeleton } from '../components/ui/States'
import { PlusIcon, RefreshIcon } from '../components/ui/icons'
import { useAppRoutedProfiles, useProfilePriorities } from '../data/profiles'
import { useDeletePolicy, usePoliciesQuery, usePolicyMetaQuery, useReloadPolicies } from '../data/rules'
import '../components/rules/rules.css'

interface EditorState {
  key: number
  original: string
  initial: PolicyDraft
}

export function RulesPage() {
  const { t } = usePreferences()
  const policies = usePoliciesQuery()
  useSearchFocus(policies.isSuccess)
  const meta = usePolicyMetaQuery()
  const reload = useReloadPolicies()
  const remove = useDeletePolicy()
  const priorities = useProfilePriorities()
  const appRouted = useAppRoutedProfiles()
  const [editor, setEditor] = useState<EditorState | null>(null)
  const [toDelete, setToDelete] = useState<Policy | null>(null)

  const profiles = useMemo(() => meta.data?.profiles ?? [], [meta.data])
  const list = policies.data ?? []

  // New rules open the shell's "＋ Agregar" sheet (ADR-0055).
  const { open: openAdd } = useAddSheet()
  const openNew = useCallback(() => openAdd({ mode: 'rule' }), [openAdd])
  const openEdit = useCallback(
    (p: Policy) => setEditor((e) => ({ key: (e?.key ?? 0) + 1, original: p.name, initial: draftFromPolicy(p) })),
    [],
  )
  const askDelete = useCallback((p: Policy) => setToDelete(p), [])

  // Opening the editor moves focus to its title, so keyboard and screen
  // reader users land where the form starts.
  useEffect(() => {
    if (editor) document.querySelector<HTMLElement>('.r-editor [data-autofocus]')?.focus()
  }, [editor])

  const closeEditor = () => {
    setEditor(null)
    document.querySelector<HTMLElement>('.r-list [data-new]')?.focus()
  }

  return (
    <div className="r-page">
      <RulesHeader />

      {editor && (
        <PolicyEditor
          key={editor.key}
          original={editor.original}
          initial={editor.initial}
          policies={list}
          profiles={profiles}
          onClose={closeEditor}
        />
      )}

      <Card
        className="r-list"
        title={t('rules.list.title')}
        busy={policies.isFetching}
        aside={
          list.length > 0 && (
            <Button icon={<PlusIcon />} onClick={openNew} aria-haspopup="dialog" data-new="">
              {t('rules.action.new')}
            </Button>
          )
        }
      >
        {policies.isPending ? (
          <Skeleton lines={4} label={t('rules.list.loading')} />
        ) : policies.isError ? (
          <ErrorState error={policies.error} onRetry={() => void policies.refetch()} />
        ) : list.length === 0 ? (
          <EmptyState
            icon={<RouteIcon />}
            title={t('rules.empty.title')}
            body={t('rules.empty.body')}
            action={
              <Button icon={<PlusIcon />} onClick={openNew} aria-haspopup="dialog" data-new="">
                {t('rules.empty.action')}
              </Button>
            }
          />
        ) : (
          <>
            <p className="muted r-list__lead">{t('rules.list.what')}</p>
            <PolicyList policies={list} priorities={priorities} appRouted={appRouted} onEdit={openEdit} onDelete={askDelete} />
            <Overlaps policies={list} priorities={priorities} />
          </>
        )}

        <div className="r-reload">
          <FileIcon />
          <div>
            <p>
              {t('rules.reload.what')}
              {meta.data?.config_path && (
                <>
                  {' '}
                  <code className="r-path">{meta.data.config_path}</code>
                </>
              )}
            </p>
          </div>
          <Button variant="outlined" size="sm" icon={<RefreshIcon />} loading={reload.isPending} onClick={() => reload.mutate()}>
            {t('rules.reload.action')}
          </Button>
        </div>
      </Card>

      <VpnRoutes />

      {list.length > 0 && <TestDestination policies={list} />}

      <ConfirmDelete
        open={toDelete !== null}
        title={t('rules.delete.title', { name: toDelete?.name ?? '' })}
        body={t('rules.delete.body')}
        confirm={t('rules.delete.confirm')}
        onConfirm={() => {
          if (!toDelete) return
          if (editor?.original === toDelete.name) setEditor(null)
          remove.mutate(toDelete.name)
        }}
        onClose={() => setToDelete(null)}
      />
    </div>
  )
}
