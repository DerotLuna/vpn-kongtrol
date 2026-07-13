import { useState } from 'react'
import { detectOS, OS } from './hooks/useOS'
import Nav from './components/Nav'
import Hero from './components/Hero'
import Download from './components/Download'
import Guide from './components/Guide'
import Footer from './components/Footer'
import DownloadGate from './components/DownloadGate'

const SESSION_KEY = 'dkt' // download token — no la clave real

export default function App() {
  const [os, setOS] = useState<OS>(detectOS)

  // Token HMAC guardado en sessionStorage — válido 24h, nunca contiene la clave real
  const [downloadToken, setDownloadToken] = useState<string | null>(
    () => sessionStorage.getItem(SESSION_KEY)
  )

  // Archivo pendiente de descarga hasta que el usuario ingrese la clave
  const [pendingFile, setPendingFile] = useState<string | null>(null)
  const handleRequestKey = (filename: string) => {
    setPendingFile(filename)
  }

  const handleKeySuccess = (token: string) => {
    sessionStorage.setItem(SESSION_KEY, token)
    setDownloadToken(token)

    if (pendingFile) {
      // Dispara la descarga — URL lleva token, nunca la clave
      window.location.href = `/api/download?file=${pendingFile}&token=${encodeURIComponent(token)}`
      setPendingFile(null)
    }
  }

  return (
    <>
      <Nav os={os} setOS={setOS} />
      <main>
        <Hero os={os} downloadKey={downloadToken} onRequestKey={handleRequestKey} />
        <Download os={os} downloadKey={downloadToken} onRequestKey={handleRequestKey} />
        <Guide os={os} setOS={setOS} />
      </main>
      <Footer />

      {pendingFile && (
        <DownloadGate
          filename={pendingFile}
          onClose={() => setPendingFile(null)}
          onSuccess={handleKeySuccess}
        />
      )}
    </>
  )
}
