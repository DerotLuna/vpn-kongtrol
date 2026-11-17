import { Lang } from '../../i18n'
import LegalApp from '../legal'
import doc from '../../content/legal/privacidad'

export default function App({ lang }: { lang: Lang }) {
  return <LegalApp page="privacidad" lang={lang} doc={doc} />
}
