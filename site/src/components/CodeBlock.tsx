import { useState } from 'react'

interface Props {
  lang?: string
  children: string
}

export default function CodeBlock({ lang = 'bash', children }: Props) {
  const [copied, setCopied] = useState(false)
  const isES = typeof document !== 'undefined'
    ? document.documentElement.lang.toLowerCase().startsWith('es')
    : false

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
        <button className={`copy-btn${copied ? ' copied' : ''}`} onClick={copy}>
          {copied ? (isES ? 'copiado ✓' : 'copied ✓') : (isES ? 'copiar' : 'copy')}
        </button>
      </div>
      <pre>{highlighted}</pre>
    </div>
  )
}
