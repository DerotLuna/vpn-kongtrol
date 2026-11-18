import { Suspense, useMemo, useRef } from 'react'
import { OS } from '../../hooks/useOS'
import { Lang } from '../../i18n'
import { useActiveSection } from '../../hooks/useActiveSection'
import CodeBlock from '../ui/CodeBlock'
import IC from '../ui/InlineCode'
import OsTabs from '../ui/OsTabs'
import SectionJumpMenu from '../ui/SectionJumpMenu'
import { GUIDE_SECTIONS_EN, GUIDE_SECTIONS_ES } from '../../content/guideSections'
import guideES from '../../content/guide.es'
import guideEN from '../../content/guide.en'
import { cmdrefGroups } from '../../content/cmdref'

interface Props { os: OS; setOS: (os: OS) => void; lang: Lang }

// One markup tree for both languages; the words come from content/guide.{es,en}.tsx.
export default function Guide({ os, setOS, lang }: Props) {
  const t = (lang === 'es' ? guideES : guideEN)(os)
  const GUIDE_SECTIONS = lang === 'es' ? GUIDE_SECTIONS_ES : GUIDE_SECTIONS_EN
  const [activeSection, setActiveSection] = useActiveSection(GUIDE_SECTIONS.map(s => s.id), '-20% 0px -70% 0px', 'prereqs')
  const contentRef = useRef<HTMLDivElement>(null)
  // The command reference (the largest block, independent of the OS) is its
  // own Suspense boundary, hydrated in a later, shorter task. Memoized so the
  // OS detected after hydration does not re-render it first (ADR-0031).
  const cmdCol = t.daily3 // same text for every OS; a new JSX element each render, so not a memo dep
  const cmdref = useMemo(() => (
    <Suspense>
      {cmdrefGroups(lang).map((g) => (
        <div key={g.title} className="table-scroll" style={{ marginTop: 12 }}>
          <table className="data-table">
            <thead>
              <tr>
                <th>{g.title}</th>
                <th>{cmdCol}</th>
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
  // eslint-disable-next-line react-hooks/exhaustive-deps
  ), [lang])

  return (
    <section id="guia" style={{ borderTop: '1px solid var(--border)' }}>
      <div className="container">
        {/* ── Cómo funciona (contexto arquitectónico) ── */}
        <div className="guide-intro">
          {t.intro1}
          <p>{t.intro2}</p>
          <div className="guide-flow">
            <div className="guide-flow-col">
              <p className="guide-flow-h">{t.intro3}</p>
              <p>{t.intro4}</p>
            </div>
            <div className="guide-flow-arrow" aria-hidden="true">──▶</div>
            <div className="guide-flow-col">
              <p className="guide-flow-h">kongtrol</p>
              <p>{t.intro5}</p>
            </div>
            <div className="guide-flow-arrow" aria-hidden="true">──▶</div>
            <div className="guide-flow-col guide-flow-vpns">
              <p className="guide-flow-h">{t.intro6}</p>
              <div className="guide-flow-vpn"><strong>FortiClient</strong><span>office</span></div>
              <div className="guide-flow-vpn"><strong>OpenVPN</strong><span>dev-server</span></div>
              <div className="guide-flow-vpn"><strong>ProtonVPN</strong><span>us-content</span></div>
            </div>
            <div className="guide-flow-footer">{t.intro7}</div>
          </div>
        </div>
        <div className="guide-layout">
          {/* ── Sidebar ── */}
          <aside className="guide-sidebar">
            <div className="guide-nav-title">{t.intro8}</div>
            <ul className="guide-nav-list">
              {GUIDE_SECTIONS.map(s => (
                    <li key={s.id}>
                      <a
                        href={`#${s.id}`}
                        className={activeSection === s.id ? 'active' : ''}
                      >
                        <span className="guide-num">{s.num}</span>{s.label}
                      </a>
                    </li>
                  ))}
            </ul>
          </aside>
          {/* ── Content ── */}
          <div className="guide-content" ref={contentRef}>
            {/* ── Mobile section nav (sidebar is hidden on small screens) ── */}
            <SectionJumpMenu label={t.intro9} value={activeSection} options={GUIDE_SECTIONS} onChange={id => {
                    setActiveSection(id)
                    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
                  }} />
            {/* → Inicio rápido */}
            <div id="quickstart" className="guide-section guide-quickstart">
              <div className="guide-quickstart-head">
                {t.quickstart1}
                <p>{t.quickstart2}</p>
              </div>
              <ol className="qs-steps">
                <li>
                  <span className="qs-n">1</span>
                  <div>
                    {t.quickstart3}
                    <p>{t.quickstart4}</p>
                  </div>
                </li>
                <li>
                  <span className="qs-n">2</span>
                  <div>
                    {t.quickstart5}
                    <p>{t.quickstart6}</p>
                  </div>
                </li>
                <li>
                  <span className="qs-n">3</span>
                  <div>
                    {t.quickstart7}
                    <p>{t.quickstart8}</p>
                    <CodeBlock ui={lang} lang="bash">{`$ kongtrol init
$ kongtrol status --watch
$ kongtrol dashboard`}</CodeBlock>
                  </div>
                </li>
              </ol>
            </div>
            {/* 01 — Prerequisitos */}
            <div id="prereqs" className="guide-section">
              <div className="guide-section-num">1</div>
              <h2 className="guide-section-title">{t.prereqs1}</h2>
              <p>{t.prereqs2}</p>
              <div className="table-scroll">
                <table className="data-table stack">
                  <thead>
                    <tr>
                      <th>VPN</th>
                      <th>{t.prereqs3}</th>
                      <th>{t.prereqs4}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td>FortiClient</td>
                      <td data-label={t.prereqs3}>FortiClient 6.4.x</td>
                      <td data-label={t.prereqs4}>{t.prereqs5}</td>
                    </tr>
                    <tr>
                      <td>OpenVPN</td>
                      <td data-label={t.prereqs3}>OpenVPN Community</td>
                      <td data-label={t.prereqs4}>{t.prereqs6}</td>
                    </tr>
                    <tr>
                      <td>WireGuard</td>
                      <td data-label={t.prereqs3}>wireguard-tools</td>
                      <td data-label={t.prereqs4}><IC c="wg --version" /></td>
                    </tr>
                    <tr>
                      <td>ProtonVPN</td>
                      <td data-label={t.prereqs3}>{t.prereqs7}</td>
                      <td data-label={t.prereqs4}>{t.prereqs8}</td>
                    </tr>
                    <tr>
                      <td>Cloudflare WARP</td>
                      <td data-label={t.prereqs3}>warp-cli</td>
                      <td data-label={t.prereqs4}><IC c="warp-cli --version" /></td>
                    </tr>
                  </tbody>
                </table>
              </div>
              {os === 'windows' && (
                <div className="callout">{t.prereqs9}</div>
              )}
              <div className="callout" style={{ marginTop: 12 }}>{t.prereqs10}</div>
              <div className="table-scroll" style={{ marginTop: 24 }}>
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>{t.prereqs11}</th>
                      <th>{t.prereqs12}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td>kongtrol init</td>
                      <td>{t.prereqs13}</td>
                    </tr>
                    <tr>
                      <td>kongtrol doctor</td>
                      <td>{t.prereqs14}</td>
                    </tr>
                    <tr>
                      <td>{t.prereqs15}</td>
                      <td>{t.prereqs16}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
            {/* 02 — Instalación */}
            <div id="install" className="guide-section">
              <div className="guide-section-num">2</div>
              <h2 className="guide-section-title">{t.install1}</h2>
              <OsTabs os={os} setOS={setOS} lang={lang} />
              {os === 'windows' && (
                <div className="os-panel active">
                  <p>{t.install2}</p>
                  <p>{t.install3}</p>
                  <CodeBlock ui={lang} lang="powershell">{t.install4}</CodeBlock>
                  <p>{t.install5}</p>
                  <CodeBlock ui={lang} lang="powershell">{t.install6}</CodeBlock>
                  <p>{t.install7}</p>
                  <CodeBlock ui={lang} lang="powershell">{`kongtrol --help`}</CodeBlock>
                  <div className="callout">{t.install8}</div>
                </div>
              )}
              {os === 'macos' && (
                <div className="os-panel active">
                  <CodeBlock ui={lang} lang="bash">{t.install9}</CodeBlock>
                  <div className="callout">{t.install10}</div>
                </div>
              )}
              {os === 'linux' && (
                <div className="os-panel active">
                  <CodeBlock ui={lang} lang="bash">{t.install11}</CodeBlock>
                </div>
              )}
              <p>{t.install12}</p>
            </div>
            {/* 03 — Wizard */}
            <div id="wizard" className="guide-section">
              <div className="guide-section-num">3</div>
              <h2 className="guide-section-title">{t.wizard1}</h2>
              <p>{t.wizard2}</p>
              <p style={{ marginTop: 8 }}>{t.wizard3}</p>
              <h3 style={{ marginBottom: 8, marginTop: 24 }}>{t.wizard4}</h3>
              <p>{t.wizard5}</p>
              <div className="table-scroll" style={{ marginTop: 12 }}>
                <table className="data-table stack">
                  <thead>
                    <tr>
                      <th>{t.wizard6}</th>
                      <th>{t.wizard7}</th>
                      <th>{t.wizard8}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td>FortiClient</td>
                      <td data-label={t.wizard7}>{t.wizard9}</td>
                      <td data-label={t.wizard8}>{t.wizard10}</td>
                    </tr>
                    <tr>
                      <td>OpenVPN</td>
                      <td data-label={t.wizard7}>{t.wizard11}</td>
                      <td data-label={t.wizard8}>{t.wizard12}</td>
                    </tr>
                    <tr>
                      <td>ProtonVPN</td>
                      <td data-label={t.wizard7}>{t.wizard13}</td>
                      <td data-label={t.wizard8}>{t.wizard14}</td>
                    </tr>
                    <tr>
                      <td>{t.wizard15}</td>
                      <td data-label={t.wizard7}>{t.wizard16}</td>
                      <td data-label={t.wizard8}>{t.wizard17}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div className="callout">{t.wizard18}</div>
              <h3 style={{ marginBottom: 8, marginTop: 28 }}>{t.wizard19}</h3>
              <CodeBlock ui={lang} lang="bash">{`$ kongtrol init`}</CodeBlock>
              <p>{t.wizard20}</p>
              <CodeBlock ui={lang} lang="text">{t.wizard21}</CodeBlock>
              <p>{t.wizard22}</p>
              <CodeBlock ui={lang} lang="text">{t.wizard23}</CodeBlock>
              <p>{t.wizard24}</p>
              <details className="guide-more">
                <summary><span className="disclosure-mark" aria-hidden="true" />{t.wizard25}</summary>
                <CodeBlock ui={lang} lang="text">{t.wizard26}</CodeBlock>
              </details>
              <p style={{ marginTop: 20 }}>{t.wizard27}</p>
              <details className="guide-more">
                <summary><span className="disclosure-mark" aria-hidden="true" />{t.wizard28}</summary>
                <CodeBlock ui={lang} lang="text">{t.wizard29}</CodeBlock>
              </details>
              <h3 style={{ marginBottom: 8, marginTop: 24 }}>{t.wizard30}</h3>
              <p>{t.wizard31}</p>
              <ul style={{ marginTop: 4, paddingLeft: 20, lineHeight: 1.8 }}>
                <li>{t.wizard32}</li>
                <li>{t.wizard33}</li>
              </ul>
              <CodeBlock ui={lang} lang="yaml">{`vpns:
  us-content:
    type: wireguard
    config: ~/.kongtrol/configs/proton-us.conf
    auth:
      method: certificate`}</CodeBlock>
              <div className="callout" style={{ marginTop: 12 }}>{t.wizard34}</div>
              <div className="callout">{t.wizard35}</div>
              <div className="callout" style={{ marginTop: 12 }}>{t.wizard36}</div>
              <p>{t.wizard37}</p>
              <details className="guide-more">
                <summary><span className="disclosure-mark" aria-hidden="true" />{t.wizard38}</summary>
                <CodeBlock ui={lang} lang="text">{t.wizard39}</CodeBlock>
              </details>
              <div className="callout">{t.wizard40}</div>
              <div className="callout" style={{ marginTop: 12 }}>{t.wizard41}</div>
            </div>
            {/* 04 — Grupos */}
            <div id="groups" className="guide-section">
              <div className="guide-section-num">4</div>
              <h2 className="guide-section-title">{t.groups1}</h2>
              <p>{t.groups2}</p>
              <CodeBlock ui={lang} lang="yaml">{`groups:
  work:
    profiles: [office, dev-server, aws]

  travel:
    profiles: [us-content]

  full:
    profiles: [office, dev-server, aws, us-content]`}</CodeBlock>
              <CodeBlock ui={lang} lang="bash">{t.groups3}</CodeBlock>
              <div className="callout">{t.groups4}</div>
              <p>{t.groups5}</p>
            </div>
            {/* 05 — Políticas de routing */}
            <div id="policies" className="guide-section">
              <div className="guide-section-num">5</div>
              <h2 className="guide-section-title">{t.policies1}</h2>
              <p>{t.policies2}</p>
              <CodeBlock ui={lang} lang="text">{t.policies3}</CodeBlock>
              <p>{t.policies4}</p>
              <ul style={{ marginBottom: 16, paddingLeft: 20 }}>
                <li>{t.policies5}</li>
                <li>{t.policies6}</li>
                <li>{t.policies7}</li>
                <li>{t.policies8}</li>
              </ul>
              <p>{t.policies9}</p>
              <CodeBlock ui={lang} lang="yaml">{t.policies10}</CodeBlock>
              <div className="callout">{t.policies11}</div>
              <div className="callout" style={{ marginTop: 12 }}>{t.policies12}</div>
              <div className="callout" style={{ marginTop: 12 }}>{t.policies13}</div>
              <CodeBlock ui={lang} lang="bash">{t.policies14}</CodeBlock>
              <div className="callout">{t.policies15}</div>
            </div>
            {/* 06 — Doctor */}
            <div id="doctor" className="guide-section">
              <div className="guide-section-num">6</div>
              <h2 className="guide-section-title">{t.doctor1}</h2>
              <p>{t.doctor2}</p>
              <CodeBlock ui={lang} lang="bash">{`$ kongtrol doctor`}</CodeBlock>
              <CodeBlock ui={lang} lang="text">{t.doctor3}</CodeBlock>
              <p>{t.doctor4}</p>
            </div>
            {/* 07 — Primera conexión */}
            <div id="connect" className="guide-section">
              <div className="guide-section-num">7</div>
              <h2 className="guide-section-title">{t.connect1}</h2>
              <p>{t.connect2}</p>
              <CodeBlock ui={lang} lang="bash">{t.connect3}</CodeBlock>
              <p>{t.connect4}</p>
              <CodeBlock ui={lang} lang="bash">{`$ kongtrol up office dev-server aws`}</CodeBlock>
              <p>{t.connect5}</p>
              <p>{t.connect6}</p>
              <CodeBlock ui={lang} lang="bash">{`$ kongtrol up office
$ kongtrol up us-content`}</CodeBlock>
              <p>{t.connect7}</p>
              <CodeBlock ui={lang} lang="bash">{t.connect8}</CodeBlock>
              <p>{t.connect9}</p>
              <CodeBlock ui={lang} lang="bash">{t.connect10}</CodeBlock>
            </div>
            {/* 08 — Dashboard */}
            <div id="dashboard" className="guide-section">
              <div className="guide-section-num">8</div>
              <h2 className="guide-section-title">{t.dashboard1}</h2>
              <CodeBlock ui={lang} lang="bash">{`$ kongtrol dashboard
# Dashboard running at http://127.0.0.1:9741`}</CodeBlock>
              <p>{t.dashboard2}</p>
              <div className="table-scroll">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>{t.dashboard3}</th>
                      <th>{t.dashboard4}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td>{t.dashName1}</td>
                      <td>{t.dashDesc1}</td>
                    </tr>
                    <tr>
                      <td>{t.dashName2}</td>
                      <td>{t.dashDesc2}</td>
                    </tr>
                    <tr>
                      <td>{t.dashName3}</td>
                      <td>{t.dashDesc3}</td>
                    </tr>
                    <tr>
                      <td>{t.dashName4}</td>
                      <td>{t.dashDesc4}</td>
                    </tr>
                    <tr>
                      <td>{t.dashName5}</td>
                      <td>{t.dashDesc5}</td>
                    </tr>
                    <tr>
                      <td>{t.dashName6}</td>
                      <td>{t.dashDesc6}</td>
                    </tr>
                    <tr>
                      <td>{t.dashName7}</td>
                      <td>{t.dashDesc7}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <CodeBlock ui={lang} lang="bash">{t.dashboard10}</CodeBlock>
              <div className="callout">{t.dashboard11}</div>
            </div>
            {/* 09 — Uso diario */}
            <div id="daily" className="guide-section">
              <div className="guide-section-num">9</div>
              <h2 className="guide-section-title">{t.daily1}</h2>
              <div className="table-scroll">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>{t.daily2}</th>
                      <th>{t.daily3}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr>
                      <td>kongtrol up --group work</td>
                      <td>{t.daily4}</td>
                    </tr>
                    <tr>
                      <td>kongtrol down --group work</td>
                      <td>{t.daily5}</td>
                    </tr>
                    <tr>
                      <td>kongtrol up us-content</td>
                      <td>Netflix, Hulu, Claude AI...</td>
                    </tr>
                    <tr>
                      <td>kongtrol down --all</td>
                      <td>{t.daily6}</td>
                    </tr>
                    <tr>
                      <td>kongtrol status</td>
                      <td>{t.daily7}</td>
                    </tr>
                    <tr>
                      <td>kongtrol status --watch</td>
                      <td>{t.daily8}</td>
                    </tr>
                    <tr>
                      <td>kongtrol policy show github.com</td>
                      <td>{t.daily9}</td>
                    </tr>
                    <tr>
                      <td>kongtrol ip</td>
                      <td>{t.daily10}</td>
                    </tr>
                    <tr>
                      <td>kongtrol dashboard</td>
                      <td>{t.daily11}</td>
                    </tr>
                    <tr>
                      <td>kongtrol doctor</td>
                      <td>{t.daily12}</td>
                    </tr>
                    <tr>
                      <td>kongtrol export</td>
                      <td>{t.daily13}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <p style={{ marginTop: 20 }}>{t.daily14}</p>
              <CodeBlock ui={lang} lang="yaml">{`monitor:
  scheduler:
    enabled: true
    interval: "1m"
    rules:
      - name: "work-hours"
        profiles: ["office", "aws"]
        weekdays: ["mon","tue","wed","thu","fri"]
        start: "09:00"
        end: "18:00"`}</CodeBlock>
              <p style={{ marginTop: 16 }}>{t.daily15}</p>
              <CodeBlock ui={lang} lang="yaml">{`vpns:
  office:
    kill_switch: true
  us-content:
    kill_switch: false`}</CodeBlock>
              <p style={{ marginTop: 24 }}>{t.daily16}</p>
              <CodeBlock ui={lang} lang="bash">{t.daily17}</CodeBlock>
              <h3 id="commands" style={{ marginTop: 32, scrollMarginTop: 'calc(var(--nav-h) + 24px)' }}>{t.cmdref1}</h3>
              <p>{t.cmdref2}</p>
              {cmdref}
            </div>
            {/* 10 — Troubleshooting */}
            <div id="trouble" className="guide-section">
              <div className="guide-section-num">10</div>
              <h2 className="guide-section-title">{t.trouble1}</h2>
              <p>{t.trouble2}</p>
              <OsTabs os={os} setOS={setOS} lang={lang} />
              {os === 'windows' && (
                <CodeBlock ui={lang} lang="bash">{`$ netsh interface ip set dns "Ethernet" dhcp`}</CodeBlock>
              )}
              {os === 'macos' && (
                <CodeBlock ui={lang} lang="bash">{`$ networksetup -setdnsservers Wi-Fi empty`}</CodeBlock>
              )}
              {os === 'linux' && (
                <CodeBlock ui={lang} lang="bash">{`$ sudo cp /etc/resolv.conf.kongtrol.bak /etc/resolv.conf`}</CodeBlock>
              )}
              <p style={{ marginTop: 32 }}>{t.trouble3}</p>
              <CodeBlock ui={lang} lang="bash">{t.trouble4}</CodeBlock>
              <p>{t.trouble5}</p>
              <OsTabs os={os} setOS={setOS} lang={lang} />
              {os === 'windows' && (
                <CodeBlock ui={lang} lang="bash">{t.trouble6}</CodeBlock>
              )}
              {os === 'macos' && (
                <CodeBlock ui={lang} lang="bash">{`$ sudo pfctl -d`}</CodeBlock>
              )}
              {os === 'linux' && (
                <CodeBlock ui={lang} lang="bash">{`$ sudo iptables -F OUTPUT`}</CodeBlock>
              )}
              <p style={{ marginTop: 32 }}>{t.trouble7}</p>
              <CodeBlock ui={lang} lang="bash">{t.trouble8}</CodeBlock>
              <p style={{ marginTop: 32 }}><strong>Cloudflare WARP — "not registered":</strong></p>
              <CodeBlock ui={lang} lang="bash">{`$ warp-cli register`}</CodeBlock>
              <p style={{ marginTop: 32 }}>{t.trouble9}</p>
              <CodeBlock ui={lang} lang="bash">{t.trouble10}</CodeBlock>
              <div className="callout" style={{ marginTop: 12 }}>{t.trouble11}</div>
              <p style={{ marginTop: 32 }}>{t.trouble12}</p>
              <CodeBlock ui={lang} lang="text">{t.trouble13}</CodeBlock>
            </div>
            {/* 11 — Updating */}
            <div id="update" className="guide-section">
              <div className="guide-section-num">11</div>
              <h2 className="guide-section-title">{t.update1}</h2>
              <p className="guide-since">{t.sinceV060}</p>
              <p>{t.update2}</p>
              <p>{t.update3}</p>
              <CodeBlock ui={lang} lang="bash">{t.update4}</CodeBlock>
              <p style={{ marginTop: 32 }}>{t.update5}</p>
              <CodeBlock ui={lang} lang="bash">{t.update6}</CodeBlock>
              <p>{t.update7}</p>
            </div>
            {/* 12 — Uninstalling */}
            <div id="uninstall" className="guide-section">
              <div className="guide-section-num">12</div>
              <h2 className="guide-section-title">{t.uninstall1}</h2>
              <p className="guide-since">{t.sinceV060}</p>
              <div className="callout">{t.uninstall2}</div>
              <p style={{ marginTop: 24 }}>{t.uninstall3}</p>
              <p style={{ marginTop: 24 }}>{t.uninstall4}</p>
              <CodeBlock ui={lang} lang="bash">{t.uninstall5}</CodeBlock>
              <p style={{ marginTop: 24 }}>{t.uninstall6}</p>
              <CodeBlock ui={lang} lang="bash">{t.uninstall7}</CodeBlock>
              <p style={{ marginTop: 24 }}>{t.uninstall8}</p>
              <CodeBlock ui={lang} lang="bash">{t.uninstall9}</CodeBlock>
              <p style={{ marginTop: 24 }}>{t.uninstall10}</p>
              <p>{t.uninstall11}</p>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
