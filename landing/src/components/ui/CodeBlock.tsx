import { useState } from 'react'
import { Lang } from '../../i18n'

interface Props {
  ui: Lang // page language for the button text: same in the prerender and the client
  lang?: string // code language label
  children: string
}

export default function CodeBlock({ ui, lang = 'bash', children }: Props) {
  const [copied, setCopied] = useState(false)
  const isES = ui === 'es'

  const copy = () => {
    // Strip prompt symbols before copying
    const clean = children.replace(/^\$ /gm, '')
    navigator.clipboard.writeText(clean).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1800)
    })
  }

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
        <button type="button" className={`copy-btn${copied ? ' copied' : ''}`} onClick={copy} aria-live="polite">
          {copied ? (isES ? 'copiado ✓' : 'copied ✓') : (isES ? 'copiar' : 'copy')}
        </button>
      </div>
      <pre>{highlighted}</pre>
    </div>
  )
}
