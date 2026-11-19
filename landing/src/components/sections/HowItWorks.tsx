import { Lang } from '../../i18n'
import { pathFor } from '../../site'
import './start.css'

interface Props {
  lang: Lang
}

// "Cómo empezar" — the first-time setup in three plain steps. "Para todos"
// (Start) covers day-to-day use, so this block only covers getting set up.
// The numbers stay because the order matters; each step's command is a small
// secondary detail for whoever wants it, never the step itself.
export default function HowItWorks({ lang }: Props) {
  const copy = lang === 'es'
    ? {
      title: 'Cómo empezar',
      sub: 'Tres pasos. La primera vez toma unos minutos; después, nada.',
      cmdLabel: 'En la terminal:',
      steps: [
        {
          name: 'Descárgalo e instálalo',
          desc: 'Para Windows, macOS o Linux. Una sola vez aceptas el permiso de administrador; después no lo vuelves a necesitar.',
          cmd: 'kongtrol service install',
        },
        {
          name: 'Cuéntale qué VPN usas',
          desc: 'Un asistente encuentra las VPN que ya tienes y te hace unas preguntas. Tus contraseñas quedan en el llavero de tu sistema, nunca en archivos.',
          cmd: 'kongtrol init',
        },
        {
          name: 'Conéctate como siempre',
          desc: 'Usa tu app de VPN de siempre, o deja que Kongtrol la abra y conecte por ti donde el proveedor lo permite. En el panel ves todo de un vistazo.',
          cmd: 'kongtrol dashboard',
        },
      ],
      ctaLead: '¿Primera vez? La guía cubre cada sistema, cada VPN y los errores comunes, paso a paso.',
      cta: 'Abrir la guía',
      ctaMeta: 'Windows / macOS / Linux · ES / EN',
    }
    : {
      title: 'How to get started',
      sub: 'Three steps. The first time takes a few minutes; after that, nothing.',
      cmdLabel: 'In the terminal:',
      steps: [
        {
          name: 'Download and install it',
          desc: 'For Windows, macOS or Linux. You accept the administrator permission once; after that you never need it again.',
          cmd: 'kongtrol service install',
        },
        {
          name: 'Tell it which VPNs you use',
          desc: 'An assistant finds the VPNs you already have and asks you a few questions. Your passwords stay in your system keychain, never in files.',
          cmd: 'kongtrol init',
        },
        {
          name: 'Connect as usual',
          desc: 'Use your usual VPN app, or let Kongtrol open it and connect for you where the provider allows it. The panel shows everything at a glance.',
          cmd: 'kongtrol dashboard',
        },
      ],
      ctaLead: 'First time? The guide covers every OS, every VPN and the common errors, step by step.',
      cta: 'Open the guide',
      ctaMeta: 'Windows / macOS / Linux · ES / EN',
    }

  return (
    <section id="steps" className="section how-section tone-ink">
      <div className="container">
        <h2 className="section-title">{copy.title}</h2>
        <p className="section-sub">{copy.sub}</p>

        <ol className="how-steps">
          {copy.steps.map((s, i) => (
            <li key={s.name} className="how-step reveal">
              <span className="how-n" aria-hidden="true">{i + 1}</span>
              <h3>{s.name}</h3>
              <p>{s.desc}</p>
              <p className="how-cmd">{copy.cmdLabel} <code>{s.cmd}</code></p>
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
