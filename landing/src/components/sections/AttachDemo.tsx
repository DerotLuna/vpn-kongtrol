import { useRef } from 'react'
import { useInView } from '../../hooks/useInView'
import { Lang } from '../../i18n'
import './attachdemo.css'

const SHIELD = 'M12 2.5 4.5 5.5v6c0 4.6 3.1 8.6 7.5 10 4.4-1.4 7.5-5.4 7.5-10v-6L12 2.5Z'

interface Props {
  lang: Lang
}

// Attach mode in three beats: your VPN app connects → Kongtrol detects the
// tunnel → the tray icon turns green. Pure CSS loop (see attachdemo.css);
// under prefers-reduced-motion it renders the final state only.
// The moving parts are decorative (aria-hidden); the ordered captions carry
// the meaning for screen readers and for anyone who skips the animation.
export default function AttachDemo({ lang }: Props) {
  // the loop runs only while the demo is on screen (ADR-0031)
  const ref = useRef<HTMLElement>(null)
  const live = useInView(ref)
  const es = lang === 'es'
  const t = es
    ? {
      label: 'Modo adjunto: así se ve',
      app: 'Tu app de VPN',
      apps: 'NordVPN · Proton VPN · Surfshark · FortiClient',
      off: 'Desconectado',
      on: 'Conectado',
      connect: 'Conectar',
      detect: 'túnel detectado',
      watching: 'vigilando',
      trayStatus: '1 VPN conectada',
      trayOpen: 'Abrir panel',
      trayStop: 'Salir y desconectar todo',
      trayQuit: 'Cerrar ícono (las VPN siguen)',
      steps: [
        'Conectas en tu app de VPN, como siempre.',
        'Kongtrol detecta el túnel y lo toma a su cargo.',
        'El ícono de la bandeja se pone en verde: estado, reconexión, DNS y reglas.',
      ],
    }
    : {
      label: 'Attach mode, at a glance',
      app: 'Your VPN app',
      apps: 'NordVPN · Proton VPN · Surfshark · FortiClient',
      off: 'Disconnected',
      on: 'Connected',
      connect: 'Connect',
      detect: 'tunnel detected',
      watching: 'watching',
      trayStatus: '1 VPN connected',
      trayOpen: 'Open panel',
      trayStop: 'Quit and disconnect all',
      trayQuit: 'Close icon (VPNs keep running)',
      steps: [
        'You connect in your VPN app, as usual.',
        'Kongtrol detects the tunnel and takes over.',
        'The tray icon turns green: status, reconnect, DNS and rules.',
      ],
    }

  return (
    <figure ref={ref} className={`attach${live ? ' is-live' : ''}`} aria-label={t.label}>
      <div className="attach-stage" aria-hidden="true">
        {/* 1 — the provider's own app (generic window, no third-party marks) */}
        <div className="attach-panel attach-app">
          <div className="attach-app-bar">
            <span /><span /><span />
            <em>{t.app}</em>
          </div>
          <div className="attach-app-body">
            {/* off and on shields stacked: the loop cross-fades them (opacity
                only — a color animation would repaint on every frame) */}
            <span className="attach-shield">
              <svg className="sh-off" viewBox="0 0 24 24" width="30" height="30">
                <path d={SHIELD} fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
              </svg>
              <svg className="sh-on" viewBox="0 0 24 24" width="30" height="30">
                <path d={SHIELD} fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
                <path d="m8.5 12 2.4 2.4 4.6-4.8" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </span>
            <span className="attach-app-state">
              <b className="st-off">{t.off}</b>
              <b className="st-on">{t.on}</b>
            </span>
            {/* idle, pressed and done looks stacked, cross-faded the same way */}
            <span className="attach-app-btn">
              <span className="b-idle">{t.connect}</span>
              <span className="b-done">{t.connect}</span>
              <span className="b-press">{t.connect}</span>
            </span>
          </div>
          <small className="attach-app-list">{t.apps}</small>
        </div>

        <span className="attach-link l1"><i /></span>

        {/* 2 — Kongtrol picks the tunnel up */}
        <div className="attach-panel attach-detect">
          <div className="attach-radar">
            <span className="attach-contour" />
            <span className="ring r1" /><span className="ring r2" /><span className="ring r3" />
            {/* the line-art original, large (DESIGN.md Logo, ADR-0060) */}
            <img className="logo-mark" src={`${import.meta.env.BASE_URL}logo-kong.svg`} alt="" width="108" height="108" />
          </div>
          <code className="attach-detect-line">
            <span className="d-off">…</span>
            <span className="d-on">✔ {t.detect} · {t.watching}</span>
          </code>
        </div>

        <span className="attach-link l2"><i /></span>

        {/* 3 — the tray icon and its real menu labels */}
        <div className="attach-panel attach-tray">
          <div className="attach-menu">
            <span className="m-status"><i className="m-dot" />{t.trayStatus}</span>
            <span>{t.trayOpen}</span>
            <span className="m-sep" />
            <span>{t.trayStop}</span>
            <span>{t.trayQuit}</span>
          </div>
          <div className="attach-taskbar">
            <span className="tb-icon" />
            <span className="tb-icon" />
            {/* the real tray icon: the solid glyph with the status badge in
                its corner (assets.BadgeGeometry: r = 15 % of the icon, a gap
                ring of size/16); hollow grey while stopped, live green once
                the tunnel is detected */}
            <span className="tb-kong">
              <img className="logo-mark" src={`${import.meta.env.BASE_URL}logo-kong-solid.svg`} alt="" width="28" height="28" />
              <i className="tb-dot" />
            </span>
            <span className="tb-clock">09:41</span>
          </div>
        </div>
      </div>

      <figcaption>
        <ol className="attach-steps">
          {t.steps.map((s, i) => (
            <li key={i}><span className="attach-step-n">{i + 1}</span>{s}</li>
          ))}
        </ol>
      </figcaption>
    </figure>
  )
}
