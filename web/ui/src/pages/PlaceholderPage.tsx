// Kongtrol UI — sections not migrated yet · v0.1.0 · 2026-11-18
//
// Phase 1 routes every target section (ADR-0038) so navigation, deep links
// and the shell are final; each page lists what will live here (the parity
// rows in docs/dashboard-parity.md) and opens the classic page that has it
// today. Phase 2 replaces one placeholder per page.
import { usePreferences } from '../app/preferences-context'
import { PageHeader } from '../components/layout/PageHeader'
import { TabLinks, type TabLinkItem } from '../components/ui/Tabs'
import type { MessageKey } from '../i18n/messages'

export interface PlaceholderSpec {
  title: MessageKey
  lead: MessageKey
  items: MessageKey[]
  legacy: string
  tabs?: { label: MessageKey; items: { to: string; label: MessageKey }[] }
}

export function PlaceholderPage({ spec }: { spec: PlaceholderSpec }) {
  const { t } = usePreferences()
  const tabs: TabLinkItem[] | undefined = spec.tabs?.items.map((i) => ({ to: i.to, label: t(i.label) }))
  return (
    <>
      <PageHeader title={t(spec.title)} lead={t(spec.lead)}>
        {spec.tabs && tabs && <TabLinks label={t(spec.tabs.label)} items={tabs} />}
      </PageHeader>
      <section className="k-card k-soon" aria-labelledby="soon-title">
        <h2 id="soon-title" className="k-card__title">
          {t('page.soon')}
        </h2>
        <p className="muted">{t('page.soon.body')}</p>
        <ul className="k-soon__list">
          {spec.items.map((key) => (
            <li key={key}>{t(key)}</li>
          ))}
        </ul>
        <a className="k-btn k-btn--outlined k-btn--md" href={spec.legacy}>
          <span className="k-btn__label">{t('page.openClassic')}</span>
        </a>
      </section>
    </>
  )
}
