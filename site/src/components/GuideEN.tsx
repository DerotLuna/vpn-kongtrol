import { useEffect, useRef, useState } from 'react'
import { OS } from '../hooks/useOS'
import CodeBlock from './CodeBlock'
import { GUIDE_SECTIONS_EN } from '../content/guideSections'

interface Props { os: OS; setOS: (os: OS) => void }

function OsTabs({ os, setOS }: { os: OS; setOS: (o: OS) => void }) {
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

const IC = ({ c }: { c: string }) => <code className="ic">{c}</code>

export default function GuideEN({ os, setOS }: Props) {
  const [activeSection, setActiveSection] = useState('prereqs')
  const contentRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const observer = new IntersectionObserver(
      entries => {
        entries.forEach(e => {
          if (e.isIntersecting) setActiveSection(e.target.id)
        })
      },
      { rootMargin: '-20% 0px -70% 0px' }
    )
    GUIDE_SECTIONS_EN.forEach(s => {
      const el = document.getElementById(s.id)
      if (el) observer.observe(el)
    })
    return () => observer.disconnect()
  }, [])

  return (
    <section id="guia" style={{ borderTop: '1px solid var(--border)' }}>
      <div className="container">
        <div className="guide-layout">
          <aside className="guide-sidebar">
            <div className="guide-nav-title">Contents</div>
            <ul className="guide-nav-list">
              {GUIDE_SECTIONS_EN.map(s => (
                <li key={s.id}>
                  <a href={`#${s.id}`} className={activeSection === s.id ? 'active' : ''}>
                    <span className="guide-num">{s.num}</span>{s.label}
                  </a>
                </li>
              ))}
            </ul>
          </aside>

          <div className="guide-content" ref={contentRef}>
            <div id="prereqs" className="guide-section">
              <div className="guide-section-num">01</div>
              <h2 className="guide-section-title">Prerequisites</h2>
              <p>Kongtrol orchestrates existing VPN clients. Install your VPN clients before running <IC c="kongtrol init" />.</p>
              <div className="callout">
                <strong>Important:</strong> close VPN GUI apps before <IC c="kongtrol up" /> to avoid conflicts with CLI control.
              </div>
            </div>

            <div id="install" className="guide-section">
              <div className="guide-section-num">02</div>
              <h2 className="guide-section-title">Installation</h2>
              <OsTabs os={os} setOS={setOS} />
              {os === 'windows' && (
                <CodeBlock lang="powershell">{`# Move binary to a stable path
New-Item -ItemType Directory -Force "C:\\tools"
Move-Item kongtrol.exe C:\\tools\\kongtrol.exe

# Add to PATH (run as Administrator)
[Environment]::SetEnvironmentVariable(
  "Path",
  $env:Path + ";C:\\tools",
  "Machine"
)

kongtrol --help`}</CodeBlock>
              )}
              {os !== 'windows' && (
                <CodeBlock lang="bash">{`# macOS/Linux
sudo mv kongtrol /usr/local/bin/
kongtrol --help`}</CodeBlock>
              )}
            </div>

            <div id="certs" className="guide-section">
              <div className="guide-section-num">03</div>
              <h2 className="guide-section-title">Connection files</h2>
              <p>If your VPNs already work today, you usually only need to provide current file paths to the wizard.</p>
              <p>Optional organization path: <IC c={os === 'windows' ? '%USERPROFILE%\\.kongtrol\\certs\\' : '~/.kongtrol/certs/'} /></p>
            </div>

            <div id="wizard" className="guide-section">
              <div className="guide-section-num">04</div>
              <h2 className="guide-section-title">Register VPN profiles with kongtrol init</h2>
              <CodeBlock lang="bash">{`kongtrol init`}</CodeBlock>
              <p>The wizard stores secrets in OS keychain, not in YAML.</p>
            </div>

            <div id="groups" className="guide-section">
              <div className="guide-section-num">05</div>
              <h2 className="guide-section-title">Groups and routing policies</h2>
              <CodeBlock lang="yaml">{`groups:
  work:
    profiles: [office, dev-server, aws]

policies:
  - name: Claude AI
    match:
      domains:
        - claude.ai
        - "*.anthropic.com"
    via: us-content

  - name: Office LAN
    match:
      ip_ranges:
        - 10.10.0.0/16
    via: office`}</CodeBlock>
              <div className="callout">
                <strong>Flow-aware policy:</strong> app + domain/IP conditions can be combined in the same rule.
              </div>
              <div className="callout" style={{ marginTop: 12 }}>
                <strong>Transparent split-DNS:</strong> enable <IC c="monitor.split_dns.enabled: true" /> to map policy domains through the right tunnel for normal apps.
              </div>
            </div>

            <div id="doctor" className="guide-section">
              <div className="guide-section-num">06</div>
              <h2 className="guide-section-title">Doctor check</h2>
              <CodeBlock lang="bash">{`kongtrol doctor`}</CodeBlock>
              <p>Fix any failed line before first production use.</p>
            </div>

            <div id="connect" className="guide-section">
              <div className="guide-section-num">07</div>
              <h2 className="guide-section-title">First connection</h2>
              <CodeBlock lang="bash">{`kongtrol up --group work
kongtrol status --watch`}</CodeBlock>
            </div>

            <div id="dashboard" className="guide-section">
              <div className="guide-section-num">08</div>
              <h2 className="guide-section-title">Web dashboard</h2>
              <CodeBlock lang="bash">{`kongtrol dashboard
# http://127.0.0.1:9741`}</CodeBlock>
              <CodeBlock lang="bash">{`# Useful endpoints
GET /api/v1/metrics/history
GET /api/v1/dns/resolve?domain=claude.ai&via=us-content
GET /api/v1/resolve?target=portal.company.com&app=chrome.exe`}</CodeBlock>
            </div>

            <div id="daily" className="guide-section">
              <div className="guide-section-num">09</div>
              <h2 className="guide-section-title">Daily usage</h2>
              <CodeBlock lang="bash">{`kongtrol up --group work
kongtrol down --group work
kongtrol check
kongtrol export`}</CodeBlock>
              <p style={{ marginTop: 12 }}><strong>Optional scheduler:</strong></p>
              <CodeBlock lang="yaml">{`monitor:
  scheduler:
    enabled: true
    interval: "1m"
    rules:
      - name: "work-hours"
        profiles: ["office", "aws"]
        weekdays: ["mon","tue","wed","thu","fri"]
        start: "09:00"
        end: "18:00"`}</CodeBlock>
              <p style={{ marginTop: 12 }}><strong>Per-profile kill switch override:</strong></p>
              <CodeBlock lang="yaml">{`vpns:
  office:
    kill_switch: true
  us-content:
    kill_switch: false`}</CodeBlock>
            </div>

            <div id="trouble" className="guide-section">
              <div className="guide-section-num">10</div>
              <h2 className="guide-section-title">Troubleshooting</h2>
              <p><strong>Re-auth required:</strong> if token/session credentials expire, Kongtrol emits adapter-specific hints.</p>
              <CodeBlock lang="bash">{`kongtrol init
# Select profile -> refresh credentials`}</CodeBlock>
              <p style={{ marginTop: 20 }}><strong>Reset all tunnels cleanly:</strong></p>
              <CodeBlock lang="bash">{`kongtrol down --all`}</CodeBlock>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
