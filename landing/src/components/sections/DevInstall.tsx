import { KeyboardEvent, useId, useRef, useState } from 'react'
import { useDetectedOS } from '../../hooks/useVisitor'
import { Lang } from '../../i18n'
import { INSTALL_CMD } from '../../links'
import CodeBlock from '../ui/CodeBlock'

type Shell = 'unix' | 'windows'
const ORDER: Shell[] = ['unix', 'windows']
const LABEL: Record<Shell, string> = { unix: 'macOS / Linux', windows: 'Windows' }

// The one-line install for the devs column. Two tabs in the code block's
// header (WAI-ARIA APG Tabs, automatic activation:
// https://www.w3.org/WAI/ARIA/apg/patterns/tabs/): one tab stop, arrows /
// Home / End move and select. The prerender shows macOS / Linux; after
// hydration a Windows visitor gets Windows until they pick. Each command is
// one line that wraps on screen with a hanging indent (CodeBlock wrap), so
// nothing scrolls sideways; copy gives the exact single line.
export default function DevInstall({ lang }: { lang: Lang }) {
  const detected = useDetectedOS()
  const [picked, setPicked] = useState<Shell | null>(null)
  const shell: Shell = picked ?? (detected === 'windows' ? 'windows' : 'unix')
  const refs = useRef<(HTMLButtonElement | null)[]>([])
  const id = useId()
  const tabId = (s: Shell) => `${id}-tab-${s}`
  const panelId = `${id}-panel`

  const onKeyDown = (e: KeyboardEvent<HTMLButtonElement>, i: number) => {
    const last = ORDER.length - 1
    const next =
      e.key === 'ArrowRight' ? (i === last ? 0 : i + 1)
      : e.key === 'ArrowLeft' ? (i === 0 ? last : i - 1)
      : e.key === 'Home' ? 0
      : e.key === 'End' ? last
      : null
    if (next == null) return
    e.preventDefault()
    setPicked(ORDER[next])
    refs.current[next]?.focus()
  }

  const tabs = (
    <div className="dev-tabs" role="tablist" aria-label={lang === 'es' ? 'Sistema operativo' : 'Operating system'}>
      {ORDER.map((s, i) => (
        <button
          key={s}
          ref={el => { refs.current[i] = el }}
          id={tabId(s)}
          type="button"
          role="tab"
          aria-selected={shell === s}
          aria-controls={panelId}
          tabIndex={shell === s ? 0 : -1}
          className="dev-tab"
          onClick={() => setPicked(s)}
          onKeyDown={e => onKeyDown(e, i)}
        >
          {LABEL[s]}
        </button>
      ))}
    </div>
  )

  return (
    <CodeBlock
      ui={lang}
      head={tabs}
      // a new key only once the visitor picks: the panel cross-fades on a
      // click, never on load (hydration may switch to Windows silently)
      swapKey={picked ?? undefined}
      wrap
      panel={{ id: panelId, role: 'tabpanel', 'aria-labelledby': tabId(shell), tabIndex: 0 }}
    >
      {`$ ${INSTALL_CMD[shell]}`}
    </CodeBlock>
  )
}
