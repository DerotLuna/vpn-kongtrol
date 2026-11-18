import { Link } from 'react-router'
import { usePreferences } from '../app/preferences-context'
import { PageHeader } from '../components/layout/PageHeader'

/** Unknown page path (old bookmark, typo): say so and offer the way home. */
export function NotFoundPage() {
  const { t } = usePreferences()
  return (
    <>
      <PageHeader title={t('page.notFound.title')} lead={t('page.notFound.body')} />
      <Link className="k-btn k-btn--filled k-btn--md" to="/">
        <span className="k-btn__label">{t('page.notFound.action')}</span>
      </Link>
    </>
  )
}
