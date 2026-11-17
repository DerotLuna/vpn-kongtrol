import { Lang } from '../../i18n'
import LegalApp from '../legal'
import doc from '../../content/legal/terminos'

export default function App({ lang }: { lang: Lang }) {
  return <LegalApp page="terminos" lang={lang} doc={doc} />
}
