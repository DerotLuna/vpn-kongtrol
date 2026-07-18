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

        {/* ── How it works (architecture context) ── */}
        <div className="guide-intro">
          <span className="guide-intro-badge">How it works</span>
          <p>
            Kongtrol isn't a VPN client — it's the layer that sits between your apps and the
            VPN clients you already have installed (FortiClient, OpenVPN, ProtonVPN, etc.).
            The <strong>policy engine</strong> looks at every outbound connection (domain, IP or
            app) and decides which tunnel it should go through; the rest of the commands
            (<IC c="up" />, <IC c="doctor" />, <IC c="status" />) just control that routing and
            the tunnels' lifecycle.
          </p>
          <div className="guide-flow">
            <div className="guide-flow-col">
              <h4>your apps</h4>
              <p>Chrome, Slack, terminal, code…</p>
            </div>
            <div className="guide-flow-arrow" aria-hidden="true">──▶</div>
            <div className="guide-flow-col">
              <h4>kongtrol</h4>
              <p>policy engine — domains · IPs · apps</p>
            </div>
            <div className="guide-flow-arrow" aria-hidden="true">──▶</div>
            <div className="guide-flow-col guide-flow-vpns">
              <h4>your VPNs</h4>
              <div className="guide-flow-vpn"><strong>FortiClient</strong><span>office</span></div>
              <div className="guide-flow-vpn"><strong>OpenVPN</strong><span>dev-server</span></div>
              <div className="guide-flow-vpn"><strong>ProtonVPN</strong><span>us-content</span></div>
            </div>
            <div className="guide-flow-footer">kill switch · dns guard · signed audit log</div>
          </div>
        </div>

        <div className="guide-layout">

          {/* ── Sidebar ── */}
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

          {/* ── Content ── */}
          <div className="guide-content" ref={contentRef}>

            {/* ── Mobile section nav (sidebar is hidden on small screens) ── */}
            <nav className="guide-mobile-nav">
              <span className="guide-mobile-nav-label">Jump to</span>
              <select
                value={activeSection}
                onChange={e => {
                  const id = e.target.value
                  setActiveSection(id)
                  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
                }}
              >
                {GUIDE_SECTIONS_EN.map(s => (
                  <option key={s.id} value={s.id}>{s.num} — {s.label}</option>
                ))}
              </select>
            </nav>

            {/* → Quickstart */}
            <div id="quickstart" className="guide-section guide-quickstart">
              <div className="guide-quickstart-head">
                <span className="guide-quickstart-badge">Quickstart</span>
                <p>Your VPNs are already installed and working. Kongtrol just orchestrates them — three commands and you're operating. The 10 sections below are reference; come back to them only if something doesn't fit.</p>
              </div>
              <ol className="qs-steps">
                <li>
                  <span className="qs-n">1</span>
                  <div>
                    <strong>Register your profiles</strong>
                    <p>Detects clients, stores credentials in the keychain, builds policies.</p>
                    <CodeBlock lang="bash">{`$ kongtrol init`}</CodeBlock>
                  </div>
                </li>
                <li>
                  <span className="qs-n">2</span>
                  <div>
                    <strong>Validate the stack</strong>
                    <p>Binaries, certificates, keychain and permissions — before connecting.</p>
                    <CodeBlock lang="bash">{`$ kongtrol doctor`}</CodeBlock>
                  </div>
                </li>
                <li>
                  <span className="qs-n">3</span>
                  <div>
                    <strong>Connect and operate</strong>
                    <p>Bring up a group, watch live status, open the dashboard.</p>
                    <CodeBlock lang="bash">{`$ kongtrol up --group work
$ kongtrol status --watch
$ kongtrol dashboard`}</CodeBlock>
                  </div>
                </li>
              </ol>
            </div>

            {/* 01 — Prerequisites */}
            <div id="prereqs" className="guide-section">
              <div className="guide-section-num">01</div>
              <h2 className="guide-section-title">Prerequisites</h2>
              <p>
                Kongtrol <strong>orchestrates</strong> the VPN clients you already have — it doesn't
                replace them. They need to be installed before you run <IC c="kongtrol init" />.
              </p>

              <div className="table-scroll">
              <table className="data-table">
                <thead>
                  <tr><th>VPN</th><th>Required client</th><th>Check</th></tr>
                </thead>
                <tbody>
                  <tr>
                    <td>FortiClient</td>
                    <td>FortiClient 6.4.x</td>
                    <td>
                      Open FortiClient — at least one saved connection should appear in the list.
                      If you can connect manually from the GUI, you're ready.
                    </td>
                  </tr>
                  <tr>
                    <td>OpenVPN</td>
                    <td>OpenVPN Community</td>
                    <td>
                      {os === 'windows'
                        ? <>CLI or GUI depending on the installer. Community: <IC c="openvpn --version" /> in a terminal (add it to PATH). OpenVPN Connect: GUI-only — Kongtrol detects it automatically.</>
                        : <IC c="openvpn --version" />}
                    </td>
                  </tr>
                  <tr>
                    <td>WireGuard</td>
                    <td>wireguard-tools</td>
                    <td><IC c="wg --version" /></td>
                  </tr>
                  <tr>
                    <td>ProtonVPN</td>
                    <td>{os === 'windows' ? 'ProtonVPN app (GUI) + WireGuard' : 'protonvpn-cli or WireGuard'}</td>
                    <td>
                      {os === 'windows'
                        ? <>Manual GUI: open ProtonVPN and check the connection. Automatic: WireGuard + <IC c="wg --version" />.</>
                        : <><IC c="protonvpn-cli --version" /> or <IC c="wg --version" /> if you'll use a WireGuard profile.</>}
                    </td>
                  </tr>
                  <tr>
                    <td>Cloudflare WARP</td>
                    <td>warp-cli</td>
                    <td><IC c="warp-cli --version" /></td>
                  </tr>
                </tbody>
              </table>
              </div>

              {os === 'windows' && (
                <div className="callout">
                  <strong>Note:</strong> if the wizard shows ✓ for a client, it's detected and ready — no need to check manually in a terminal.
                </div>
              )}

              <div className="callout" style={{ marginTop: 12 }}>
                <strong>Before connecting:</strong> close the VPN client GUIs.
                Kongtrol calls each client's CLI directly — two instances managing
                the same tunnel will conflict.
              </div>

              <div className="table-scroll" style={{ marginTop: 24 }}>
              <table className="data-table">
                <thead>
                  <tr><th>Command</th><th>Required client state</th></tr>
                </thead>
                <tbody>
                  <tr><td>kongtrol init</td><td>Any — only reads files and the keychain</td></tr>
                  <tr><td>kongtrol doctor</td><td>Any — only validates, doesn't connect</td></tr>
                  <tr><td>kongtrol up &lt;profile&gt;</td><td>Client GUI <strong>closed</strong></td></tr>
                </tbody>
              </table>
              </div>
            </div>

            {/* 02 — Installation */}
            <div id="install" className="guide-section">
              <div className="guide-section-num">02</div>
              <h2 className="guide-section-title">Installation</h2>
              <OsTabs os={os} setOS={setOS} />

              {os === 'windows' && (
                <div className="os-panel active">
                  <p>
                    1. Download <IC c="kongtrol_windows_amd64.zip" /> from{' '}
                    <a href="/#install">the downloads section</a>.
                  </p>
                  <p>
                    2. Extract the ZIP. You'll get a <IC c="kongtrol.exe" /> file.
                    Move it to a fixed folder — for example create{' '}
                    <IC c="C:\tools\" /> if you don't have one:
                  </p>
                  <CodeBlock lang="powershell">{`# Create the folder (if missing) and move the binary
New-Item -ItemType Directory -Force "C:\\tools"
Move-Item kongtrol.exe C:\\tools\\kongtrol.exe`}</CodeBlock>
                  <p>
                    3. Add <IC c="C:\tools" /> to PATH so you can run{' '}
                    <IC c="kongtrol" /> from any terminal:
                  </p>
                  <CodeBlock lang="powershell">{`# Run this in PowerShell as Administrator, once:
[Environment]::SetEnvironmentVariable(
  "Path",
  $env:Path + ";C:\\tools",
  "Machine"
)`}</CodeBlock>
                  <p>4. Open a <strong>new</strong> terminal and verify:</p>
                  <CodeBlock lang="powershell">{`kongtrol --help`}</CodeBlock>
                  <div className="callout">
                    <strong>Why a new terminal?</strong> PATH changes only apply to
                    terminals opened after the change — ones already open won't see it.
                  </div>
                </div>
              )}

              {os === 'macos' && (
                <div className="os-panel active">
                  <CodeBlock lang="bash">{`# Apple Silicon (M1+)
$ curl -L https://github.com/vpn-kongtrol/kongtrol/releases/latest/download/kongtrol_darwin_arm64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Intel
$ curl -L https://github.com/vpn-kongtrol/kongtrol/releases/latest/download/kongtrol_darwin_amd64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Verify
$ kongtrol --help`}</CodeBlock>
                  <div className="callout">
                    macOS may ask for confirmation the first time you run the binary.
                    Go to <strong>System Settings → Privacy &amp; Security</strong> and click "Allow".
                  </div>
                </div>
              )}

              {os === 'linux' && (
                <div className="os-panel active">
                  <CodeBlock lang="bash">{`$ curl -L https://github.com/vpn-kongtrol/kongtrol/releases/latest/download/kongtrol_linux_amd64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Verify
$ kongtrol --help`}</CodeBlock>
                  <p>
                    Kongtrol needs network permissions on Linux. Some commands (kill switch,
                    modifying routes) require <IC c="sudo" /> or capabilities.
                  </p>
                </div>
              )}
            </div>

            {/* 03 — Wizard */}
            <div id="wizard" className="guide-section">
              <div className="guide-section-num">03</div>
              <h2 className="guide-section-title">Registering your VPNs: kongtrol init</h2>
              <p>
                <IC c="kongtrol init" /> <strong>doesn't configure your VPNs</strong> — that's
                already done. What it does is tell Kongtrol where they are and how to talk to them:
                which client to use, what tunnel name it has, where the <IC c=".ovpn" /> lives,
                and it stores credentials in the <strong>OS keychain</strong> (never in the YAML).
              </p>
              <p style={{ marginTop: 8 }}>
                If you already have FortiClient with a working tunnel and OpenVPN with its{' '}
                <IC c=".ovpn" />, this step just collects that information.
              </p>

              <h3 style={{ marginBottom: 8, marginTop: 24 }}>Connection files</h3>
              <p>
                <strong>If your VPNs are already configured, you don't need to do anything special here.</strong>{' '}
                Kongtrol just points to the files where they already live — in the wizard you
                simply type each file's current path. You only need to act if you receive new
                files from a coworker or IT and have nowhere to put them, or if you want to
                reorganize everything into a single directory.
              </p>

              <div className="table-scroll" style={{ marginTop: 12 }}>
              <table className="data-table">
                <thead>
                  <tr><th>Adapter</th><th>What the wizard needs</th><th>Anything to do?</th></tr>
                </thead>
                <tbody>
                  <tr>
                    <td>FortiClient</td>
                    <td>{os === 'windows' ? 'Tunnel name + username/password' : 'Cert path, key path, credentials'}</td>
                    <td>
                      {os === 'windows'
                        ? <>No. The tunnel already lives in the FortiClient GUI — Kongtrol activates it by name. No certs.</>
                        : <>Have the <IC c=".crt" /> and <IC c=".key" /> in some accessible path and note it down.</>}
                    </td>
                  </tr>
                  <tr>
                    <td>OpenVPN</td>
                    <td>Path to the <IC c=".ovpn" /></td>
                    <td>
                      {os === 'windows'
                        ? <>No. Use the <IC c=".ovpn" /> wherever it already is (e.g. <IC c="C:\Users\YOU\OpenVPN\config\" />).</>
                        : <>No, if you already have the <IC c=".ovpn" />. Just note its current path.</>}
                    </td>
                  </tr>
                  <tr>
                    <td>ProtonVPN</td>
                    <td>GUI mode: Proton credentials. Automatic mode: WireGuard <IC c=".conf" /> file.</td>
                    <td>GUI: no files. WireGuard: export the config from Proton and save the path.</td>
                  </tr>
                  <tr>
                    <td>WireGuard / others</td>
                    <td>Path to the config file</td>
                    <td>No, if you already have the file. Just note its path.</td>
                  </tr>
                </tbody>
              </table>
              </div>

              <div className="callout">
                <strong>Optional organization:</strong> if you want to centralize things, you can create{' '}
                <IC c={os === 'windows' ? '%USERPROFILE%\\.kongtrol\\certs\\' : '~/.kongtrol/certs/'} />{' '}
                and copy the files there. Useful when onboarding a new teammate or receiving
                fresh certs. Not required if everything's already in place.
              </div>

              <h3 style={{ marginBottom: 8, marginTop: 28 }}>Running the wizard</h3>

              <CodeBlock lang="bash">{`$ kongtrol init`}</CodeBlock>

              <p>
                You first pick a language, then the wizard shows the detected clients
                and asks whether you want to add a profile:
              </p>

              <CodeBlock lang="text">{`▸  VPN clients detected on this system:
    ✓  FortiClient             6.4.10.1821
    ✓  OpenVPN                 2.6.8
    ✓  ProtonVPN               4.3.14

Add a new VPN profile? [y/N]: y`}</CodeBlock>

              <p>
                The wizard uses <strong>numbered menus</strong> where it makes sense — you don't
                have to memorize valid values:
              </p>

              <CodeBlock lang="text">{`  Profile name: office

  Adapter type:
    1) forticlient         ✓ detected  FortiClient SSL VPN
    2) openvpn             ✓ detected  OpenVPN (.ovpn file)
    3) protonvpn           ✓ detected  ProtonVPN (Proton account)
    ───────────────────────────────
    4) ciscoanyconnect                  Cisco AnyConnect
    5) wireguard                        WireGuard (.conf file)
    6) globalprotect                    Palo Alto GlobalProtect
    7) tailscale                        Tailscale mesh / exit node
    8) cloudflarewarp                   Cloudflare WARP

  Choose [1]: 1`}</CodeBlock>

              <p>Every field comes with a hint for where to find the value:</p>

              <details className="guide-more">
                <summary>See the full field-by-field flow (FortiClient)</summary>
              <CodeBlock lang="text">{os === 'windows'
? `    Find it in FortiClient > Settings or ask IT. e.g. vpn.company.com
  VPN host: vpn.yourcompany.com

    443 for SSL VPN (default). Only change if IT tells you otherwise.
  Port [443]:

    The exact connection name as it appears in the FortiClient GUI list.
  Tunnel name [Office]: Office

  FortiClient version:
    1) 6  6.4.x — most common
    2) 7  7.x
    3) 5  5.x — legacy
  Choose [1]:

    On Windows FortiClient connects by tunnel name — no certificate. 'credentials' will be used.
  Username: your_username
  Password for office (stored in keychain): ****`
: `    Find it in FortiClient > Settings or ask IT. e.g. vpn.company.com
  VPN host: vpn.yourcompany.com

    443 for SSL VPN (default). Only change if IT tells you otherwise.
  Port [443]:

    The exact connection name as it appears in the FortiClient GUI list.
  Tunnel name [Office]: Office

  FortiClient version:
    1) 6  6.4.x — most common
    2) 7  7.x
    3) 5  5.x — legacy
  Choose [1]:

  Auth method:
    1) credentials              username and password
    2) certificate              client certificate only
    3) certificate+credentials  certificate + username and password (more secure)
  Choose [3]:

  Client certificate path: ~/.kongtrol/certs/office.crt
  Private key path: ~/.kongtrol/certs/office.key
  Username: your_username
  Password for office (stored in keychain): ****`}</CodeBlock>
              </details>

              <p style={{ marginTop: 20 }}>
                For OpenVPN, the wizard just points to your existing <IC c=".ovpn" /> file:
              </p>

              <details className="guide-more">
                <summary>See the full field-by-field flow (OpenVPN)</summary>
              <CodeBlock lang="text">{os === 'windows'
? `  Profile name: dev-server

  Adapter type:
    ...
  Choose [2]: 2   ← openvpn

    Full path to the .ovpn. Use its current location, no need to copy it.
  Path to .ovpn file: C:\\Users\\YOU\\OpenVPN\\config\\server.ovpn

    If the .ovpn has <cert> and <key> embedded, choose 'certificate' and leave the paths blank.
  Auth method:
    1) credentials              username and password
    2) certificate              client certificate only
    3) certificate+credentials  certificate + username and password (more secure)
  Choose [2]:

  Client certificate path (blank if embedded in the .ovpn):
  Private key path (blank if embedded in the .ovpn):`
: `  Profile name: dev-server

  Adapter type:
    ...
  Choose [2]: 2   ← openvpn

    Full path to the .ovpn. Use its current location, no need to copy it.
  Path to .ovpn file: ~/.config/openvpn/server.ovpn

    If the .ovpn has <cert> and <key> embedded, choose 'certificate' and leave the paths blank.
  Auth method:
    1) credentials              username and password
    2) certificate              client certificate only
    3) certificate+credentials  certificate + username and password (more secure)
  Choose [2]:

  Client certificate path (blank if embedded in the .ovpn):
  Private key path (blank if embedded in the .ovpn):`}</CodeBlock>
              </details>

              <h3 style={{ marginBottom: 8, marginTop: 24 }}>ProtonVPN: manual GUI vs automatic WireGuard</h3>
              <p>
                There are two valid paths with ProtonVPN depending on how much you want to automate.
              </p>
              <ul style={{ marginTop: 4, paddingLeft: 20, lineHeight: 1.8 }}>
                <li>
                  <strong>Manual GUI (fast):</strong> you operate Proton from its app and Kongtrol
                  routes policies over that profile.
                </li>
                <li>
                  <strong>Automatic WireGuard (recommended):</strong> export a <IC c=".conf" /> file from Proton,
                  create a <IC c="type: wireguard" /> profile and Kongtrol can bring the connection up/down by command.
                </li>
              </ul>

              <CodeBlock lang="yaml">{`vpns:
  us-content:
    type: wireguard
    config: ~/.kongtrol/configs/proton-us.conf
    auth:
      method: certificate`}</CodeBlock>

              <div className="callout" style={{ marginTop: 12 }}>
                <strong>ProtonVPN alongside other adapters (Forti/OpenVPN/etc.):</strong> use it in a
                separate profile and only route target domains/IPs (e.g. streaming). Keep
                corporate traffic in its dedicated profiles to avoid route conflicts.
              </div>

              <div className="callout">
                <strong>Where do I find FortiClient's Host and Tunnel Name?</strong><br />
                Open FortiClient → <strong>Remote Access</strong> tab. You'll see a list of saved connections.
                The <strong>connection name</strong> (e.g. "Office VPN") is the <em>tunnel name</em>.
                Click the pencil (edit) icon next to that connection — there you'll see the{' '}
                <strong>Remote Gateway</strong> or <strong>Server</strong> field: that's the <em>host</em>.
              </div>

              <div className="callout" style={{ marginTop: 12 }}>
                If a client isn't auto-detected, the wizard tells you and asks for the
                binary path. You can also set it later in the YAML with <IC c="binary_path" />.
              </div>

              <p>At the end: security options, routing policies and confirmation:</p>

              <details className="guide-more">
                <summary>See the security and policy flow</summary>
              <CodeBlock lang="text">{`Enable kill switch? [Y/n]:
Enable DNS guard? [Y/n]:
Enable signed audit log? [Y/n]:
Enable web dashboard? (http://127.0.0.1:9741) [Y/n]:

── Routing policies ──────────────────────────────────────────────

Add a routing policy? [Y/n]: y

  Policy name (e.g. work, streaming): Claude AI
  VPN profile for this policy:
    1) office
    2) us-content
  Choose [1]: 2

    Domain or suffix (e.g. company.com, .internal) — leave blank to finish
  Domain: claude.ai
  Domain: *.anthropic.com
  Domain:

    IP range or single IP (e.g. 10.0.0.0/8) — leave blank to finish
  IP / range:

Add a routing policy? [Y/n]: y

  Policy name (e.g. work, streaming): Internal network
  VPN profile for this policy:
    1) office
    2) us-content
  Choose [1]: 1

  Domain:

  IP / range: 10.10.0.0/16
  IP / range: 192.168.50.0/24
  IP / range:

Add a routing policy? [Y/n]:

    You can also edit policies directly in kongtrol.yaml → 'policies' section.

Write configuration to ~/.kongtrol/kongtrol.yaml? [Y/n]: y

[✓] Configuration written to ~/.kongtrol/kongtrol.yaml
[✓] Configuration is valid.`}</CodeBlock>
              </details>

              <div className="callout">
                <strong>You can type several domains separated by commas</strong> in a single prompt —
                the wizard splits them automatically. <IC c="claude.ai, *.anthropic.com" /> → two entries.
              </div>

              <div className="callout" style={{ marginTop: 12 }}>
                To add another profile or policy later: run <IC c="kongtrol init" /> again.
                The wizard detects the existing config and won't overwrite anything without confirmation.
              </div>
            </div>

            {/* 04 — Groups */}
            <div id="groups" className="guide-section">
              <div className="guide-section-num">04</div>
              <h2 className="guide-section-title">Groups</h2>
              <p>
                Groups bundle profiles so you can bring them up together with a single command.
                Edit <IC c={os === 'windows' ? 'C:\\Users\\YOU\\.kongtrol\\kongtrol.yaml' : '~/.kongtrol/kongtrol.yaml'} /> and append:
              </p>

              <CodeBlock lang="yaml">{`groups:
  work:
    profiles: [office, dev-server, aws]

  travel:
    profiles: [us-content]

  full:
    profiles: [office, dev-server, aws, us-content]`}</CodeBlock>

              <CodeBlock lang="bash">{`kongtrol up --group work     # brings up office + dev-server + aws in parallel
kongtrol down --group work`}</CodeBlock>

              <div className="callout">
                A profile can belong to several groups at once — <IC c="office" /> can be
                in <IC c="work" /> and also in <IC c="full" /> without any conflict.
              </div>
            </div>

            {/* 05 — Routing policies */}
            <div id="policies" className="guide-section">
              <div className="guide-section-num">05</div>
              <h2 className="guide-section-title">Routing policies</h2>

              <p>
                When you have several tunnels active at once, how does your machine know
                which traffic goes over which VPN? That's what <strong>policies</strong> define.
              </p>

              <CodeBlock lang="text">{`  request                     policy                            tunnel
  ──────────────────────────────────────────────────────────────────────────
  claude.ai            ──▶   domains: claude.ai            ──▶  us-content (ProtonVPN)
  10.10.4.12           ──▶   ip_ranges: 10.10.0.0/16        ──▶  office     (FortiClient)
  api.yourcompany.com  ──▶   no match                       ──▶  direct connection (no VPN)`}</CodeBlock>

              <p>
                Without policies, each VPN installs its own routes on connect — the last
                one to connect usually wins general traffic, and you can't control what goes
                where. With policies, Kongtrol routes precisely:
              </p>

              <ul style={{ marginBottom: 16, paddingLeft: 20 }}>
                <li><IC c="claude.ai" /> and <IC c="*.anthropic.com" /> → go through <IC c="us-content" /> (ProtonVPN)</li>
                <li><IC c="10.10.0.0/16" /> (office network) → goes through <IC c="office" /> (FortiClient)</li>
                <li><IC c="*.yourcompany.com" /> → goes through <IC c="office" /></li>
                <li>Everything else → direct connection, no VPN</li>
              </ul>

              <p>
                The wizard (<IC c="kongtrol init" />) lets you create policies at the end of the flow.
                You can also add or edit them directly in the YAML under the <IC c="policies:" /> section:
              </p>

              <CodeBlock lang="yaml">{`policies:

  # ── US content / Claude AI ──────────────────────────────────────────────
  - name: Claude AI
    match:
      domains:
        - claude.ai
        - "*.anthropic.com"
    via: us-content

  - name: Geo-restricted content
    match:
      domains:
        - "*.netflix.com"
        - "*.hulu.com"
        - "*.disneyplus.com"
    via: us-content

  # ── Office ─────────────────────────────────────────────────────────────
  - name: Office internal network
    match:
      ip_ranges:
        - 10.10.0.0/16          # internal network — adjust to yours
        - 192.168.50.0/24
    via: office

  - name: Internal domains
    match:
      domains:
        - "*.yourcompany.com"   # change to your company's domain
        - intranet.local
    via: office

  # ── Dev server ─────────────────────────────────────────────────────────
  - name: Dev server
    match:
      ip_ranges:
        - 185.0.0.0/32          # replace with your server's real IP
    via: dev-server

  # ── AWS ────────────────────────────────────────────────────────────────
  - name: AWS workloads
    match:
      ip_ranges:
        - 172.31.0.0/16         # AWS default VPC — adjust to your CIDRs
      domains:
        - "*.amazonaws.com"
    via: aws

  # Traffic that doesn't match any rule goes through your normal
  # connection (no VPN). To force EVERYTHING through one VPN:
  # - name: Default
  #   match:
  #     ip_ranges: [0.0.0.0/0]
  #   via: office`}</CodeBlock>

              <div className="callout">
                <strong>How does matching work?</strong><br />
                Domains: exact glob or with a <IC c="*." /> prefix — <IC c="*.company.com" /> covers all subdomains but not <IC c="company.com" /> alone (add both if you need them).<br />
                IPs: longest-prefix match — the most specific rule wins. <IC c="10.10.1.0/24" /> takes priority over <IC c="10.0.0.0/8" />.
              </div>

              <div className="callout" style={{ marginTop: 12 }}>
                <strong>Flow-aware policy:</strong> you can combine app + domain/IP in the same rule.
                If a rule defines <IC c="apps" /> and also <IC c="domains" /> / <IC c="ip_ranges" />,
                both conditions must match to route through that profile.
              </div>

              <div className="callout" style={{ marginTop: 12 }}>
                <strong>Transparent split-DNS:</strong> with <IC c="monitor.split_dns.enabled: true" />,
                Kongtrol injects policy domains into system resolution (hosts) so regular apps
                resolve through the right tunnel without calling the API.
              </div>

              <CodeBlock lang="bash">{`# Validate after editing
$ kongtrol config validate
# [OK] Config is valid.`}</CodeBlock>

              <div className="callout">
                Policies are read on connect — you don't need to restart anything for changes to take effect.
              </div>
            </div>

            {/* 06 — Doctor */}
            <div id="doctor" className="guide-section">
              <div className="guide-section-num">06</div>
              <h2 className="guide-section-title">Verify with doctor</h2>
              <p>
                Before connecting for the first time, let Kongtrol validate everything's in order:
              </p>

              <CodeBlock lang="bash">{`$ kongtrol doctor`}</CodeBlock>

              <CodeBlock lang="text">{os === 'windows'
? `Kongtrol Doctor
────────────────────────────────────────────────────

  Configuration
  ✓  config file         C:\\Users\\you\\.kongtrol\\kongtrol.yaml
  ✓  config valid        4 profile(s) defined

  VPN Binaries
  ✓  forticlient         FortiClient 6.4.10.1821
  ✓  openvpn             OpenVPN 2.6.8
  ✓  protonvpn           ProtonVPN 4.3.14

  Certificates & Keys
  ✓  office: cert        C:\\Users\\you\\.kongtrol\\certs\\office.crt
  ✓  office: key         C:\\Users\\you\\.kongtrol\\certs\\office.key
  ✓  dev-server: config  C:\\Users\\you\\.kongtrol\\configs\\server.ovpn

  Keychain Credentials
  ✓  office: password    found in OS keychain (Credential Manager)
  ✓  us-content          found in OS keychain (Credential Manager)

  Permissions
  ✓  kill switch         available
  ✓  dns guard           available

All checks passed. You're good to go.`
: `Kongtrol Doctor
────────────────────────────────────────────────────

  Configuration
  ✓  config file         ~/.kongtrol/kongtrol.yaml
  ✓  config valid        4 profile(s) defined

  VPN Binaries
  ✓  forticlient         FortiClient 6.4.10.1821
  ✓  openvpn             OpenVPN 2.6.8
  ✓  protonvpn           protonvpn-cli 4.3.14

  Certificates & Keys
  ✓  office: cert        ~/.kongtrol/certs/office.crt
  ✓  office: key         ~/.kongtrol/certs/office.key
  ✓  dev-server: config  ~/.kongtrol/configs/server.ovpn

  Keychain Credentials
  ✓  office: password    found in OS keychain
  ✓  us-content          found in OS keychain

  Permissions
  ✓  kill switch         available
  ✓  dns guard           available

All checks passed. You're good to go.`}</CodeBlock>

              <p>If any line shows ✗, the message tells you exactly what's missing.</p>
            </div>

            {/* 07 — First connection */}
            <div id="connect" className="guide-section">
              <div className="guide-section-num">07</div>
              <h2 className="guide-section-title">First connection</h2>

              <p><strong>A whole group:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol up --group work
[+] office      connected
[+] dev-server  connected
[+] aws         connected`}</CodeBlock>

              <p><strong>Several profiles without a group:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol up office dev-server aws`}</CodeBlock>

              <p>All three tunnels come up in parallel. The routing policy decides which traffic goes through which.</p>

              <p><strong>A single profile:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol up office
$ kongtrol up us-content`}</CodeBlock>

              <p>Kongtrol stays running in the foreground. When you're done:</p>
              <CodeBlock lang="bash">{`# Ctrl+C  →  disconnects everything cleanly

# Or from another terminal:
$ kongtrol down --group work
$ kongtrol down --all`}</CodeBlock>

              <p><strong>Check status:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol status

PROFILE      STATUS        IP           UPTIME
office       connected     10.10.0.5    1h 23m
dev-server   connected     185.x.x.x    1h 23m
aws          connected     172.31.4.7   1h 23m
us-content   disconnected  —            —

Kill Switch: ON

# Live view (refreshes every 2s):
$ kongtrol status --watch`}</CodeBlock>
            </div>

            {/* 08 — Dashboard */}
            <div id="dashboard" className="guide-section">
              <div className="guide-section-num">08</div>
              <h2 className="guide-section-title">Web dashboard</h2>

              <CodeBlock lang="bash">{`$ kongtrol dashboard
# Dashboard running at http://127.0.0.1:9741`}</CodeBlock>

              <p>
                Open <a href="http://localhost:9741" target="_blank" rel="noreferrer">http://localhost:9741</a> in
                your browser. It's a full management console, with sidebar navigation:
              </p>

              <div className="table-scroll">
              <table className="data-table">
                <thead><tr><th>Page</th><th>What it does</th></tr></thead>
                <tbody>
                  <tr><td>Overview</td><td>Live tunnels, per-tunnel traffic charts, active routes, policy resolver, connect/disconnect all</td></tr>
                  <tr><td>Policy Studio</td><td>Routing policy CRUD + a rule tester before saving</td></tr>
                  <tr><td>Security</td><td>Live Kill Switch/DNS Guard toggles, per-profile kill switch override, leak check status</td></tr>
                  <tr><td>VPN Profiles</td><td>VPN profile and group CRUD (config + keychain; needs a daemon restart to activate)</td></tr>
                  <tr><td>Audit Log</td><td>Audit events filterable by profile/level, with an HMAC-signed indicator</td></tr>
                  <tr><td>Settings</td><td>Health check, scheduler + rules, split DNS, kill switch/DNS guard tuning, audit log</td></tr>
                </tbody>
              </table>
              </div>

              <CodeBlock lang="bash">{`# Useful endpoints
GET  /api/v1/metrics/history
GET  /api/v1/dns/resolve?domain=claude.ai&via=us-content
GET  /api/v1/resolve?target=portal.company.com&app=chrome.exe
POST /api/v1/security/killswitch   {"enabled": true}
POST /api/v1/security/dnsguard     {"enabled": true}
GET  /api/v1/vpns                  # profiles · POST/PUT/DELETE for CRUD
GET  /api/v1/groups                # groups · POST/PUT/DELETE + /connect /disconnect
GET  /api/v1/settings              # PUT to save
GET  /api/v1/audit                 # ?profile=&level=&limit=`}</CodeBlock>

              <div className="callout">
                The dashboard is compiled into the binary — nothing extra to install.
                It's full management now (create/edit profiles, policies, groups, toggle the
                kill switch live), not just read-only monitoring. The dashboard's own port/bind
                is still CLI-only (<code>kongtrol config dashboard set-port &lt;port&gt;</code>)
                — changing it from the page serving the request would break the connection.
              </div>
            </div>

            {/* 09 — Daily usage */}
            <div id="daily" className="guide-section">
              <div className="guide-section-num">09</div>
              <h2 className="guide-section-title">Daily usage</h2>

              <div className="table-scroll">
              <table className="data-table">
                <thead><tr><th>Command</th><th>What it does</th></tr></thead>
                <tbody>
                  <tr><td>kongtrol up --group work</td><td>Start the day (office + dev-server + aws)</td></tr>
                  <tr><td>kongtrol down --group work</td><td>End the day</td></tr>
                  <tr><td>kongtrol up us-content</td><td>Netflix, Hulu, Claude AI...</td></tr>
                  <tr><td>kongtrol down --all</td><td>Shut everything down</td></tr>
                  <tr><td>kongtrol status</td><td>See what's connected</td></tr>
                  <tr><td>kongtrol status --watch</td><td>Live monitoring (2s)</td></tr>
                  <tr><td>kongtrol status --watch --dashboard</td><td>Live monitoring + starts the embedded dashboard at the same time</td></tr>
                  <tr><td>kongtrol check</td><td>Run a leak test right now</td></tr>
                  <tr><td>kongtrol dashboard</td><td>Open the web UI</td></tr>
                  <tr><td>kongtrol doctor</td><td>Full diagnostics</td></tr>
                  <tr><td>kongtrol export</td><td>Export config without passwords (to share)</td></tr>
                </tbody>
              </table>
              </div>

              <p style={{ marginTop: 20 }}><strong>Optional time-based scheduler:</strong></p>
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

              <p style={{ marginTop: 16 }}><strong>Per-profile kill switch override:</strong></p>
              <CodeBlock lang="yaml">{`vpns:
  office:
    kill_switch: true
  us-content:
    kill_switch: false`}</CodeBlock>

              <p style={{ marginTop: 24 }}><strong>Update a password:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol init
# → pick the profile → "Update credentials?" → enter the new one`}</CodeBlock>
            </div>

            {/* 10 — Troubleshooting */}
            <div id="trouble" className="guide-section">
              <div className="guide-section-num">10</div>
              <h2 className="guide-section-title">Troubleshooting</h2>

              <p><strong>DNS not restored after a crash:</strong></p>
              <OsTabs os={os} setOS={setOS} />

              {os === 'windows' && (
                <CodeBlock lang="bash">{`$ netsh interface ip set dns "Ethernet" dhcp`}</CodeBlock>
              )}
              {os === 'macos' && (
                <CodeBlock lang="bash">{`$ networksetup -setdnsservers Wi-Fi empty`}</CodeBlock>
              )}
              {os === 'linux' && (
                <CodeBlock lang="bash">{`$ sudo cp /etc/resolv.conf.kongtrol.bak /etc/resolv.conf`}</CodeBlock>
              )}

              <p style={{ marginTop: 32 }}><strong>Kill switch still active with no internet after disconnecting:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol down --all    # disables the kill switch automatically`}</CodeBlock>
              <p>If Kongtrol was closed abruptly:</p>
              <OsTabs os={os} setOS={setOS} />
              {os === 'windows' && (
                <CodeBlock lang="bash">{`# As Administrator:
$ netsh advfirewall reset`}</CodeBlock>
              )}
              {os === 'macos' && (
                <CodeBlock lang="bash">{`$ sudo pfctl -d`}</CodeBlock>
              )}
              {os === 'linux' && (
                <CodeBlock lang="bash">{`$ sudo iptables -F OUTPUT`}</CodeBlock>
              )}

              <p style={{ marginTop: 32 }}>
                <strong>The VPN client isn't showing up in auto-detection:</strong>
              </p>
              <CodeBlock lang="bash">{os === 'windows'
? `# The wizard asks for the binary path if it can't detect it:
> kongtrol init
#   !  This VPN client wasn't auto-detected.
#   Binary path (blank = auto-detect): C:\\custom\\path\\FortiClient.exe

# Or add it manually in %USERPROFILE%\\.kongtrol\\kongtrol.yaml:
# office:
#   type: forticlient
#   binary_path: "C:\\\\custom\\\\path\\\\FortiClient.exe"`
: `# The wizard asks for the binary path if it can't detect it:
$ kongtrol init
#   !  This VPN client wasn't auto-detected.
#   Binary path (blank = auto-detect): /path/to/binary

# Or add it manually in ~/.kongtrol/kongtrol.yaml:
# office:
#   type: forticlient
#   binary_path: "/custom/path/FortiClient"`}</CodeBlock>

              <p style={{ marginTop: 32 }}><strong>Cloudflare WARP — "not registered":</strong></p>
              <CodeBlock lang="bash">{`$ warp-cli register`}</CodeBlock>

              <p style={{ marginTop: 32 }}><strong>Wrong or expired password:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol init
# Select the profile → "Refresh credentials" → enter the new one`}</CodeBlock>

              <div className="callout" style={{ marginTop: 12 }}>
                If a session/token/credential failure is detected during connect/reconnect,
                Kongtrol raises a <strong>re-auth required</strong> alert with an adapter-specific hint.
              </div>

              <p style={{ marginTop: 32 }}><strong>File layout:</strong></p>
              <CodeBlock lang="text">{`~/.kongtrol/
├── kongtrol.yaml     ← main config (no passwords)
├── certs/
│   ├── office.crt
│   └── office.key
├── configs/
│   ├── server.ovpn
│   └── aws.ovpn
└── audit.log         ← signed log of every event

Passwords stored in:
  Windows → Windows Credential Manager
  macOS   → Keychain
  Linux   → libsecret / GNOME Keyring`}</CodeBlock>
            </div>

          </div>
        </div>
      </div>
    </section>
  )
}
