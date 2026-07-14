import { useState } from 'react'
import { detectOS, OS } from './hooks/useOS'
import { usePreferences } from './hooks/usePreferences'
import { useDownloadGate } from './hooks/useDownloadGate'
import Nav from './components/Nav'
import Hero from './components/Hero'
import Download from './components/Download'
import Guide from './components/Guide'
import Footer from './components/Footer'
import DownloadGate from './components/DownloadGate'

export default function App() {
  const [os, setOS] = useState<OS>(detectOS)
  const { lang, setLang, theme, setTheme } = usePreferences()
  const { downloadToken, pendingFile, requestKey, closeGate, handleKeySuccess } = useDownloadGate()

  return (
    <>
      <Nav
        os={os}
        setOS={setOS}
        lang={lang}
        setLang={setLang}
        theme={theme}
        setTheme={setTheme}
      />
      <main>
        <Hero os={os} lang={lang} downloadKey={downloadToken} onRequestKey={requestKey} />
        <Download os={os} lang={lang} downloadKey={downloadToken} onRequestKey={requestKey} />
        <Guide os={os} setOS={setOS} lang={lang} />
      </main>
      <Footer lang={lang} />

      {pendingFile && (
        <DownloadGate
          lang={lang}
          filename={pendingFile}
          onClose={closeGate}
          onSuccess={handleKeySuccess}
        />
      )}
    </>
  )
}
