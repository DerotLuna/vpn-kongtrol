// Kongtrol UI kit — Card · v0.1.0 · 2026-11-18
//
// Flat (DESIGN.md Flat-By-Default): surface + 1px hairline, 6px corners, no
// shadow. A card groups one task; never nest cards.
import { useId, type ReactNode } from 'react'

interface CardProps {
  title?: ReactNode
  /** Right side of the header: a status pill or one action. */
  aside?: ReactNode
  children: ReactNode
  className?: string
  busy?: boolean
}

export function Card({ title, aside, children, className = '', busy }: CardProps) {
  const id = useId()
  return (
    <section className={`k-card ${className}`.trim()} aria-labelledby={title ? id : undefined} aria-busy={busy || undefined}>
      {(title || aside) && (
        <header className="k-card__head">
          {title && (
            <h2 id={id} className="k-card__title">
              {title}
            </h2>
          )}
          {aside}
        </header>
      )}
      {children}
    </section>
  )
}
