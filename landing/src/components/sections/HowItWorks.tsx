import { Lang } from '../../i18n'
import { pathFor } from '../../site'

interface Props {
  lang: Lang
}

// "Cómo empezar" — three plain steps that work for both paths. The numbers
// stay because the order matters; the CLI shows up only as an alternative.
export default function HowItWorks({ lang }: Props) {
  const copy = lang === 'es'
    ? {
      title: 'Cómo empezar',
      sub: 'Tres pasos. La primera vez toma unos minutos; después, nada.',
      steps: [
        {
          name: 'Descarga e instala',
          desc: 'Windows, macOS o Linux. Luego corre kongtrol service install una vez: pide permiso de administrador. Después no abres nada como administrador.',
          alt: '',
        },
        {
          name: 'Agrega tus VPN',
          desc: 'La primera vez, con kongtrol init: detecta tus VPN instaladas y te guía. Las contraseñas se guardan en el llavero de tu sistema, nunca en archivos.',
          alt: 'Después, más VPN desde el panel (kongtrol dashboard); se activan al reiniciar Kongtrol',
        },
        {
          name: 'Conéctate como siempre',
          desc: 'Usa tu app de VPN de siempre, o deja que Kongtrol la abra y conecte por ti donde el proveedor lo permite. Kongtrol se queda vigilando en segundo plano.',
          alt: 'O en la terminal: kongtrol up',
        },
      ],
      ctaLead: '¿Primera vez? La guía cubre cada sistema, cada VPN y los errores comunes, paso a paso.',
      cta: 'Abrir la guía',
      ctaMeta: 'Windows / macOS / Linux · ES / EN',
    }
    : {
      title: 'How to get started',
      sub: 'Three steps. The first time takes a few minutes; after that, nothing.',
      steps: [
        {
          name: 'Download and install',
          desc: 'Windows, macOS or Linux. Then run kongtrol service install once: it asks for administrator permission. After that you never run anything as administrator.',
          alt: '',
        },
        {
          name: 'Add your VPNs',
          desc: 'The first time, with kongtrol init: it detects your installed VPNs and guides you. Passwords are stored in your system keychain, never in files.',
          alt: 'Later, more VPNs from the panel (kongtrol dashboard); they start working after restarting Kongtrol',
        },
        {
          name: 'Connect as usual',
          desc: 'Use your usual VPN app, or let Kongtrol open it and connect for you where the provider allows it. Kongtrol keeps watch in the background.',
          alt: 'Or in the terminal: kongtrol up',
        },
      ],
      ctaLead: 'First time? The guide covers every OS, every VPN and the common errors, step by step.',
      cta: 'Open the guide',
      ctaMeta: 'Windows / macOS / Linux · ES / EN',
    }

  return (
    <section id="steps" className="section how-section">
      <div className="container">
        <h2 className="section-title">{copy.title}</h2>
        <p className="section-sub">{copy.sub}</p>

        <ol className="how-steps">
          {copy.steps.map((s, i) => (
            <li key={s.name} className="how-step reveal">
              <span className="how-n" aria-hidden="true">{i + 1}</span>
              <h3>{s.name}</h3>
              <p>{s.desc}</p>
              {s.alt && <p className="how-alt">{s.alt}</p>}
            </li>
          ))}
        </ol>

        <div className="how-cta reveal">
          <p>{copy.ctaLead}</p>
          <div className="how-cta-actions">
            <a href={pathFor('guia', lang)} className="btn-download">{copy.cta}</a>
            <span className="how-cta-meta">{copy.ctaMeta}</span>
          </div>
        </div>
      </div>
    </section>
  )
}
