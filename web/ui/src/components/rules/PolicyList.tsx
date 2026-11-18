// Reglas — saved rules in evaluation order, and where they overlap · v1.0.0 · 2026-11-18
import { memo, useMemo } from 'react'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import type { MessageKey } from '../../i18n/messages'
import { IconButton } from '../ui/Button'
import { DataList, type Column } from '../ui/DataList'
import { EditIcon, TrashIcon } from './icons'
import { findOverlaps, onDownOf, type Policy } from './model'
import { overlapText } from './preview'
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
  onEdit: (p: Policy) => void
  onDelete: (p: Policy) => void
}

export function PolicyList({ policies, onEdit, onDelete }: ListProps) {
  const { t } = usePreferences()
  const columns = useMemo<Column<Policy>[]>(
    () => [
      { id: 'name', header: t('rules.list.name'), cell: (p) => p.name },
      { id: 'match', header: t('rules.list.match'), cell: (p) => <MatchCell p={p} /> },
      { id: 'via', header: t('rules.list.via'), cell: (p) => <VpnState name={p.via} /> },
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
    [t, onEdit, onDelete],
  )
  return <DataList caption={t('rules.list.caption')} columns={columns} rows={policies} rowKey={(p) => p.name} />
}

/** Rules that send the same destination to different VPNs, and which one wins. */
export function Overlaps({ policies }: { policies: readonly Policy[] }) {
  const { t } = usePreferences()
  const overlaps = useMemo(() => findOverlaps(policies), [policies])
  if (overlaps.length === 0) return null
  const domainOverlap = overlaps.some((o) => o.kind !== 'nestedRange')
  return (
    <section className="r-overlaps" aria-labelledby="r-overlaps-title">
      <h3 id="r-overlaps-title">{t('rules.overlap.title', { count: overlaps.length })}</h3>
      <ul>
        {overlaps.map((o) => (
          <li key={`${o.kind}|${o.value}|${o.winner.policy}|${o.other.policy}`}>{overlapText(o, t)}</li>
        ))}
      </ul>
      {domainOverlap && (
        <p className="r-note">
          {t('rules.overlap.change')} <Link to="/profiles">{t('rules.editor.priority.link')}</Link>
        </p>
      )}
    </section>
  )
}
