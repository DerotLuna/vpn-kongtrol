// Reglas — saved rules in evaluation order, and where they overlap · v1.1.0 · 2026-11-24
//
// The "Sale por" cell adds the order hint (ADR-0108) only on rows in an
// overlap the order settles: "Prioridad N" or "por posición". Elsewhere the
// order changes nothing, and new profiles all start at 10, so a hint on
// every row would be noise. It links to that profile's editor with the
// priority field focused.
import { memo, useMemo } from 'react'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { useTogglePolicy } from '../../data/rules'
import type { MessageKey } from '../../i18n/messages'
import { IconButton } from '../ui/Button'
import { DataList, type Column } from '../ui/DataList'
import { Switch } from '../ui/Switch'
import { Tooltip } from '../ui/Tooltip'
import { EditIcon, TrashIcon } from './icons'
import { findOverlaps, onDownOf, orderMatters, overlapOrder, priorityLink, type Policy, type Priorities } from './model'
import { overlapOrderText, overlapText } from './preview'
import { VpnState } from './shared'

const MAX_CHIPS = 4

/** Destinations as mono chips; apps marked as such; long lists end in "+N". */
const MatchCell = memo(function MatchCell({ p }: { p: Policy }) {
  const { t } = usePreferences()
  const items = [
    ...(p.domains ?? []).map((v) => ({ v, app: false })),
    ...(p.ip_ranges ?? []).map((v) => ({ v, app: false })),
    ...(p.apps ?? []).map((v) => ({ v, app: true })),
  ]
  const shown = items.slice(0, MAX_CHIPS)
  const rest = items.length - shown.length
  return (
    <ul className="r-chips" aria-label={t('rules.list.matchLabel', { count: items.length })}>
      {shown.map(({ v, app }) => (
        <li key={`${app}:${v}`} className={`r-chip r-chip--static ${app ? 'is-app' : ''}`}>
          {app && <span className="r-chip__tag">{t('rules.list.app')}</span>}
          {v}
        </li>
      ))}
      {rest > 0 && <li className="r-chips__more">{t('rules.list.more', { count: rest })}</li>}
    </ul>
  )
})

interface ListProps {
  policies: readonly Policy[]
  /** Each profile's own priority (absent = by position). */
  priorities: Priorities
  onEdit: (p: Policy) => void
  onDelete: (p: Policy) => void
}

export function PolicyList({ policies, priorities, onEdit, onDelete }: ListProps) {
  const { t } = usePreferences()
  const matters = useMemo(() => orderMatters(findOverlaps(policies)), [policies])
  const { mutate: toggle, isPending, variables } = useTogglePolicy()
  const columns = useMemo<Column<Policy>[]>(
    () => [
      { id: 'name', header: t('rules.list.name'), cell: (p) => p.name },
      { id: 'match', header: t('rules.list.match'), cell: (p) => <MatchCell p={p} /> },
      {
        id: 'via',
        header: t('rules.list.via'),
        cell: (p) => {
          const priority = priorities.get(p.via) ?? 0
          return (
            <span className="r-via">
              <VpnState name={p.via} />
              {matters.has(p.name) && <OrderHint via={p.via} priority={priority} />}
            </span>
          )
        },
      },
      {
        id: 'status',
        header: t('rules.list.status'),
        cell: (p) => (
          <div className="r-rule-switch">
            <Switch
              label={<span className="visually-hidden">{t('rules.action.toggleNamed', { name: p.name })}</span>}
              checked={!p.disabled}
              onText={t('rules.status.active')}
              offText={t('rules.status.paused')}
              pending={isPending && variables?.name === p.name}
              disabled={isPending}
              onChange={(active) => toggle({ name: p.name, disabled: !active })}
            />
          </div>
        ),
      },
      {
        id: 'down',
        header: t('rules.list.onDown'),
        cell: (p) => <span className="r-down">{t(`rules.onDown.${onDownOf(p)}.short` as MessageKey)}</span>,
      },
      {
        id: 'actions',
        header: t('rules.list.actions'),
        hideHeader: true,
        cell: (p) => (
          <span className="r-row-actions">
            <IconButton label={t('rules.action.editNamed', { name: p.name })} icon={<EditIcon />} onClick={() => onEdit(p)} />
            <IconButton label={t('rules.action.deleteNamed', { name: p.name })} icon={<TrashIcon />} onClick={() => onDelete(p)} />
          </span>
        ),
      },
    ],
    [t, onEdit, onDelete, toggle, isPending, variables, priorities, matters],
  )
  return <DataList caption={t('rules.list.caption')} columns={columns} rows={policies} rowKey={(p) => p.name} />
}

/** Rules that send the same destination to different VPNs, and which one wins. */
export function Overlaps({ policies, priorities }: { policies: readonly Policy[]; priorities: Priorities }) {
  const { t } = usePreferences()
  const overlaps = useMemo(() => findOverlaps(policies), [policies])
  if (overlaps.length === 0) return null
  const orderOverlap = overlaps.some((o) => o.kind !== 'nestedRange')
  return (
    <section className="r-overlaps" aria-labelledby="r-overlaps-title">
      <h3 id="r-overlaps-title">{t('rules.overlap.title', { count: overlaps.length })}</h3>
      <ul>
        {overlaps.map((o) => {
          const by = overlapOrder(o, priorities)
          return (
            <li key={`${o.kind}|${o.value}|${o.winner.policy}|${o.other.policy}`}>
              {overlapText(o, t)}
              {by && (
                <>
                  {' '}
                  <span className="r-overlaps__by">{overlapOrderText(o, by, priorities, t)}</span>
                  {/* Decided by position: offer the number that lets the user choose (the other VPN's, so it can win). */}
                  {by !== 'priority' && (
                    <>
                      {' '}
                      <Link to={priorityLink(o.other.via)} aria-label={t('rules.overlap.changePriorityOf', { name: o.other.via })}>
                        {t('rules.overlap.changePriority')}
                      </Link>
                    </>
                  )}
                </>
              )}
            </li>
          )
        })}
      </ul>
      {orderOverlap && <p className="r-note">{t('rules.overlap.change')}</p>}
    </section>
  )
}

/** "Prioridad N" / "por posición" next to the VPN, linking to its priority field. */
function OrderHint({ via, priority }: { via: string; priority: number }) {
  const { t } = usePreferences()
  return (
    <Tooltip content={t('rules.order.tip')} describe>
      <Link to={priorityLink(via)} className={`r-order${priority > 0 ? ' is-set' : ''}`}>
        {priority > 0 ? t('rules.order.priority', { n: priority }) : t('rules.order.position')}
      </Link>
    </Tooltip>
  )
}
