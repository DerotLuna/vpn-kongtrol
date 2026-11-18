// Horarios (/rules/schedule) — VPNs that connect and disconnect by themselves · v1.1.0 · 2026-11-18
//
// Parity with the classic Settings "Scheduler" tab: on/off (saves at once,
// and only that field), check interval (its own save), rule CRUD, refresh.
// Each rule reads as a sentence and says whether it is in effect right now;
// each can be paused without deleting it (per-rule `enabled`). The daemon
// reloads the scheduler on every save (ADR-0042), so changes apply at once:
// no restart note.
import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { usePreferences } from '../app/preferences-context'
import { useUnsavedChanges } from '../app/unsaved'
import { ClockIcon, EditIcon, TrashIcon } from '../components/rules/icons'
import {
  DURATION_RE,
  emptySchedule,
  isActiveAt,
  scheduleDraftFrom,
  type ScheduleDraft,
  type ScheduleRule,
} from '../components/rules/model'
import { describeSchedule } from '../components/rules/preview'
import { ScheduleEditor } from '../components/rules/ScheduleEditor'
import { ConfirmDelete, RulesHeader } from '../components/rules/shared'
import { Button, IconButton } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { DataList, type Column } from '../components/ui/DataList'
import { TextField } from '../components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '../components/ui/States'
import { StatusPill } from '../components/ui/StatusPill'
import { Switch } from '../components/ui/Switch'
import { PlusIcon, RefreshIcon } from '../components/ui/icons'
import {
  useDeleteSchedule,
  usePolicyMetaQuery,
  useSaveSchedulerSettings,
  useScheduleRulesQuery,
  useSettingsQuery,
  useToggleSchedule,
} from '../data/rules'
import '../components/rules/rules.css'

interface EditorState {
  key: number
  original?: string | undefined
  initial: ScheduleDraft
}

/** Re-evaluates "in effect now" each minute. */
function useMinute(): Date {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const id = window.setInterval(() => setNow(new Date()), 30_000)
    return () => window.clearInterval(id)
  }, [])
  return now
}

export function SchedulePage() {
  const { t, lang } = usePreferences()
  const rules = useScheduleRulesQuery()
  const meta = usePolicyMetaQuery()
  const settings = useSettingsQuery()
  const saveSettings = useSaveSchedulerSettings()
  const remove = useDeleteSchedule()
  const toggle = useToggleSchedule()
  const [editor, setEditor] = useState<EditorState | null>(null)
  const [toDelete, setToDelete] = useState<ScheduleRule | null>(null)
  const now = useMinute()

  const list = rules.data ?? []
  const enabled = settings.data?.scheduler_enabled ?? false
  const metaProfiles = useMemo(() => meta.data?.profiles ?? [], [meta.data])

  const openNew = useCallback(
    (office = false) =>
      setEditor((e) => ({
        key: (e?.key ?? 0) + 1,
        initial: {
          ...emptySchedule(),
          ...(office ? { name: t('schedule.template.office.name'), profiles: metaProfiles.slice(0, 1) } : {}),
        },
      })),
    [metaProfiles, t],
  )
  const openEdit = useCallback(
    (r: ScheduleRule) => setEditor((e) => ({ key: (e?.key ?? 0) + 1, original: r.name, initial: scheduleDraftFrom(r) })),
    [],
  )

  useEffect(() => {
    if (editor) document.querySelector<HTMLElement>('.r-editor [data-autofocus]')?.focus()
  }, [editor])

  // A rule may name a profile that no longer exists; keep it visible so it can be unticked.
  const editorProfiles = useMemo(
    () => [...new Set([...metaProfiles, ...(editor?.initial.profiles ?? [])])],
    [metaProfiles, editor],
  )

  const columns = useMemo<Column<ScheduleRule>[]>(
    () => [
      { id: 'name', header: t('schedule.list.name'), cell: (r) => r.name },
      { id: 'when', header: t('schedule.list.when'), cell: (r) => <span className="r-sentence">{describeSchedule(r, t, lang)}</span> },
      {
        id: 'now',
        header: t('schedule.list.now'),
        cell: (r) =>
          !enabled ? (
            <StatusPill tone="idle">{t('schedule.now.paused')}</StatusPill>
          ) : r.enabled === false ? (
            <StatusPill tone="idle">{t('schedule.now.ruleOff')}</StatusPill>
          ) : isActiveAt(r, now) ? (
            <StatusPill tone="on">{t('schedule.now.in')}</StatusPill>
          ) : (
            <StatusPill tone="idle">{t('schedule.now.out')}</StatusPill>
          ),
      },
      {
        id: 'on',
        header: t('schedule.list.on'),
        cell: (r) => (
          <span className="r-rule-switch">
            <Switch
              checked={r.enabled !== false}
              onChange={(on) => toggle.mutate({ name: r.name, enabled: on })}
              label={<span className="visually-hidden">{t('schedule.rule.toggle', { name: r.name })}</span>}
              onText={t('schedule.rule.on')}
              offText={t('schedule.rule.off')}
            />
          </span>
        ),
      },
      {
        id: 'actions',
        header: t('rules.list.actions'),
        hideHeader: true,
        cell: (r) => (
          <span className="r-row-actions">
            <IconButton label={t('rules.action.editNamed', { name: r.name })} icon={<EditIcon />} onClick={() => openEdit(r)} />
            <IconButton label={t('rules.action.deleteNamed', { name: r.name })} icon={<TrashIcon />} onClick={() => setToDelete(r)} />
          </span>
        ),
      },
    ],
    // `now` ticks every 30 s, so "in effect now" stays true.
    [t, lang, enabled, now, openEdit, toggle],
  )

  return (
    <div className="r-page">
      <RulesHeader />

      <Card title={t('schedule.switch.title')} className="r-sched-settings">
        {settings.isPending ? (
          <Skeleton lines={2} label={t('schedule.loading')} />
        ) : settings.isError ? (
          <ErrorState error={settings.error} onRetry={() => void settings.refetch()} />
        ) : (
          <>
            <Switch
              checked={enabled}
              onChange={(on) => saveSettings.mutate({ scheduler_enabled: on })}
              label={t('schedule.switch.label')}
              description={t('schedule.switch.what')}
              onText={t('schedule.switch.on')}
              offText={t('schedule.switch.off')}
              pending={saveSettings.isPending}
            />
            <IntervalForm key={settings.data.scheduler_interval} current={settings.data.scheduler_interval} />
          </>
        )}
      </Card>

      {editor && (
        <ScheduleEditor
          key={editor.key}
          original={editor.original}
          initial={editor.initial}
          rules={list}
          profiles={editorProfiles}
          onClose={() => setEditor(null)}
        />
      )}

      <Card
        className="r-list"
        title={t('schedule.list.title')}
        busy={rules.isFetching}
        aside={
          <span className="r-row-actions">
            <IconButton
              label={t('schedule.action.refresh')}
              icon={<RefreshIcon />}
              onClick={() => {
                void rules.refetch()
                void settings.refetch()
              }}
            />
            {list.length > 0 && (
              <Button icon={<PlusIcon />} onClick={() => openNew()}>
                {t('schedule.action.new')}
              </Button>
            )}
          </span>
        }
      >
        {rules.isPending ? (
          <Skeleton lines={3} label={t('schedule.loading')} />
        ) : rules.isError ? (
          <ErrorState error={rules.error} onRetry={() => void rules.refetch()} />
        ) : list.length === 0 ? (
          <EmptyState
            icon={<ClockIcon />}
            title={t('schedule.empty.title')}
            body={t('schedule.empty.body')}
            action={
              <span className="r-row-actions">
                <Button icon={<PlusIcon />} onClick={() => openNew(true)}>
                  {t('schedule.empty.action')}
                </Button>
                <Button variant="text" onClick={() => openNew()}>
                  {t('schedule.action.new')}
                </Button>
              </span>
            }
          />
        ) : (
          <DataList caption={t('schedule.list.caption')} columns={columns} rows={list} rowKey={(r) => r.name} />
        )}
      </Card>

      <ConfirmDelete
        open={toDelete !== null}
        title={t('schedule.delete.title', { name: toDelete?.name ?? '' })}
        body={t('schedule.delete.body')}
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

/** "Revisar cada": its own save, so the switch never saves a half-typed interval. */
function IntervalForm({ current }: { current: string }) {
  const { t } = usePreferences()
  const save = useSaveSchedulerSettings()
  const [value, setValue] = useState(current || '1m')
  const [invalid, setInvalid] = useState(false)
  const dirty = value.trim() !== (current || '1m')
  useUnsavedChanges(dirty)

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    const v = value.trim()
    if (!DURATION_RE.test(v)) {
      setInvalid(true)
      return
    }
    setInvalid(false)
    save.mutate({ scheduler_interval: v })
  }

  return (
    <form className="r-inline-form r-interval" onSubmit={onSubmit} noValidate>
      <TextField
        name="interval"
        label={t('schedule.interval.label')}
        hint={t('schedule.interval.hint')}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        error={invalid ? { message: t('schedule.interval.err', { value }), fix: t('schedule.interval.err.fix') } : null}
        mono
        spellCheck={false}
        autoCapitalize="off"
      />
      <Button type="submit" variant="outlined" disabled={!dirty} loading={save.isPending}>
        {t('schedule.interval.save')}
      </Button>
    </form>
  )
}
