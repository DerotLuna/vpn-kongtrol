// Kongtrol UI — Tráfico › Recientes · v1.1.0 · 2026-11-24
//
// The destinations this computer connected to lately (host or IP, program,
// the tunnel it left through or the normal connection, when) from the
// daemon's in-memory list (data/recent.ts, ADR-0054). Each row offers
// "Hacer que esto salga por…": the "＋ Agregar" sheet on the rule step
// with that destination prefilled (ADR-0055). On by default (ADR-0054); when turned off the card says
// where to turn it on; where the OS does not allow it without admin rights,
// it says so. Hidden on a daemon without the endpoint.
// ADR-0106: 10 rows, then "Ver 20 más"; search + "Salió por" filter the
// whole list; live updates never reorder what is on screen: new entries
// wait behind "N nuevas · Mostrar" (polite live region). Long names are
// clipped with the full text in a tooltip (title), never widening a column.
import { useMemo, useRef, useState } from 'react'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { destinationOf, useRecentConnections, type RecentConnection } from '../../data/recent'
import { formatRelative } from '../../data/traffic'
import { AddButton } from '../add/entries'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { DataList, type Column } from '../ui/DataList'
import { SelectField, TextField } from '../ui/Field'
import type { SelectOption } from '../ui/Select'
import { EmptyState, Skeleton } from '../ui/States'
import { SectionError } from './SectionError'
import { matchesRecent, RECENT_FIRST, RECENT_STEP, recentKey, useStableList, viaNames, viaProfile, type ViaFilter } from './recentView'

const Dim = () => <span className="muted">—</span>

/** A long value clipped to its column; the full text in the tooltip. */
const Clip = ({ text }: { text: string }) => (
  <span className="mono tr-recent__clip" title={text}>
    {text}
  </span>
)

export function RecentCard() {
  const { t, lang } = usePreferences()
  const recent = useRecentConnections()
  const data = recent.data
  const live = data && data.supported && data.enabled ? (data.connections ?? []) : undefined
  const { shown, fresh, merge } = useStableList(live, recentKey)

  const [search, setSearch] = useState('')
  const [via, setVia] = useState<ViaFilter>('all')
  const [count, setCount] = useState(RECENT_FIRST)
  const listRef = useRef<HTMLDivElement>(null)

  const columns = useMemo<Column<RecentConnection>[]>(
    () => [
      { id: 'dest', header: t('traffic.recent.col.dest'), cell: (c) => <Clip text={destinationOf(c)} /> },
      { id: 'program', header: t('traffic.recent.col.program'), cell: (c) => (c.process ? <Clip text={c.process} /> : <Dim />) },
      { id: 'via', header: t('traffic.recent.col.via'), cell: (c) => (c.via ? c.via : t('traffic.recent.direct')) },
      {
        id: 'seen',
        header: t('traffic.recent.col.seen'),
        cell: (c) => <time dateTime={c.last_seen}>{formatRelative(c.last_seen, lang)}</time>,
      },
      {
        id: 'action',
        header: t('add.thisVia'),
        hideHeader: true,
        cell: (c) => (
          <AddButton
            req={{ mode: 'rule', target: destinationOf(c) }}
            variant="text"
          >
            {t('add.thisVia')}
          </AddButton>
        ),
      },
    ],
    [t, lang],
  )

  const viaOptions = useMemo<SelectOption<ViaFilter>[]>(() => {
    const names = viaNames([...shown, ...fresh])
    // Keep the chosen profile listed even after it left the list.
    if (via.startsWith('p:') && !names.includes(via.slice(2))) names.push(via.slice(2))
    return [
      { value: 'all', label: t('traffic.recent.via.all') },
      { value: 'direct', label: t('traffic.recent.direct') },
      ...names.map((n) => ({ value: viaProfile(n), label: n })),
    ]
  }, [shown, fresh, via, t])

  const filtering = search.trim() !== '' || via !== 'all'
  const matches = shown.filter((c) => matchesRecent(c, search, via))
  const visible = matches.slice(0, count)
  const left = matches.length - visible.length
  const freshMatches = fresh.filter((c) => matchesRecent(c, search, via)).length

  // A new filter starts again from the first rows.
  const onSearch = (v: string) => {
    setSearch(v)
    setCount(RECENT_FIRST)
  }
  const onVia = (v: ViaFilter) => {
    setVia(v)
    setCount(RECENT_FIRST)
  }
  const clearFilters = () => {
    onSearch('')
    setVia('all')
  }
  // The control the user pressed goes away: keep keyboard focus on the
  // list without scrolling.
  const focusList = () => listRef.current?.focus({ preventScroll: true })
  const showFresh = () => {
    // New rows go on top; what was on screen stays on screen below them.
    setCount((n) => n + freshMatches)
    merge()
    focusList()
  }
  const showMore = () => {
    setCount((n) => n + RECENT_STEP)
    if (left <= RECENT_STEP) focusList()
  }

  // An older daemon has no list: nothing to show.
  if (data === null) return null

  return (
    <Card title={t('traffic.recent.title')} className="tr-recent" busy={recent.isFetching && !data}>
      <p className="muted">{t('traffic.recent.what')}</p>
      {recent.isPending && <Skeleton lines={3} label={t('traffic.recent.loading')} />}
      {recent.isError && !data && <SectionError error={recent.error} onRetry={() => void recent.refetch()} />}
      {data && !data.supported && <p>{t('traffic.recent.unsupported')}</p>}
      {data && data.supported && !data.enabled && (
        <EmptyState
          title={t('traffic.recent.off')}
          body={t('traffic.recent.off.fix')}
          action={
            <Link className="k-btn k-btn--outlined k-btn--md" to="/settings#privacy">
              <span className="k-btn__label">{t('traffic.recent.off.action')}</span>
            </Link>
          }
        />
      )}
      {live && shown.length === 0 && <p className="muted">{t('traffic.recent.empty')}</p>}
      {live && shown.length > 0 && (
        <>
          <div className="tr-recent__tools">
            <TextField
              type="search"
              label={t('traffic.recent.search')}
              value={search}
              onChange={(e) => onSearch(e.target.value)}
              autoComplete="off"
              spellCheck={false}
            />
            <SelectField label={t('traffic.recent.col.via')} value={via} onChange={onVia} options={viaOptions} />
          </div>
          <div role="status" className="tr-recent__fresh">
            {freshMatches > 0 && (
              <>
                <span>{t(freshMatches === 1 ? 'traffic.recent.fresh.one' : 'traffic.recent.fresh', { n: freshMatches })}</span>
                <Button variant="text" size="sm" onClick={showFresh}>
                  {t('traffic.recent.fresh.show')}
                </Button>
              </>
            )}
          </div>
          <div ref={listRef} tabIndex={-1} className="tr-recent__list" aria-label={t('traffic.recent.caption')}>
            {matches.length === 0 && filtering ? (
              <EmptyState
                title={t('traffic.recent.noMatch')}
                body={t('traffic.recent.noMatch.fix')}
                action={
                  <Button variant="outlined" onClick={clearFilters}>
                    {t('traffic.recent.noMatch.clear')}
                  </Button>
                }
              />
            ) : (
              <DataList caption={t('traffic.recent.caption')} columns={columns} rows={visible} rowKey={recentKey} />
            )}
          </div>
          {left > 0 && (
            <div className="tr-recent__more">
              <Button variant="outlined" onClick={showMore}>
                {t('traffic.recent.more', { n: Math.min(RECENT_STEP, left), left })}
              </Button>
            </div>
          )}
        </>
      )}
    </Card>
  )
}
