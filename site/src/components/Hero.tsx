import { useEffect, useRef, useState } from 'react'
import { OS, OS_LABELS } from '../hooks/useOS'
import { Lang } from '../i18n'

interface Props {
  os: OS
  lang: Lang
  downloadKey: string | null
  onRequestKey: (filename: string) => void
}

const TERMINAL_LINES: Record<Lang, { type: string; text: string }[]> = {
  es: [
    { type: 'cmd', text: 'kongtrol up --group work' },
    { type: 'ok', text: '[+] office      conectado   10.10.0.5' },
    { type: 'ok', text: '[+] dev-server  conectado   185.x.x.x' },
    { type: 'ok', text: '[+] aws         conectado   172.31.4.7' },
    { type: 'gap', text: '' },
    { type: 'cmd', text: 'kongtrol status' },
    { type: 'tbl', text: 'office      ● connected    10.10.0.5   1h 23m' },
    { type: 'tbl', text: 'dev-server  ● connected    185.x.x.x   1h 23m' },
    { type: 'tbl', text: 'aws         ● connected    172.31.4.7  1h 23m' },
    { type: 'dim', text: 'Kill Switch: ON  ·  DNS Guard: ON' },
  ],
  en: [
    { type: 'cmd', text: 'kongtrol up --group work' },
    { type: 'ok', text: '[+] office      connected   10.10.0.5' },
    { type: 'ok', text: '[+] dev-server  connected   185.x.x.x' },
    { type: 'ok', text: '[+] aws         connected   172.31.4.7' },
    { type: 'gap', text: '' },
    { type: 'cmd', text: 'kongtrol status' },
    { type: 'tbl', text: 'office      ● connected    10.10.0.5   1h 23m' },
    { type: 'tbl', text: 'dev-server  ● connected    185.x.x.x   1h 23m' },
    { type: 'tbl', text: 'aws         ● connected    172.31.4.7  1h 23m' },
    { type: 'dim', text: 'Kill Switch: ON  ·  DNS Guard: ON' },
  ],
}

const FILENAME: Record<OS, string> = {
  windows: 'kongtrol-windows-amd64.exe',
  macos:   'kongtrol-darwin-arm64',
  linux:   'kongtrol-linux-amd64',
}

export default function Hero({ os, lang, downloadKey, onRequestKey }: Props) {
  const [visibleLines, setVisibleLines] = useState(0)
  const started = useRef(false)

  useEffect(() => {
    started.current = false
    setVisibleLines(0)
  }, [lang])

  useEffect(() => {
    if (started.current) return
    started.current = true
    let i = 0
    const lines = TERMINAL_LINES[lang]
    const tick = () => {
      if (i >= lines.length) return
      i++
      setVisibleLines(i)
      setTimeout(tick, lines[i - 1].type === 'cmd' ? 600 : 260)
    }
    setTimeout(tick, 800)
  }, [lang])

  const filename = FILENAME[os]
  const lines = TERMINAL_LINES[lang]
  const copy = lang === 'es'
    ? {
      eyebrow: 'Kongtrol CLI',
      title1: 'CONTROLA',
      title2: 'TODAS',
      title3: 'TUS VPNs',
      sub: 'Orquestación multi-VPN terminal-first con políticas flow-aware, health-check real, failover automático y observabilidad en vivo.',
      download: `Descargar para ${OS_LABELS[os]}`,
      guide: 'Ver guía →',
      detected: 'Sistema detectado:',
      otherPlatforms: 'ver otras plataformas',
      pill1: 'Single binary',
      pill2: 'Policy routing',
      pill3: 'Watchdog + failover',
    }
    : {
      eyebrow: 'Kongtrol CLI',
      title1: 'CONTROL',
      title2: 'ALL',
      title3: 'YOUR VPNs',
      sub: 'Terminal-first multi-VPN orchestration with flow-aware policy routing, active tunnel health checks, automatic failover, and live observability.',
      download: `Download for ${OS_LABELS[os]}`,
      guide: 'View guide →',
      detected: 'Detected system:',
      otherPlatforms: 'view other platforms',
      pill1: 'Single binary',
      pill2: 'Policy routing',
      pill3: 'Watchdog + failover',
    }

  const downloadUrl = downloadKey
    ? `/api/download?file=${filename}&token=${encodeURIComponent(downloadKey)}`
    : '#'

  const handleDownload = (e: React.MouseEvent) => {
    if (!downloadKey) {
      e.preventDefault()
      onRequestKey(filename)
    }
  }

  return (
    <section id="top" className="section" style={{ borderTop: 'none', paddingTop: 0 }}>
      <div className="container">
        <div className="hero">
          <div className="hero-text">
            <div className="hero-eyebrow fade-up">{copy.eyebrow}</div>

            <h1 className="hero-title fade-up fade-up-1">
              {copy.title1}<br />
              <span>{copy.title2}</span><br />
              {copy.title3}.
            </h1>

            <p className="hero-sub fade-up fade-up-2">
              {copy.sub}
            </p>

            <div className="hero-pills fade-up fade-up-2">
              <span className="hero-pill">{copy.pill1}</span>
              <span className="hero-pill">{copy.pill2}</span>
              <span className="hero-pill">{copy.pill3}</span>
            </div>

            <div className="hero-cta fade-up fade-up-3">
              <a href={downloadUrl} className="btn-download" onClick={handleDownload} download>
                <svg width="15" height="15" viewBox="0 0 15 15" fill="none">
                  <path d="M7.5 1v9M4 7l3.5 3.5L11 7M2 13h11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
                {copy.download}
              </a>
              <a href="#guia" className="btn-ghost">{copy.guide}</a>
            </div>

            <div className="hero-detected fade-up fade-up-4">
              {copy.detected} <span>{OS_LABELS[os]}</span> —{' '}
              <a href="#descargar">{copy.otherPlatforms}</a>
            </div>
          </div>

          <div className="hero-terminal-wrap fade-up fade-up-2">
            <div className="terminal">
              <div className="terminal-bar">
                <div className="terminal-dots">
                  <div className="terminal-dot red" />
                  <div className="terminal-dot yellow" />
                  <div className="terminal-dot green" />
                </div>
                <span className="terminal-title">kongtrol — bash</span>
              </div>
              <div className="terminal-body">
                {lines.slice(0, visibleLines).map((line, i) => {
                  if (line.type === 'gap') return <br key={i} />
                  return (
                    <span key={i} className="t-line">
                      {line.type === 'cmd' && (
                        <><span className="t-prompt">$ </span><span className="t-cmd">{line.text}</span></>
                      )}
                      {line.type === 'ok' && (
                        <span className="t-ok">{line.text}</span>
                      )}
                      {line.type === 'tbl' && (
                        <span className="t-muted">{line.text}</span>
                      )}
                      {line.type === 'dim' && (
                        <span className="t-dim">{line.text}</span>
                      )}
                      {'\n'}
                    </span>
                  )
                })}
                {visibleLines < lines.length && (
                  <span className="t-cursor" />
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
