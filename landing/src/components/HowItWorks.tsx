import { Lang } from '../i18n'
import CodeBlock from './CodeBlock'

interface Props {
  lang: Lang
}

export default function HowItWorks({ lang }: Props) {
  const copy = lang === 'es'
    ? {
      label: '$ kongtrol init',
      title: 'Tres comandos y estás operando.',
      sub: 'La configuración honesta toma 15–20 minutos la primera vez — la guía te acompaña en cada paso.',
      steps: [
        {
          n: '01',
          name: 'Registra tus perfiles',
          desc: 'El wizard detecta los clientes VPN instalados, guarda credenciales en el keychain del OS y arma tus políticas de ruteo. En español o inglés.',
          code: '$ kongtrol init',
        },
        {
          n: '02',
          name: 'Valida el stack',
          desc: 'Binarios, certificados, keychain, permisos, adaptadores registrados — todo verificado antes de conectar nada.',
          code: '$ kongtrol doctor',
        },
        {
          n: '03',
          name: 'Conecta y opera',
          desc: 'Levanta un grupo completo, mira el estado en vivo cada 2 segundos, abre el dashboard en localhost:9741.',
          code: `$ kongtrol up --group work
$ kongtrol status --watch
$ kongtrol dashboard`,
        },
      ],
      ctaLead: '¿Tediosa? Sí, la primera vez — por eso la guía cubre cada OS, cada adaptador y cada error común.',
      cta: 'Abrir la guía completa',
      ctaMeta: '11 secciones · Windows / macOS / Linux · ES / EN',
    }
    : {
      label: '$ kongtrol init',
      title: 'Three commands and you are running.',
      sub: 'Honest setup takes 15–20 minutes the first time — the guide walks you through every step.',
      steps: [
        {
          n: '01',
          name: 'Register your profiles',
          desc: 'The wizard detects installed VPN clients, stores credentials in the OS keychain, and builds your routing policies. In Spanish or English.',
          code: '$ kongtrol init',
        },
        {
          n: '02',
          name: 'Validate the stack',
          desc: 'Binaries, certificates, keychain, permissions, registered adapters — everything verified before connecting anything.',
          code: '$ kongtrol doctor',
        },
        {
          n: '03',
          name: 'Connect and operate',
          desc: 'Bring up a whole group, watch live status every 2 seconds, open the dashboard on localhost:9741.',
          code: `$ kongtrol up --group work
$ kongtrol status --watch
$ kongtrol dashboard`,
        },
      ],
      ctaLead: 'Tedious? Yes, the first time — the guide covers every OS, every adapter, and every common error.',
      cta: 'Open the full guide',
      ctaMeta: '11 sections · Windows / macOS / Linux · ES / EN',
    }

  return (
    <section id="init" className="section how-section">
      <div className="container">
        <div className="section-label cmd-label">{copy.label}</div>
        <h2 className="section-title">{copy.title}</h2>
        <p className="section-sub">{copy.sub}</p>

        <div className="how-steps">
          {copy.steps.map(s => (
            <article key={s.n} className="how-step reveal">
              <span className="how-n">{s.n}</span>
              <h3>{s.name}</h3>
              <p>{s.desc}</p>
              <CodeBlock lang="bash">{s.code}</CodeBlock>
            </article>
          ))}
        </div>

        <div className="how-cta reveal">
          <p>{copy.ctaLead}</p>
          <div className="how-cta-actions">
            <a href="/guia" className="btn-download">{copy.cta}</a>
            <span className="how-cta-meta">{copy.ctaMeta}</span>
          </div>
        </div>
      </div>
    </section>
  )
}
