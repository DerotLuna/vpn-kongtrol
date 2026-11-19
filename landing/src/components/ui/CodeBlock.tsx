import { HTMLAttributes, ReactNode, useState } from 'react'
import { Lang } from '../../i18n'

interface Props {
  ui: Lang // page language for the button text: same in the prerender and the client
  lang?: string // code language label
  /** replaces the language label in the header (e.g. a tablist) */
  head?: ReactNode
  /** attributes for the <pre> (e.g. role="tabpanel") */
  panel?: HTMLAttributes<HTMLPreElement> & { 'aria-labelledby'?: string }
  /** a new value re-mounts the <pre> and cross-fades it in */
  swapKey?: string
  /** long lines wrap on screen with a hanging indent instead of scrolling
      (one-liners); copy still gives the exact text */
  wrap?: boolean
  children: string
}

// "$ cmd   # note": the trailing comment of a command line. On narrow blocks
// it drops to its own dimmed line under the command (index.css .cmt-*), so
// the block never scrolls sideways; copy still gives the original text.
const TRAILING_COMMENT = /^(.*?\S)(\s{2,})(#.*)$/

export default function CodeBlock({ ui, lang = 'bash', head, panel, swapKey, wrap, children }: Props) {
  // idle → copied | failed → idle. A refused clipboard (no permission, plain
  // http, old browser) says so instead of failing silently (Apple HIG Feedback).
  const [state, setState] = useState<'idle' | 'copied' | 'failed'>('idle')
  // after the first click every label change fades in (index.css .copy-label.is-fb)
  const [used, setUsed] = useState(false)
  const isES = ui === 'es'

  const copy = () => {
    // Strip prompt symbols before copying
    // ($ shell, > PowerShell: the same two prompts highlighted below)
    const clean = children.replace(/^[$>] /gm, '')
    const done = (s: 'copied' | 'failed') => {
      setUsed(true)
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
  // (wrap: every line is its own block, so each one gets the hanging indent)
  const highlighted = children.split('\n').map((line, i) => {
    let cls: string | undefined
    let body: ReactNode = line
    if (line.startsWith('#')) {
      cls = 'comment'
    } else if (line.startsWith('$ ') || line.startsWith('> ')) {
      const m = TRAILING_COMMENT.exec(line.slice(2))
      body = (
        <>
          <span className="prompt">{line.slice(0, 2)}</span>
          {m ? <>{m[1]}<span className="cmt-pad">{m[2]}</span><span className="comment cmt-inline">{m[3]}</span></> : line.slice(2)}
        </>
      )
    } else if (line.startsWith('[+]') || line.startsWith('[✓]')) {
      cls = 'hi'
    }
    if (wrap) return <span key={i} className={cls ? `ln ${cls}` : 'ln'}>{body}</span>
    return <span key={i} className={cls}>{body}{'\n'}</span>
  })

  return (
    <div className="code-block">
      <div className="code-block-header">
        {head ?? <span className="code-block-lang">{lang}</span>}
        <button type="button" className={`copy-btn${state === 'idle' ? '' : ` ${state}`}`} onClick={copy} aria-live="polite">
          <span key={state} className={`copy-label${used ? ' is-fb' : ''}`}>{label}</span>
        </button>
      </div>
      <pre key={swapKey} {...panel} className={[swapKey && 'is-swapped', wrap && 'is-wrapped'].filter(Boolean).join(' ') || undefined}>{highlighted}</pre>
    </div>
  )
}
