import { ReactNode, useRef } from 'react'
import { useActiveSection } from '../../hooks/useActiveSection'
import SectionJumpMenu from '../ui/SectionJumpMenu'

interface LegalSection {
  id: string
  num: string
  label: string // sidebar / jump menu
  tmux: string // short name in the tmux bar
  title: string
  body: ReactNode[]
  list?: ReactNode[]
  after?: ReactNode[]
  callout?: 'warn'
}

export interface LegalDocContent {
  label: string // e.g. "$ cat TERMS.md"
  title: string
  sub: string
  version: string
  updated: string // ISO date
  session: string // tmux session name
  summaryBadge: string
  summary: ReactNode
  toc: string
  goto: string
  versionLabel: string
  contactTitle: string
  contact: ReactNode
  sections: LegalSection[]
}

// Shared body for /terminos, /privacidad and /seguridad (and their /en/ twins).
export default function LegalDoc({ doc }: { doc: LegalDocContent }) {
  const { sections } = doc
  const [activeSection, setActiveSection] = useActiveSection(sections.map(s => s.id), '-20% 0px -70% 0px', sections[0].id)
  const contentRef = useRef<HTMLDivElement>(null)

  return (
    <section id="doc" style={{ borderTop: '1px solid var(--border)' }}>
      <div className="container">
        <div className="guide-intro">
          <span className="guide-intro-badge">{doc.summaryBadge}</span>
          <p style={{ marginBottom: 6 }}>{doc.summary}</p>
          <p className="section-sub mono" style={{ fontSize: '0.76rem' }}>
            {doc.versionLabel} {doc.version} · <time dateTime={doc.updated}>{doc.updated}</time>
          </p>
        </div>

        <div className="guide-layout">
          <aside className="guide-sidebar">
            <div className="guide-nav-title">{doc.toc}</div>
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
            <SectionJumpMenu
              label={doc.goto}
              value={activeSection}
              options={sections}
              onChange={id => {
                setActiveSection(id)
                document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
              }}
            />

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
                {s.list && <ul>{s.list.map((item, i) => <li key={i}>{item}</li>)}</ul>}
                {s.after?.map((p, i) => <p key={i}>{p}</p>)}
              </div>
            ))}

            <div className="callout" style={{ marginTop: 8 }}>
              <strong>{doc.contactTitle}:</strong> {doc.contact}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
