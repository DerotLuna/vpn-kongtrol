import { useEffect, useState } from 'react'
import { Lang } from '../../i18n'
import { useActiveSection } from '../../hooks/useActiveSection'

export interface TmuxWindow {
  n: number
  name: string
  id: string
}

// each page passes its own sections as windows (pages/*/App.tsx, legal.tsx);
// names are plain words in the visitor's language, ids are the URL hashes
interface Props {
  lang: Lang
  session?: string
  windows: TmuxWindow[]
  showStatus?: boolean // security + tunnel counters (landing demo)
}

export default function TmuxBar({ lang, session = 'kongtrol', windows, showStatus = true }: Props) {
  const [active] = useActiveSection(windows.map(w => w.id), '-40% 0px -55% 0px')
  const [clock, setClock] = useState('')

  useEffect(() => {
    const tick = () => {
      const d = new Date()
      setClock(`${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`)
    }
    tick()
    const t = window.setInterval(tick, 20_000)
    return () => window.clearInterval(t)
  }, [])

  return (
    <div className="tmux-bar" role="navigation" aria-label={lang === 'es' ? 'secciones' : 'sections'}>
      <span className="tmux-session">[{session}]</span>
      <ul className="tmux-windows">
        {windows.map(w => (
          <li key={w.n}>
            <a href={`#${w.id}`} className={active === w.id ? 'active' : ''} aria-current={active === w.id ? 'true' : undefined}>
              {w.n}:{w.name}{active === w.id ? '*' : ''}
            </a>
          </li>
        ))}
      </ul>
      <span className="tmux-right" aria-hidden="true">
        {showStatus && (
          <>
            <span className="tmux-shield">⛨ armed</span>
            <span className="tmux-sep">│</span>
            <span>3↑</span>
            <span className="tmux-sep">│</span>
          </>
        )}
        <span>{clock}</span>
      </span>
    </div>
  )
}
