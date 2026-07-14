import { useCallback, useState } from 'react'

const SESSION_KEY = 'dkt' // download token — no la clave real

export function useDownloadGate() {
  const [downloadToken, setDownloadToken] = useState<string | null>(() => {
    if (typeof window === 'undefined') return null
    return sessionStorage.getItem(SESSION_KEY)
  })
  const [pendingFile, setPendingFile] = useState<string | null>(null)

  const requestKey = useCallback((filename: string) => {
    setPendingFile(filename)
  }, [])

  const closeGate = useCallback(() => {
    setPendingFile(null)
  }, [])

  const handleKeySuccess = useCallback((token: string) => {
    sessionStorage.setItem(SESSION_KEY, token)
    setDownloadToken(token)

    setPendingFile(current => {
      if (current) {
        window.location.href = `/api/download?file=${current}&token=${encodeURIComponent(token)}`
      }
      return null
    })
  }, [])

  return {
    downloadToken,
    pendingFile,
    requestKey,
    closeGate,
    handleKeySuccess,
  }
}
