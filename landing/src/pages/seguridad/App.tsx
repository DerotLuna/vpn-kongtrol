import { Lang } from '../../i18n'
import LegalApp from '../legal'
import doc from '../../content/legal/seguridad'

export default function App({ lang }: { lang: Lang }) {
  return <LegalApp page="seguridad" lang={lang} doc={doc} />
}
