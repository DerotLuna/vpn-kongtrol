// Setup guide — version 3.0.0 · 2026-11-19 (ADR-0061).
// One markup tree for both languages; the words come from
// content/guide.{es,en}.tsx. Bone page with two Ink emphasis blocks (the
// quickstart and "how it works"), per DESIGN.md and ADR-0060.
import { Suspense, useMemo, type ReactNode } from 'react'
import { OS } from '../../hooks/useOS'
import { Lang } from '../../i18n'
import { useActiveSection } from '../../hooks/useActiveSection'
import CodeBlock from '../ui/CodeBlock'
import OsTabs from '../ui/OsTabs'
import SectionJumpMenu from '../ui/SectionJumpMenu'
import { GUIDE_SECTIONS_EN, GUIDE_SECTIONS_ES } from '../../content/guideSections'
import guideES, { GuideStep } from '../../content/guide.es'
import guideEN from '../../content/guide.en'
import { cmdrefGroups } from '../../content/cmdref'
import './guide.css'

interface Props { os: OS; setOS: (os: OS) => void; lang: Lang }

/** Numbered steps: what to do, then "you'll see" (the sign it worked). */
function Steps({ items, see, lang }: { items: GuideStep[]; see: string; lang: Lang }) {
  return (
    <ol className="g-steps">
      {items.map((s, i) => (
        <li key={i}>
          <span className="g-n" aria-hidden="true">{i + 1}</span>
          <div className="g-step">
            <h3 className="g-step-t">{s.t}</h3>
            <p>{s.d}</p>
            {s.code && <CodeBlock ui={lang} lang="bash" wrap>{s.code}</CodeBlock>}
            <p className="g-see"><span className="g-see-k">{see}</span>{s.see}</p>
          </div>
        </li>
      ))}
    </ol>
  )
}

function Head({ num, children }: { num: string; children: ReactNode }) {
  return (
    <>
      <div className="guide-section-num">{num}</div>
      <h2 className="guide-section-title">{children}</h2>
    </>
  )
}

export default function Guide({ os, setOS, lang }: Props) {
  const t = (lang === 'es' ? guideES : guideEN)(os)
  const L = t.labels
  const SECTIONS = lang === 'es' ? GUIDE_SECTIONS_ES : GUIDE_SECTIONS_EN
  const num = (id: string) => SECTIONS.find(s => s.id === id)?.num ?? ''
  const [activeSection, setActiveSection] = useActiveSection(SECTIONS.map(s => s.id), '-20% 0px -70% 0px', 'quickstart')
  // The command reference (the largest block, independent of the OS) is its
  // own Suspense boundary, hydrated in a later, shorter task. Memoized so the
  // OS detected after hydration does not re-render it first (ADR-0031).
  const cmdref = useMemo(() => (
    <Suspense>
      {cmdrefGroups(lang).map((g) => (
        <div key={g.title} className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>{g.title}</th>
                <th>{L.does}</th>
              </tr>
            </thead>
            <tbody>
              {g.rows.map((r) => (
                <tr key={r.cmd}>
                  <td>{r.cmd}</td>
                  <td>{r.desc}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ))}
    </Suspense>
  // L.does is a plain string fixed per language
  // eslint-disable-next-line react-hooks/exhaustive-deps
  ), [lang])

  return (
    <section id="guia" className="guide-body">
      <div className="container">
        <div className="guide-layout">
          <aside className="guide-sidebar" aria-label={L.toc}>
            <div className="guide-nav-title">{L.toc}</div>
            <ul className="guide-nav-list">
              {SECTIONS.map(s => (
                <li key={s.id}>
                  <a href={`#${s.id}`} className={activeSection === s.id ? 'active' : ''} aria-current={activeSection === s.id ? 'location' : undefined}>
                    <span className="guide-num">{s.num}</span>{s.label}
                  </a>
                </li>
              ))}
            </ul>
          </aside>
          <div className="guide-content">
            {/* Mobile section nav (the sidebar is hidden on small screens) */}
            <SectionJumpMenu label={L.jump} value={activeSection} options={SECTIONS} onChange={id => {
              setActiveSection(id)
              document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
            }} />

            {/* 0 — Inicio rápido: Ink emphasis block #1 */}
            <div id="quickstart" className="guide-section g-qs tone-ink">
              <h2 className="g-badge">{t.qs.badge}</h2>
              <p className="g-qs-lead">{t.qs.lead}</p>
              <Steps items={t.qs.steps} see={L.see} lang={lang} />
              <p className="g-qs-cli">{t.qs.cli}</p>
            </div>

            {/* 1 — Cómo funciona: Ink emphasis block #2 */}
            <div id="how" className="guide-section">
              <Head num={num('how')}>{t.how.title}</Head>
              <p>{t.how.lead}</p>
              <div className="guide-flow tone-ink">
                <div className="guide-flow-col">
                  <p className="guide-flow-h">{t.how.apps}</p>
                  <p>{t.how.appsSub}</p>
                </div>
                <div className="guide-flow-arrow" aria-hidden="true">──▶</div>
                <div className="guide-flow-col">
                  <p className="guide-flow-h">kongtrol</p>
                  <p>{t.how.engine}</p>
                </div>
                <div className="guide-flow-arrow" aria-hidden="true">──▶</div>
                <div className="guide-flow-col guide-flow-vpns">
                  <p className="guide-flow-h">{t.how.vpns}</p>
                  <div className="guide-flow-vpn"><strong>FortiClient</strong><span>office</span></div>
                  <div className="guide-flow-vpn"><strong>WireGuard</strong><span>dev-server</span></div>
                  <div className="guide-flow-vpn"><strong>Proton VPN</strong><span>us-content</span></div>
                </div>
                <div className="guide-flow-footer">{t.how.foot}</div>
              </div>
            </div>

            {/* 2 — Antes de empezar */}
            <div id="prereqs" className="guide-section">
              <Head num={num('prereqs')}>{t.prereqs.title}</Head>
              <p>{t.prereqs.lead}</p>
              <div className="table-scroll">
                <table className="data-table stack g-prereqs">
                  <thead>
                    <tr><th>{t.prereqs.col1}</th><th>{t.prereqs.col2}</th></tr>
                  </thead>
                  <tbody>
                    {t.prereqs.rows.map((r, i) => (
                      <tr key={i}><td>{r.vpn}</td><td data-label={t.prereqs.col2}>{r.need}</td></tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <p>{t.prereqs.note}</p>
            </div>

            {/* 3 — Instalar */}
            <div id="install" className="guide-section">
              <Head num={num('install')}>{t.install.title}</Head>
              <OsTabs os={os} setOS={setOS} lang={lang} />
              <div className="os-panel active">
                <Steps items={t.install.steps} see={L.see} lang={lang} />
                <p>{t.install.scriptNote}</p>
                {os === 'windows' && <CodeBlock ui={lang} lang="powershell" wrap>{`> ${t.install.winScript}`}</CodeBlock>}
              </div>
              <div className="callout">{t.install.helper}</div>
            </div>

            {/* 4 — Tu primera VPN */}
            <div id="wizard" className="guide-section">
              <Head num={num('wizard')}>{t.wizard.title}</Head>
              <p>{t.wizard.lead}</p>
              <ul className="g-list">
                {t.wizard.ways.map((w, i) => <li key={i}>{w}</li>)}
              </ul>
              <div className="callout">{t.wizard.attach}</div>
              <h3 className="g-h3">{t.wizard.cliTitle}</h3>
              <p>{t.wizard.cli}</p>
              <ol className="g-list g-ol">
                {t.wizard.cliSteps.map((w, i) => <li key={i}>{w}</li>)}
              </ol>
              <CodeBlock ui={lang} lang="bash">{t.wizard.code}</CodeBlock>
              <p>{t.wizard.cliMore}</p>
              <p className="g-aside">{t.wizard.forti}</p>
            </div>

            {/* 5 — Conectar */}
            <div id="connect" className="guide-section">
              <Head num={num('connect')}>{t.connect.title}</Head>
              <p>{t.connect.lead}</p>
              <p>{t.connect.bg}</p>
              <p>{t.connect.stop}</p>
              <CodeBlock ui={lang} lang="bash">{t.connect.code}</CodeBlock>
              <p>{t.connect.waiting}</p>
              <p>{t.connect.drop}</p>
            </div>

            {/* 6 — Reglas */}
            <div id="policies" className="guide-section">
              <Head num={num('policies')}>{t.policies.title}</Head>
              <p>{t.policies.lead}</p>
              <CodeBlock ui={lang} lang="text">{t.policies.diagram}</CodeBlock>
              <p>{t.policies.panel}</p>
              <CodeBlock ui={lang} lang="bash">{t.policies.code}</CodeBlock>
              <div className="callout">{t.policies.down}</div>
              <p>{t.policies.yamlLead}</p>
              <CodeBlock ui={lang} lang="yaml">{t.policies.yaml}</CodeBlock>
              <p>{t.policies.match}</p>
              <p>{t.policies.defaultEgress}</p>
              <p>{t.policies.apply}</p>
            </div>

            {/* 7 — Grupos */}
            <div id="groups" className="guide-section">
              <Head num={num('groups')}>{t.groups.title}</Head>
              <p>{t.groups.lead}</p>
              <CodeBlock ui={lang} lang="yaml">{t.groups.yaml}</CodeBlock>
              <CodeBlock ui={lang} lang="bash">{t.groups.code}</CodeBlock>
              <p>{t.groups.note}</p>
            </div>

            {/* 8 — Diagnóstico */}
            <div id="doctor" className="guide-section">
              <Head num={num('doctor')}>{t.doctor.title}</Head>
              <p>{t.doctor.lead}</p>
              <CodeBlock ui={lang} lang="bash">{t.doctor.code}</CodeBlock>
              <p>{t.doctor.export}</p>
            </div>

            {/* 9 — El panel */}
            <div id="dashboard" className="guide-section">
              <Head num={num('dashboard')}>{t.dashboard.title}</Head>
              <p>{t.dashboard.lead}</p>
              <dl className="g-pages">
                {t.dashboard.rows.map((r, i) => (
                  <div key={i}><dt>{r.name}</dt><dd>{r.desc}</dd></div>
                ))}
              </dl>
              <p>{t.dashboard.port}</p>
            </div>

            {/* 10 — Día a día */}
            <div id="daily" className="guide-section">
              <Head num={num('daily')}>{t.daily.title}</Head>
              <div className="table-scroll">
                <table className="data-table">
                  <thead><tr><th>{L.cmd}</th><th>{L.does}</th></tr></thead>
                  <tbody>
                    {t.daily.rows.map((r) => <tr key={r.cmd}><td>{r.cmd}</td><td>{r.desc}</td></tr>)}
                  </tbody>
                </table>
              </div>
              <p>{t.daily.ks}</p>
              <CodeBlock ui={lang} lang="yaml">{t.daily.ksYaml}</CodeBlock>
              <p>{t.daily.sched}</p>
              <CodeBlock ui={lang} lang="yaml">{t.daily.schedYaml}</CodeBlock>
              <p>{t.daily.pass}</p>
              <h3 id="commands" className="g-h3 g-anchor">{t.daily.refTitle}</h3>
              <p>{t.daily.ref}</p>
              {cmdref}
            </div>

            {/* 11 — Si algo falla */}
            <div id="trouble" className="guide-section">
              <Head num={num('trouble')}>{t.trouble.title}</Head>
              <p>{t.trouble.lead}</p>
              <OsTabs os={os} setOS={setOS} lang={lang} />
              <div className="g-fixes">
                {t.trouble.items.map((it, i) => (
                  <div key={i} className="g-fix">
                    <h3 className="g-fix-q">{it.q}</h3>
                    <p><span className="g-see-k">{L.fix}</span>{it.fix}</p>
                    {it.code && <CodeBlock ui={lang} lang={it.code.startsWith('vpns:') ? 'yaml' : os === 'windows' ? 'powershell' : 'bash'} wrap>{it.code}</CodeBlock>}
                  </div>
                ))}
              </div>
              <p>{t.trouble.files}</p>
            </div>

            {/* 12 — Actualizar */}
            <div id="update" className="guide-section">
              <Head num={num('update')}>{t.update.title}</Head>
              <p>{t.update.lead}</p>
              <p>{t.update.how}</p>
              <p>{t.update.notice}</p>
              <CodeBlock ui={lang} lang="bash">{t.update.code}</CodeBlock>
              <p>{t.update.privacy}</p>
            </div>

            {/* 13 — Desinstalar */}
            <div id="uninstall" className="guide-section">
              <Head num={num('uninstall')}>{t.uninstall.title}</Head>
              <div className="callout">{t.uninstall.keep}</div>
              <p>{t.uninstall.how}</p>
              <CodeBlock ui={lang} lang="bash">{t.uninstall.code}</CodeBlock>
              <p>{t.uninstall.scriptLead}</p>
              <CodeBlock ui={lang} lang={os === 'windows' ? 'powershell' : 'bash'} wrap>{t.uninstall.script}</CodeBlock>
              <p>{t.uninstall.data}</p>
            </div>
            <div id="faq" className="guide-section">
              <Head num={num('faq')}>{t.faq.title}</Head>
              {t.faq.items.map((item) => (
                <details className="guide-more guide-faq" key={item.question}>
                  <summary>{item.question}</summary>
                  <p>{item.answer}</p>
                </details>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
