import { Lang } from '../../i18n'
import { GITHUB_RELEASES } from '../../links'
import { pathFor } from '../../site'
import AttachDemo from './AttachDemo'
import CodeBlock from '../ui/CodeBlock'

interface Props {
  lang: Lang
}

// The plain-language promise right under the terminal hero, plus the two
// ways in: the app (everyone) and the CLI (devs). Same product, same engine.
export default function Start({ lang }: Props) {
  const es = lang === 'es'
  const t = es
    ? {
      title: 'Todas tus VPN en un solo lugar.',
      sub: 'Conéctate como siempre; Kongtrol se encarga del resto.',
      lead: 'Conectas en tu app de VPN de siempre. Kongtrol detecta el túnel y se hace cargo: lo vigila, lo reconecta si cae, protege tu DNS y aplica tus reglas. Donde el proveedor lo permite, también abre la app y conecta por ti.',
      everyoneTitle: 'Para todos',
      everyoneLead: 'Sin terminal. Todo desde el ícono de la bandeja y el panel.',
      everyone: [
        'Descarga Kongtrol para tu sistema.',
        'Ábrelo: vive en la bandeja, junto al reloj.',
        'Agrega tu VPN desde el panel y listo.',
      ],
      everyoneCta: 'Descargar gratis',
      everyoneNote: 'Cerrar la ventana no desconecta: Kongtrol sigue en segundo plano.',
      devsTitle: 'Para devs',
      devsLead: 'La misma app, con una CLI que hace todo lo que hace el panel. Ideal para scripts.',
      devsCode: `$ kongtrol init           # detecta tus VPN y crea perfiles
$ kongtrol up             # arranca en segundo plano
$ kongtrol status --watch # estado en vivo`,
      devsCta: 'man kongtrol — guía completa',
    }
    : {
      title: 'All your VPNs in one place.',
      sub: 'Connect the way you always do; Kongtrol handles the rest.',
      lead: 'You connect in the VPN app you already use. Kongtrol detects the tunnel and takes over: it watches it, reconnects it if it drops, protects your DNS and applies your rules. Where the provider allows it, it can also open the app and connect for you.',
      everyoneTitle: 'For everyone',
      everyoneLead: 'No terminal. Everything from the tray icon and the panel.',
      everyone: [
        'Download Kongtrol for your system.',
        'Open it: it lives in the tray, next to the clock.',
        'Add your VPN from the panel. Done.',
      ],
      everyoneCta: 'Download free',
      everyoneNote: 'Closing the window does not disconnect: Kongtrol keeps running in the background.',
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
        <div className="promise-head">
          <h2 className="promise-title">{t.title}</h2>
          <p className="promise-sub">{t.sub}</p>
          <p className="promise-lead">{t.lead}</p>
        </div>

        <AttachDemo lang={lang} />

        <div className="paths">
          <div className="path path-everyone">
            <h3>{t.everyoneTitle}</h3>
            <p className="path-lead">{t.everyoneLead}</p>
            <ol className="path-list">
              {t.everyone.map((s, i) => <li key={i}>{s}</li>)}
            </ol>
            <div className="path-actions">
              <a href={GITHUB_RELEASES} className="btn-download" target="_blank" rel="noreferrer">
                <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
                  <path d="M8 1a.5.5 0 0 1 .5.5v7.793l2.646-2.647a.5.5 0 0 1 .708.708l-3.5 3.5a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L7.5 9.293V1.5A.5.5 0 0 1 8 1ZM2.5 13a.5.5 0 0 0 0 1h11a.5.5 0 0 0 0-1h-11Z"/>
                </svg>
                {t.everyoneCta}
              </a>
            </div>
            <p className="path-note">{t.everyoneNote}</p>
          </div>

          <div className="path path-devs">
            <h3>{t.devsTitle}</h3>
            <p className="path-lead">{t.devsLead}</p>
            <CodeBlock lang="bash">{t.devsCode}</CodeBlock>
            <div className="path-actions">
              <a href={pathFor('guia', lang)} className="btn-ghost">{t.devsCta}</a>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
