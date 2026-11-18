import { CSSProperties, KeyboardEvent, useRef } from 'react'
import { OS, OS_LABELS } from '../../hooks/useOS'
import { Lang } from '../../i18n'

const ORDER: OS[] = ['windows', 'macos', 'linux']

// Windows / macOS / Linux switcher for per-OS instructions.
//
// It picks one value that every OS-specific block on the page follows, so it
// is a radio group, not tabs (WAI-ARIA APG "Radio Group" pattern:
// https://www.w3.org/WAI/ARIA/apg/patterns/radio/): one tab stop, arrow keys
// move and select, Home/End jump to the ends.
export default function OsTabs({ os, setOS, lang, style }: { os: OS; setOS: (o: OS) => void; lang: Lang; style?: CSSProperties }) {
  const refs = useRef<(HTMLButtonElement | null)[]>([])
  const label = lang === 'es' ? 'Sistema operativo' : 'Operating system'

  const onKeyDown = (e: KeyboardEvent<HTMLButtonElement>, i: number) => {
    const last = ORDER.length - 1
    const next =
      e.key === 'ArrowRight' || e.key === 'ArrowDown' ? (i === last ? 0 : i + 1)
      : e.key === 'ArrowLeft' || e.key === 'ArrowUp' ? (i === 0 ? last : i - 1)
      : e.key === 'Home' ? 0
      : e.key === 'End' ? last
      : null
    if (next == null) return
    e.preventDefault()
    setOS(ORDER[next])
    refs.current[next]?.focus()
  }

  return (
    <div className="os-tabs" role="radiogroup" aria-label={label} style={style ?? { marginBottom: 16 }}>
      {ORDER.map((o, i) => (
        <button
          key={o}
          ref={el => { refs.current[i] = el }}
          type="button"
          role="radio"
          aria-checked={os === o}
          tabIndex={os === o ? 0 : -1}
          className={`os-tab${os === o ? ' active' : ''}`}
          onClick={() => setOS(o)}
          onKeyDown={e => onKeyDown(e, i)}
        >
          {OS_LABELS[o]}
        </button>
      ))}
    </div>
  )
}
