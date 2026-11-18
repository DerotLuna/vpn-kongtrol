import { Lang } from '../../i18n'
import { pathFor } from '../../site'
import { useDetectedOS, useIsMobile } from '../../hooks/useVisitor'
import { downloadHref, downloadLabel, DownloadIcon } from '../ui/downloadCta'
import SendToComputer from '../ui/SendToComputer'
import AttachDemo from './AttachDemo'
import CodeBlock from '../ui/CodeBlock'

interface Props {
  lang: Lang
}

// Right under the hero (which carries the h1 and the promise): the attach
// demo first — the clearest explanation for non-technical visitors — then the
// two ways in: the app (everyone) and the CLI (devs). Same product, same engine.
export default function Start({ lang }: Props) {
  const detectedOS = useDetectedOS()
  const mobile = useIsMobile()
  const es = lang === 'es'
  const t = es
    ? {
      title: 'Cómo empezar',
      everyoneTitle: 'Para todos',
      everyoneLead: 'Un comando para empezar; después, todo desde el panel.',
      everyone: [
        'Descarga Kongtrol para tu sistema.',
        'Corre kongtrol init una vez: detecta tus VPN y te guía.',
        'Corre kongtrol dashboard: el panel se abre en tu navegador para conectar y ajustar todo.',
      ],
      everyoneNote: 'Cerrar el panel o la terminal no desconecta: Kongtrol sigue en segundo plano.',
      devsTitle: 'Para devs',
      devsLead: 'La misma app, con una CLI que hace todo lo que hace el panel. Ideal para scripts.',
      devsCode: `$ kongtrol init           # detecta tus VPN y crea perfiles
$ kongtrol up             # arranca en segundo plano
$ kongtrol status --watch # estado en vivo`,
      devsCta: 'man kongtrol — guía completa',
    }
    : {
      title: 'Getting started',
      everyoneTitle: 'For everyone',
      everyoneLead: 'One command to set up; then everything from the panel.',
      everyone: [
        'Download Kongtrol for your system.',
        'Run kongtrol init once: it detects your VPNs and guides you.',
        'Run kongtrol dashboard: the panel opens in your browser to connect and adjust everything.',
      ],
      everyoneNote: 'Closing the panel or the terminal does not disconnect: Kongtrol keeps running in the background.',
      devsTitle: 'For devs',
      devsLead: 'The same app, with a CLI that does everything the panel does. Built for scripts.',
      devsCode: `$ kongtrol init           # detects your VPNs, creates profiles
$ kongtrol up             # starts in the background
$ kongtrol status --watch # live status`,
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
            <ol className="path-list">
              {t.everyone.map((s, i) => <li key={i}>{s}</li>)}
            </ol>
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
            <p className="path-note">{t.everyoneNote}</p>
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
