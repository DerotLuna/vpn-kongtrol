// Kongtrol UI — Tráfico › Recientes · v1.0.0 · 2026-11-19
//
// The destinations this computer connected to lately (host or IP, program,
// the tunnel it left through or the normal connection, when) from the
// daemon's in-memory list (data/recent.ts, ADR-0054). Each row offers
// "Hacer que esto salga por…": the "＋ Agregar" sheet on the rule step
// with that destination prefilled (ADR-0055). On by default (ADR-0054); when turned off the card says
// where to turn it on; where the OS does not allow it without admin rights,
// it says so. Hidden on a daemon without the endpoint.
import { useMemo } from 'react'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { destinationOf, useRecentConnections, type RecentConnection } from '../../data/recent'
import { formatRelative } from '../../data/traffic'
import { AddButton } from '../add/entries'
import { Card } from '../ui/Card'
import { DataList, type Column } from '../ui/DataList'
import { EmptyState, Skeleton } from '../ui/States'
import { SectionError } from './SectionError'

const Dim = () => <span className="muted">—</span>

export function RecentCard() {
  const { t, lang } = usePreferences()
  const recent = useRecentConnections()
  const data = recent.data

  const columns = useMemo<Column<RecentConnection>[]>(
    () => [
      { id: 'dest', header: t('traffic.recent.col.dest'), cell: (c) => <span className="mono">{destinationOf(c)}</span> },
      { id: 'program', header: t('traffic.recent.col.program'), cell: (c) => (c.process ? <span className="mono">{c.process}</span> : <Dim />) },
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
      {data && data.supported && data.enabled && (
        <DataList
          caption={t('traffic.recent.caption')}
          columns={columns}
          rows={data.connections ?? []}
          rowKey={(c) => `${c.remote_ip}|${c.port}|${c.process ?? ''}`}
          empty={<p className="muted">{t('traffic.recent.empty')}</p>}
        />
      )}
    </Card>
  )
}
