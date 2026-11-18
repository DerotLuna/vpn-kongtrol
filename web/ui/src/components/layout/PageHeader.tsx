// Kongtrol UI — page title block · v0.1.0 · 2026-11-18
//
// Every page starts with one h1 (focusable, so the shell can move focus to
// it after navigation: screen readers then hear the new page) and sets the
// tab title "<page> · Kongtrol".
import { useEffect, type ReactNode } from 'react'
import { usePreferences } from '../../app/preferences-context'

export function PageHeader({ title, lead, children }: { title: string; lead?: ReactNode; children?: ReactNode }) {
  const { t } = usePreferences()
  useEffect(() => {
    document.title = `${title} · ${t('app.name')}`
  }, [title, t])
  return (
    <header className="k-page-head">
      <h1 tabIndex={-1}>{title}</h1>
      {lead && <p className="k-page-head__lead">{lead}</p>}
      {children}
    </header>
  )
}
