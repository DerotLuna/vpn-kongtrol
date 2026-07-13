import { useRef, useState } from 'react'

interface Props {
  filename: string
  onClose: () => void
  onSuccess: (token: string) => void
}

export default function DownloadGate({ filename, onClose, onSuccess }: Props) {
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
        setError(data.error ?? 'Clave incorrecta')
        setKey('')
        inputRef.current?.focus()
      }
    } catch {
      setError('Error de conexión')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="gate-overlay" onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div className="gate-modal">
        <button className="gate-close" onClick={onClose} aria-label="Cerrar">×</button>

        <div className="gate-icon">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
            <rect x="3" y="11" width="18" height="11" rx="2" stroke="currentColor" strokeWidth="1.5"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
          </svg>
        </div>

        <h2 className="gate-title">Acceso requerido</h2>
        <p className="gate-sub">
          Ingresa la clave para descargar{' '}
          <code className="mono" style={{ fontSize: '0.78em' }}>{filename}</code>
        </p>

        <form onSubmit={submit} className="gate-form">
          <input
            ref={inputRef}
            type="password"
            className={`gate-input${error ? ' error' : ''}`}
            placeholder="Clave de descarga"
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
            {loading ? 'Verificando…' : 'Descargar'}
          </button>
        </form>
      </div>
    </div>
  )
}
