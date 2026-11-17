import { Lang } from '../../i18n'

interface Props {
  lang: Lang
}

// Attach mode in three beats: your VPN app connects → Kongtrol detects the
// tunnel → the tray icon turns green. Pure CSS loop (see .attach in
// index.css); under prefers-reduced-motion it renders the final state only.
// The moving parts are decorative (aria-hidden); the ordered captions carry
// the meaning for screen readers and for anyone who skips the animation.
export default function AttachDemo({ lang }: Props) {
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
    <figure className="attach" aria-label={t.label}>
      <div className="attach-stage" aria-hidden="true">
        {/* 1 — the provider's own app (generic window, no third-party marks) */}
        <div className="attach-panel attach-app">
          <div className="attach-app-bar">
            <span /><span /><span />
            <em>{t.app}</em>
          </div>
          <div className="attach-app-body">
            <svg className="attach-shield" viewBox="0 0 24 24" width="30" height="30">
              <path d="M12 2.5 4.5 5.5v6c0 4.6 3.1 8.6 7.5 10 4.4-1.4 7.5-5.4 7.5-10v-6L12 2.5Z" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
              <path className="attach-shield-check" d="m8.5 12 2.4 2.4 4.6-4.8" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
            <span className="attach-app-state">
              <b className="st-off">{t.off}</b>
              <b className="st-on">{t.on}</b>
            </span>
            <span className="attach-app-btn">{t.connect}</span>
          </div>
          <small className="attach-app-list">{t.apps}</small>
        </div>

        <span className="attach-link l1"><i /></span>

        {/* 2 — Kongtrol picks the tunnel up */}
        <div className="attach-panel attach-detect">
          <div className="attach-radar">
            <span className="ring r1" /><span className="ring r2" /><span className="ring r3" />
            <img src={`${import.meta.env.BASE_URL}logo-kong.svg`} alt="" width="44" height="44" />
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
            <span className="tb-kong">
              <img src={`${import.meta.env.BASE_URL}logo-kong.svg`} alt="" width="16" height="16" />
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
