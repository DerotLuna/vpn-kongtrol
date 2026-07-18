import { useEffect, useState } from 'react'
import { Lang } from '../i18n'

export interface TmuxWindow {
  n: number
  name: string
  id: string
}

// landing sections as tmux windows — names match the URL hashes
const HOME_WINDOWS: TmuxWindow[] = [
  { n: 0, name: 'motd', id: 'motd' },
  { n: 1, name: 'map', id: 'map' },
  { n: 2, name: 'check', id: 'check' },
  { n: 3, name: 'init', id: 'init' },
  { n: 4, name: 'install', id: 'install' },
]

interface Props {
  lang: Lang
  session?: string
  windows?: TmuxWindow[]
  showStatus?: boolean // security + tunnel counters (landing demo)
}

export default function TmuxBar({ lang, session = 'kongtrol', windows = HOME_WINDOWS, showStatus = true }: Props) {
  const [active, setActive] = useState(windows[0]?.id ?? '')
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

  useEffect(() => {
    const observer = new IntersectionObserver(
      entries => entries.forEach(e => { if (e.isIntersecting) setActive(e.target.id) }),
      { rootMargin: '-40% 0px -55% 0px' },
    )
    windows.forEach(w => {
      const el = document.getElementById(w.id)
      if (el) observer.observe(el)
    })
    return () => observer.disconnect()
  }, [windows])

  return (
    <div className="tmux-bar" role="navigation" aria-label={lang === 'es' ? 'secciones' : 'sections'}>
      <span className="tmux-session">[{session}]</span>
      <ul className="tmux-windows">
        {windows.map(w => (
          <li key={w.n}>
            <a href={`#${w.id}`} className={active === w.id ? 'active' : ''}>
              {w.n}:{w.name}{active === w.id ? '*' : ''}
            </a>
          </li>
        ))}
      </ul>
      <span className="tmux-right">
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
