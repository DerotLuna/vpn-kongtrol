import type { ReactNode } from 'react'
import { usePreferences } from '../../app/preferences-context'

interface Props {
  href: string
  children: ReactNode
  className?: string
}

/** Opens in a new tab, tells screen readers so, and never leaks the opener/referrer. */
export function ExternalLink({ href, children, className = '' }: Props) {
  const { t } = usePreferences()
  return (
    <a href={href} target="_blank" rel="noopener noreferrer" className={`ext ${className}`.trim()}>
      {children}
      <span className="visually-hidden"> {t('help.newTab')}</span>
    </a>
  )
}
