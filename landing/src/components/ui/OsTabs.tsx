import { OS } from '../../hooks/useOS'

// Windows / macOS / Linux switcher for per-OS instructions.
export default function OsTabs({ os, setOS }: { os: OS; setOS: (o: OS) => void }) {
  return (
    <div className="os-tabs" style={{ marginBottom: 16 }}>
      {(['windows', 'macos', 'linux'] as OS[]).map(o => (
        <button key={o} className={`os-tab${os === o ? ' active' : ''}`} onClick={() => setOS(o)}>
          {o === 'windows' ? 'Windows' : o === 'macos' ? 'macOS' : 'Linux'}
        </button>
      ))}
    </div>
  )
}
