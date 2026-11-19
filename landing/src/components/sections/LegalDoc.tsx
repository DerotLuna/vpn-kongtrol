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
  summary: ReactNode[] // short plain-language points, read before anything else
  toc: string
  goto: string
  versionLabel: string
  updatedLabel: string // "actualizada el" / "updated"
  contactTitle: string
  contact: ReactNode
  sections: LegalSection[]
}

// Version line shown under the page title (pages/legal.tsx).
export function LegalMeta({ doc }: { doc: LegalDocContent }) {
  return (
    <p className="legal-meta mono">
      {doc.versionLabel} {doc.version} · {doc.updatedLabel} <time dateTime={doc.updated}>{doc.updated}</time>
    </p>
  )
}

// Shared body for /terminos, /privacidad and /seguridad (and their /en/ twins):
// the summary as the page's one Ink block (ADR-0060), then the sections on Bone.
export default function LegalDoc({ doc }: { doc: LegalDocContent }) {
  const { sections } = doc
  const [activeSection, setActiveSection] = useActiveSection(sections.map(s => s.id), '-20% 0px -70% 0px', sections[0].id)
  const contentRef = useRef<HTMLDivElement>(null)

  return (
    <section id="doc" className="legal-doc">
      <div className="container">
        <aside className="legal-summary tone-ink" aria-labelledby="legal-summary-title">
          <h2 id="legal-summary-title" className="legal-summary-title">{doc.summaryBadge}</h2>
          <ul className="legal-summary-list">
            {doc.summary.map((p, i) => <li key={i}>{p}</li>)}
          </ul>
        </aside>

        <div className="guide-layout legal-layout">
          <nav className="guide-sidebar" aria-label={doc.toc}>
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
          </nav>

          <div className="guide-content legal-content" ref={contentRef}>
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
              <div key={s.id} id={s.id} className="guide-section legal-section">
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

            <div className="callout legal-contact">
              <strong>{doc.contactTitle}:</strong> {doc.contact}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
