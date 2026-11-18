import { useState } from 'react'
import { Lang } from '../../i18n'

interface Props {
  ui: Lang // page language for the button text: same in the prerender and the client
  lang?: string // code language label
  children: string
}

export default function CodeBlock({ ui, lang = 'bash', children }: Props) {
  // idle → copied | failed → idle. A refused clipboard (no permission, plain
  // http, old browser) says so instead of failing silently (Apple HIG Feedback).
  const [state, setState] = useState<'idle' | 'copied' | 'failed'>('idle')
  const isES = ui === 'es'

  const copy = () => {
    // Strip prompt symbols before copying
    const clean = children.replace(/^\$ /gm, '')
    const done = (s: 'copied' | 'failed') => {
      setState(s)
      setTimeout(() => setState('idle'), s === 'copied' ? 1800 : 3200)
    }
    if (!navigator.clipboard) { done('failed'); return }
    navigator.clipboard.writeText(clean).then(() => done('copied'), () => done('failed'))
  }
  const label = state === 'copied' ? (isES ? 'copiado ✓' : 'copied ✓')
    : state === 'failed' ? (isES ? 'no se pudo: selecciónalo y copia' : "couldn't copy: select it and copy")
    : (isES ? 'copiar' : 'copy')

  // Simple syntax highlight: lines starting with # are comments, $ is prompt
  const highlighted = children.split('\n').map((line, i) => {
    if (line.startsWith('#')) {
      return <span key={i} className="comment">{line}{'\n'}</span>
    }
    if (line.startsWith('$ ') || line.startsWith('> ')) {
      return (
        <span key={i}>
          <span className="prompt">{line.slice(0, 2)}</span>
          {line.slice(2)}{'\n'}
        </span>
      )
    }
    if (line.startsWith('[+]') || line.startsWith('[✓]')) {
      return <span key={i} className="hi">{line}{'\n'}</span>
    }
    return <span key={i}>{line}{'\n'}</span>
  })

  return (
    <div className="code-block">
      <div className="code-block-header">
        <span className="code-block-lang">{lang}</span>
        <button type="button" className={`copy-btn${state === 'idle' ? '' : ` ${state}`}`} onClick={copy} aria-live="polite">
          {label}
        </button>
      </div>
      <pre>{highlighted}</pre>
    </div>
  )
}
