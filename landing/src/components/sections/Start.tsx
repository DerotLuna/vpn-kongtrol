import { Lang } from '../../i18n'
import { pathFor } from '../../site'
import { useDetectedOS, useIsMobile } from '../../hooks/useVisitor'
import { downloadHref, downloadLabel, DownloadIcon } from '../ui/downloadCta'
import SendToComputer from '../ui/SendToComputer'
import AttachDemo from './AttachDemo'
import CodeBlock from '../ui/CodeBlock'
import './start.css'

interface Props {
  lang: Lang
}

// Right under the hero (which carries the h1 and the promise): the attach
// demo first — the clearest explanation for non-technical visitors — then the
// two ways in: the app (everyone) and the CLI (devs). Same product, same engine.
// "Para todos" says what the person does and what Kongtrol does, in plain
// words; every literal command lives in the devs column. "Cómo empezar"
// (HowItWorks) covers the first-time setup, so the two blocks don't repeat.
export default function Start({ lang }: Props) {
  const detectedOS = useDetectedOS()
  const mobile = useIsMobile()
  const es = lang === 'es'
  const t = es
    ? {
      title: 'Formas de usar Kongtrol',
      everyoneTitle: 'Para todos',
      everyoneLead: 'Tú haces lo de siempre. Kongtrol hace el resto.',
      youLabel: 'Tú',
      kongLabel: 'Kongtrol',
      everyone: [
        { you: 'Conectas tu VPN como siempre.', kong: 'La detecta, la vigila y la reconecta si se cae.' },
        { you: 'Eliges qué sitios van por cada VPN.', kong: 'Manda cada sitio por la VPN correcta.' },
        { you: 'Cierras el panel cuando quieras.', kong: 'Sigue trabajando en segundo plano.' },
      ],
      setupNote: 'La primera vez, todavía se configura escribiendo un par de comandos. Sin problema: la guía te dice cuáles, y Kongtrol te pregunta lo demás.',
      setupLink: 'Ver cómo, paso a paso',
      devsTitle: 'Para devs',
      devsLead: 'La misma app, con una CLI que hace todo lo que hace el panel. Ideal para scripts.',
      devsCode: `$ kongtrol service install # una vez (pide admin)
$ kongtrol init            # detecta tus VPN, crea perfiles
$ kongtrol up              # arranca en segundo plano
$ kongtrol dashboard       # abre el panel
$ kongtrol status --watch  # estado en vivo`,
      devsCta: 'man kongtrol — guía completa',
    }
    : {
      title: 'Ways to use Kongtrol',
      everyoneTitle: 'For everyone',
      everyoneLead: 'You do what you always do. Kongtrol does the rest.',
      youLabel: 'You',
      kongLabel: 'Kongtrol',
      everyone: [
        { you: 'Connect your VPN as usual.', kong: 'Detects it, watches it and reconnects it if it drops.' },
        { you: 'Choose which sites go through each VPN.', kong: 'Sends each site through the right VPN.' },
        { you: 'Close the panel whenever you like.', kong: 'Keeps working in the background.' },
      ],
      setupNote: 'For now, the first setup still means typing a couple of commands. No worries: the guide shows you which ones, and Kongtrol asks you the rest.',
      setupLink: 'See how, step by step',
      devsTitle: 'For devs',
      devsLead: 'The same app, with a CLI that does everything the panel does. Built for scripts.',
      devsCode: `$ kongtrol service install # once (asks for admin)
$ kongtrol init            # detects your VPNs, creates profiles
$ kongtrol up              # starts in the background
$ kongtrol dashboard       # opens the panel
$ kongtrol status --watch  # live status`,
      devsCta: 'man kongtrol — full guide',
    }

  return (
    <section id="start" className="section promise-section">
      <div className="container">
        {/* outline only: the demo below is the visible lead-in */}
        <h2 className="sr-only">{t.title}</h2>

        <AttachDemo lang={lang} />

        <div className="paths reveal">
          <div className="path path-everyone">
            <h3>{t.everyoneTitle}</h3>
            <p className="path-lead">{t.everyoneLead}</p>
            <ul className="path-pairs">
              {t.everyone.map(p => (
                <li key={p.you} className="path-pair">
                  <p className="path-you"><span className="path-who">{t.youLabel}</span>{p.you}</p>
                  <p className="path-kong"><span className="path-who">{t.kongLabel}</span>{p.kong}</p>
                </li>
              ))}
            </ul>
            <p className="path-setup">
              {t.setupNote}{' '}
              <a href={`${pathFor('guia', lang)}#quickstart`}>{t.setupLink}</a>
            </p>
            {/* same as the hero: scroll to the download section; phones get "send to my computer" */}
            {mobile ? (
              <SendToComputer lang={lang} className="path-stc" />
            ) : (
              <div className="path-actions">
                <a href={downloadHref('home', lang)} className="btn-download">
                  <DownloadIcon size={14} />
                  {downloadLabel(lang, detectedOS)}
                </a>
              </div>
            )}
          </div>

          <div className="path path-devs">
            <h3>{t.devsTitle}</h3>
            <p className="path-lead">{t.devsLead}</p>
            <CodeBlock ui={lang} lang="bash">{t.devsCode}</CodeBlock>
            <div className="path-actions">
              <a href={pathFor('guia', lang)} className="btn-ghost">{t.devsCta}</a>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
