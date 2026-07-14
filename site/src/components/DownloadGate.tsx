import { useRef, useState } from 'react'
import { Lang } from '../i18n'

interface Props {
  lang: Lang
  filename: string
  onClose: () => void
  onSuccess: (token: string) => void
}

export default function DownloadGate({ lang, filename, onClose, onSuccess }: Props) {
  const copy = lang === 'es'
    ? {
      badKey: 'Clave incorrecta',
      connError: 'Error de conexión',
      close: 'Cerrar',
      access: 'Acceso requerido',
      prompt: 'Ingresa la clave para descargar',
      placeholder: 'Clave de descarga',
      checking: 'Verificando…',
      download: 'Descargar',
    }
    : {
      badKey: 'Invalid key',
      connError: 'Connection error',
      close: 'Close',
      access: 'Access required',
      prompt: 'Enter the key to download',
      placeholder: 'Download key',
      checking: 'Checking…',
      download: 'Download',
    }

  const [key, setKey] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!key.trim()) return
    setLoading(true)
    setError('')

    try {
      // Clave en el body (POST) — nunca en la URL
      const res = await fetch('/api/auth', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key: key.trim() }),
      })

      const data = await res.json()

      if (res.ok && data.token) {
        onSuccess(data.token)
      } else {
        setError(data.error ?? copy.badKey)
        setKey('')
        inputRef.current?.focus()
      }
    } catch {
      setError(copy.connError)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="gate-overlay" onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div className="gate-modal">
        <button className="gate-close" onClick={onClose} aria-label={copy.close}>×</button>

        <div className="gate-icon">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
            <rect x="3" y="11" width="18" height="11" rx="2" stroke="currentColor" strokeWidth="1.5"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
          </svg>
        </div>

        <h2 className="gate-title">{copy.access}</h2>
        <p className="gate-sub">
          {copy.prompt}{' '}
          <code className="mono" style={{ fontSize: '0.78em' }}>{filename}</code>
        </p>

        <form onSubmit={submit} className="gate-form">
          <input
            ref={inputRef}
            type="password"
            className={`gate-input${error ? ' error' : ''}`}
            placeholder={copy.placeholder}
            value={key}
            onChange={e => { setKey(e.target.value); setError('') }}
            autoFocus
            autoComplete="off"
          />
          {error && <span className="gate-error">{error}</span>}
          <button
            type="submit"
            className="btn-download"
            disabled={loading || !key.trim()}
            style={{ width: '100%', justifyContent: 'center' }}
          >
            {loading ? copy.checking : copy.download}
          </button>
        </form>
      </div>
    </div>
  )
}
