// Kongtrol UI kit — DataList (table on wide screens, cards on phones) · v0.2.0 · 2026-11-18
//
// A real <table> (headers announced per cell); under 640px each row becomes
// a stacked card whose cells show their column name (CSS, data-label), so
// there is never horizontal page scroll. Rows are memoized and keyed by a
// stable id: with the cache's structural sharing a live tick re-renders only
// the rows whose object changed. Numeric columns use tabular figures and a
// fixed alignment so changing numbers don't shift the layout.
// A row added after the first render (a new profile, rule, group) rises in
// once (data-new); the first render and live ticks never animate.
import { memo, useLayoutEffect, useRef, type ReactNode } from 'react'

export interface Column<T> {
  id: string
  header: string
  cell: (row: T) => ReactNode
  /** Right-aligned tabular numbers. */
  numeric?: boolean
  /** Hidden column header (e.g. an actions column), still announced. */
  hideHeader?: boolean
}

interface DataListProps<T> {
  caption: string
  columns: Column<T>[]
  rows: readonly T[]
  rowKey: (row: T) => string
  /** Shown instead of the table when rows is empty. */
  empty?: ReactNode
  /** Test hook: called whenever a row renders. */
  onRowRender?: (key: string) => void
}

interface RowProps<T> {
  row: T
  columns: Column<T>[]
  rowId: string
  onRowRender: ((key: string) => void) | undefined
}

function RowImpl<T>({ row, columns, rowId, onRowRender }: RowProps<T>) {
  onRowRender?.(rowId)
  return (
    <tr data-key={rowId}>
      {columns.map((c, i) =>
        i === 0 ? (
          <th key={c.id} scope="row" data-label={c.header} className={c.numeric ? 'num' : undefined}>
            {c.cell(row)}
          </th>
        ) : (
          <td key={c.id} data-label={c.hideHeader ? undefined : c.header} className={c.numeric ? 'num' : undefined}>
            {c.cell(row)}
          </td>
        ),
      )}
    </tr>
  )
}

// Same columns + same row object = skip. Columns must be stable (module
// constant or useMemo) for this to hold.
const Row = memo(RowImpl) as typeof RowImpl

export function DataList<T>({ caption, columns, rows, rowKey, empty, onRowRender }: DataListProps<T>) {
  // Keys on screen after the last commit; null until the first one. New rows
  // are marked in the DOM (no extra render, memoized rows stay untouched).
  const seen = useRef<ReadonlySet<string> | null>(null)
  const body = useRef<HTMLTableSectionElement>(null)
  useLayoutEffect(() => {
    const prev = seen.current
    if (prev && body.current) {
      for (const tr of Array.from(body.current.rows)) {
        const key = tr.dataset.key
        if (key === undefined || prev.has(key)) continue
        tr.dataset.new = ''
        tr.addEventListener('animationend', () => delete tr.dataset.new, { once: true })
      }
    }
    seen.current = new Set(rows.map(rowKey))
  })
  if (rows.length === 0 && empty) return <>{empty}</>
  return (
    <div className="k-datalist">
      <table>
        <caption className="visually-hidden">{caption}</caption>
        <thead>
          <tr>
            {columns.map((c) => (
              <th key={c.id} scope="col" className={c.numeric ? 'num' : undefined}>
                {c.hideHeader ? <span className="visually-hidden">{c.header}</span> : c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody ref={body}>
          {rows.map((r) => {
            const id = rowKey(r)
            return <Row key={id} row={r} columns={columns} rowId={id} onRowRender={onRowRender} />
          })}
        </tbody>
      </table>
    </div>
  )
}
