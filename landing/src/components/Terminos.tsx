import { useEffect, useRef, useState } from 'react'
import { Lang } from '../i18n'
import { GITHUB_REPO } from '../links'

interface Props { lang: Lang }

interface Section {
  id: string
  num: string
  label: string
  title: string
  body: string[]
  callout?: 'warn' | 'default'
}

const SECTIONS_ES: Section[] = [
  {
    id: 'que-es', num: '01', label: 'Qué es Kongtrol', title: 'Qué es Kongtrol',
    body: [`Kongtrol es software de código abierto, publicado bajo licencia MIT, que actúa como una capa de
    orquestación sobre clientes VPN que ya tienes instalados en tu equipo (OpenVPN, WireGuard, ProtonVPN, etc.).
    No es un proveedor de VPN, no opera servidores, y no tiene acceso a tu tráfico salvo el que corre localmente
    en tu propia máquina.`],
  },
  {
    id: 'garantia', num: '02', label: 'Sin garantía', title: 'Software "tal cual", sin garantía',
    callout: 'warn',
    body: [`Kongtrol se distribuye "AS IS" (tal cual), sin garantías de ningún tipo, expresas o implícitas,
    incluyendo — sin limitarse a — garantías de comerciabilidad, idoneidad para un propósito particular, o no
    infracción. En ningún caso los mantenedores o colaboradores del proyecto serán responsables por reclamos,
    daños u otras responsabilidades derivadas del uso del software, según los términos de la licencia MIT.`],
  },
  {
    id: 'beta', num: '03', label: 'Beta activa', title: 'Software en beta activa',
    body: [`El proyecto está en pruebas activas: no todos los adaptadores, escenarios de red, ni sistemas
    operativos están completamente validados. Puede fallar, comportarse de forma inesperada, o no bloquear una
    fuga de tráfico correctamente en ciertos escenarios (kill switch, DNS guard).`,
    `No uses Kongtrol como única capa de protección en escenarios donde una fuga de IP o DNS tenga consecuencias
    legales o de seguridad graves para ti.`],
  },
  {
    id: 'responsabilidad', num: '04', label: 'Tu responsabilidad', title: 'Tu responsabilidad',
    body: [`Eres el único responsable de:`],
  },
  {
    id: 'binarios', num: '05', label: 'Binarios no firmados', title: 'Binarios no firmados',
    body: [`Los binarios publicados en GitHub Releases no están firmados con un certificado de firma de código
    pagado. Es normal que Windows SmartScreen o macOS Gatekeeper muestren una advertencia al ejecutarlos — eso
    no implica que contengan malware.`,
    `Cada release incluye un checksums.txt (SHA256); verifica el hash antes de ejecutar un binario, o compila
    desde el código fuente si prefieres no confiar en un binario prebuild.`],
  },
  {
    id: 'privacidad', num: '06', label: 'Privacidad', title: 'Privacidad',
    body: [`Kongtrol no realiza llamadas externas, no envía telemetría, ni recolecta analíticas. El daemon, el
    CLI, la app de bandeja y el dashboard embebido solo hablan con tus clientes VPN, tu stack de red local y
    127.0.0.1. Ver docs/SECURITY.md en el repositorio para el detalle técnico completo.`],
  },
  {
    id: 'cambios', num: '07', label: 'Cambios', title: 'Cambios a estos términos',
    body: [`Estos términos pueden actualizarse a medida que el proyecto evoluciona. La versión vigente siempre
    vive en esta página y en el repositorio de GitHub.`],
  },
]

const SECTIONS_EN: Section[] = [
  {
    id: 'que-es', num: '01', label: 'What it is', title: 'What Kongtrol is',
    body: [`Kongtrol is open-source software, released under the MIT license, that acts as an orchestration
    layer on top of VPN clients already installed on your machine (OpenVPN, WireGuard, ProtonVPN, etc.). It is
    not a VPN provider, does not operate any servers, and has no access to your traffic beyond what runs locally
    on your own machine.`],
  },
  {
    id: 'garantia', num: '02', label: 'No warranty', title: 'Software provided "as is"',
    callout: 'warn',
    body: [`Kongtrol is distributed "AS IS", without warranty of any kind, express or implied, including but
    not limited to warranties of merchantability, fitness for a particular purpose, or non-infringement. In no
    event shall the project maintainers or contributors be liable for any claim, damages, or other liability
    arising from the use of the software, as set out in the MIT license terms.`],
  },
  {
    id: 'beta', num: '03', label: 'Active beta', title: 'Active beta software',
    body: [`The project is under active testing: not every adapter, network scenario, or operating system is
    fully validated yet. It can fail, behave unexpectedly, or not correctly block a traffic leak in certain
    scenarios (kill switch, DNS guard).`,
    `Do not rely on Kongtrol as your only layer of protection in scenarios where an IP or DNS leak would have
    serious legal or security consequences for you.`],
  },
  {
    id: 'responsabilidad', num: '04', label: 'Your responsibility', title: 'Your responsibility',
    body: [`You are solely responsible for:`],
  },
  {
    id: 'binarios', num: '05', label: 'Unsigned binaries', title: 'Unsigned binaries',
    body: [`Binaries published on GitHub Releases are not signed with a paid code-signing certificate. It is
    normal for Windows SmartScreen or macOS Gatekeeper to show a warning when running them — that does not mean
    they contain malware.`,
    `Every release ships a checksums.txt (SHA256); verify the hash before running a binary, or build from source
    if you would rather not trust a prebuilt one.`],
  },
  {
    id: 'privacidad', num: '06', label: 'Privacy', title: 'Privacy',
    body: [`Kongtrol makes no external calls, sends no telemetry, and collects no analytics. The daemon, CLI,
    tray app, and embedded dashboard only talk to your VPN clients, your local network stack, and 127.0.0.1. See
    docs/SECURITY.md in the repository for the full technical detail.`],
  },
  {
    id: 'cambios', num: '07', label: 'Changes', title: 'Changes to these terms',
    body: [`These terms may be updated as the project evolves. The current version always lives on this page and
    in the GitHub repository.`],
  },
]

export default function Terminos({ lang }: Props) {
  const sections = lang === 'es' ? SECTIONS_ES : SECTIONS_EN
  const [activeSection, setActiveSection] = useState(sections[0].id)
  const contentRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const observer = new IntersectionObserver(
      entries => {
        entries.forEach(e => {
          if (e.isIntersecting) setActiveSection(e.target.id)
        })
      },
      { rootMargin: '-20% 0px -70% 0px' }
    )
    sections.forEach(s => {
      const el = document.getElementById(s.id)
      if (el) observer.observe(el)
    })
    return () => observer.disconnect()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lang])

  const copy = lang === 'es'
    ? {
      updated: 'Última actualización: 30 de julio de 2026',
      toc: 'Contenido',
      goto: 'Ir a',
      summaryBadge: 'En resumen',
      summary: `Kongtrol es código abierto (MIT), se entrega "tal cual" sin garantía, y está en beta activa —
      úsalo con criterio en escenarios donde una fuga de red tenga consecuencias serias.`,
      respList: [
        'Cumplir los términos de servicio de cada proveedor VPN que conectes a través de Kongtrol.',
        'Cumplir las leyes aplicables en tu jurisdicción respecto al uso de VPNs y herramientas de enrutamiento de red.',
        'Verificar que el comportamiento de tus políticas de enrutamiento coincide con lo que esperas antes de depender de ellas.',
      ],
      respFoot: 'Kongtrol solo enruta tráfico según la configuración que tú defines — no valida legalidad ni idoneidad de esa configuración.',
      contact: 'Preguntas o problemas',
      contactText: 'Abre un issue en el repositorio de GitHub:',
    }
    : {
      updated: 'Last updated: July 30, 2026',
      toc: 'Contents',
      goto: 'Jump to',
      summaryBadge: 'TL;DR',
      summary: `Kongtrol is open source (MIT), shipped "as is" with no warranty, and under active beta — use
      good judgment in scenarios where a network leak would have serious consequences.`,
      respList: [
        'Complying with the terms of service of every VPN provider you connect through Kongtrol.',
        'Complying with applicable laws in your jurisdiction regarding VPN use and network routing tools.',
        'Verifying that your routing policies behave as you expect before relying on them.',
      ],
      respFoot: 'Kongtrol only routes traffic according to the configuration you define — it does not validate the legality or suitability of that configuration.',
      contact: 'Questions or issues',
      contactText: 'Open an issue on the GitHub repository:',
    }

  return (
    <section id="terminos" style={{ borderTop: '1px solid var(--border)' }}>
      <div className="container">
        <div className="guide-intro">
          <span className="guide-intro-badge">{copy.summaryBadge}</span>
          <p style={{ marginBottom: 6 }}>{copy.summary}</p>
          <p className="section-sub mono" style={{ fontSize: '0.76rem' }}>{copy.updated}</p>
        </div>

        <div className="guide-layout">
          <aside className="guide-sidebar">
            <div className="guide-nav-title">{copy.toc}</div>
            <ul className="guide-nav-list">
              {sections.map(s => (
                <li key={s.id}>
                  <a href={`#${s.id}`} className={activeSection === s.id ? 'active' : ''}>
                    <span className="guide-num">{s.num}</span>{s.label}
                  </a>
                </li>
              ))}
            </ul>
          </aside>

          <div className="guide-content" ref={contentRef}>
            <nav className="guide-mobile-nav">
              <span className="guide-mobile-nav-label">{copy.goto}</span>
              <select
                value={activeSection}
                onChange={e => {
                  const id = e.target.value
                  setActiveSection(id)
                  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
                }}
              >
                {sections.map(s => (
                  <option key={s.id} value={s.id}>{s.num} — {s.label}</option>
                ))}
              </select>
            </nav>

            {sections.map(s => (
              <div key={s.id} id={s.id} className="guide-section">
                <div className="guide-section-num">{s.num}</div>
                <h2 className="guide-section-title">{s.title}</h2>

                {s.callout === 'warn' ? (
                  <div className="callout warn">
                    {s.body.map((p, i) => <p key={i} style={{ margin: i === 0 ? 0 : '10px 0 0' }}>{p}</p>)}
                  </div>
                ) : (
                  s.body.map((p, i) => <p key={i}>{p}</p>)
                )}

                {s.id === 'responsabilidad' && (
                  <>
                    <ul>
                      {copy.respList.map((item, i) => <li key={i}>{item}</li>)}
                    </ul>
                    <p>{copy.respFoot}</p>
                  </>
                )}
              </div>
            ))}

            <div className="callout" style={{ marginTop: 8 }}>
              <strong>{copy.contact}:</strong> {copy.contactText}{' '}
              <a href={`${GITHUB_REPO}/issues`} target="_blank" rel="noreferrer">
                github.com/DerotLuna/vpn-kongtrol/issues
              </a>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
