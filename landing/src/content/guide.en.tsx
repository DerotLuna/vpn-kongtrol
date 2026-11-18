// Setup guide content — English.
// Version 2.4.0 · 2026-11-17.
//
// Rendered by components/sections/Guide.tsx, which owns the markup; this file
// holds only the words. Keys and shapes must match guide.es.tsx exactly
// (enforced by the GuideContent type), so both languages always carry the
// same content.
import type { OS } from '../hooks/useOS'
import IC from '../components/ui/InlineCode'
import type { GuideContent } from './guide.es'

export default function guideEN(os: OS): GuideContent {
  return {
    intro1: <><span className="guide-intro-badge">How it works</span></>,
    intro2: <>Kongtrol isn't a VPN client — it's the layer that sits between your apps and the
            VPN clients you already have installed (FortiClient, OpenVPN, ProtonVPN, etc.).
            The <strong>policy engine</strong> looks at every outbound connection (domain, IP or
            app) and decides which tunnel it should go through; the rest of the commands
            (<IC c="up" />, <IC c="doctor" />, <IC c="status" />) just control that routing and
            the tunnels' lifecycle.</>,
    intro3: <>your apps</>,
    intro4: <>Chrome, Slack, terminal, code…</>,
    intro5: <>policy engine — domains · IPs · apps</>,
    intro6: <>your VPNs</>,
    intro7: <>kill switch · dns guard · signed audit log</>,
    intro8: <>Contents</>,
    intro9: "Jump to",
    quickstart1: <><span className="guide-quickstart-badge">Quickstart</span></>,
    quickstart2: <>Your VPNs are already installed and working. Kongtrol just orchestrates them — three commands and you're operating. The 10 sections below are reference; come back to them only if something doesn't fit.</>,
    quickstart3: <><strong>Register your profiles</strong></>,
    quickstart4: <>Detects clients, stores credentials in the keychain, builds policies.</>,
    quickstart5: <><strong>Validate the stack</strong></>,
    quickstart6: <>Binaries, certificates, keychain and permissions — before connecting.</>,
    quickstart7: <><strong>Connect and operate</strong></>,
    quickstart8: <>Bring up a group, watch live status, open the dashboard.</>,
    // feature:providers
    prereqs1: <>Prerequisites</>,
    prereqs2: <>Kongtrol <strong>orchestrates</strong> the VPN clients you already have — it doesn't
                replace them. They need to be installed before you run <IC c="kongtrol init" />.</>,
    prereqs3: 'Required client',
    prereqs4: 'Check',
    prereqs5: <>Open FortiClient — at least one saved connection should appear in the list.
                      If you can connect manually from the GUI, you're ready.</>,
    prereqs6: <>{os === 'windows'
                        ? <>CLI or GUI depending on the installer. Community: <IC c="openvpn --version" /> in a terminal (add it to PATH). OpenVPN Connect: GUI-only — Kongtrol detects it automatically.</>
                        : <IC c="openvpn --version" />}</>,
    prereqs7: 'Official ProtonVPN app (or WireGuard with its .conf)',
    prereqs8: <>{os === 'linux'
                        ? <>Sign in to the app. On Linux Kongtrol can connect for you; if not, you connect and it detects it.</>
                        : <>Sign in to the app. You connect and Kongtrol detects the tunnel on its own.</>}
                      {' '}No app: a <IC c="wireguard" /> profile with Proton's <IC c=".conf" />.</>,
    prereqs9: <><strong>Note:</strong> if the wizard shows ✓ for a client, it's detected and ready — no need to check manually in a terminal.</>,
    prereqs10: <><strong>Keep your VPN app:</strong> Kongtrol works with it.
                If the tunnel is already up, it attaches to it. If not, it opens the app and connects
                for you where it can; if it can't, it tells you what to do and waits for you to connect.</>,
    prereqs11: <>Command</>,
    prereqs12: <>Required client state</>,
    prereqs13: <>Any — only reads files and the keychain</>,
    prereqs14: <>Any — only validates, doesn't connect</>,
    prereqs15: <>kongtrol up &lt;profile&gt;</>,
    prereqs16: <>App installed; open or closed, either works (Kongtrol opens it if needed)</>,
    // feature:helper
    install1: <>Installation</>,
    install2: <>1. Download <IC c="kongtrol_windows_amd64.zip" /> from{' '}
                    <a href="/en/#install">the downloads section</a>.</>,
    install3: <>2. Extract the ZIP. You'll get a <IC c="kongtrol.exe" /> file.
                    Move it to a fixed folder — for example create{' '}
                    <IC c="C:\tools\" /> if you don't have one:</>,
    install4: `# Create the folder (if missing) and move the binary
New-Item -ItemType Directory -Force "C:\\tools"
Move-Item kongtrol.exe C:\\tools\\kongtrol.exe`,
    install5: <>3. Add <IC c="C:\tools" /> to your user PATH (no administrator needed) so you can
                    run <IC c="kongtrol" /> from any terminal:</>,
    install6: `# In a normal PowerShell, once:
[Environment]::SetEnvironmentVariable(
  "Path",
  [Environment]::GetEnvironmentVariable("Path", "User") + ";C:\\tools",
  "User"
)`,
    install7: <>4. Open a <strong>new</strong> terminal and verify:</>,
    install8: <><strong>Why a new terminal?</strong> PATH changes only apply to
                    terminals opened after the change — ones already open won't see it.</>,
    install9: `# Apple Silicon (M1+)
$ curl -L https://github.com/DerotLuna/vpn-kongtrol/releases/latest/download/kongtrol_darwin_arm64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Intel
$ curl -L https://github.com/DerotLuna/vpn-kongtrol/releases/latest/download/kongtrol_darwin_amd64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Verify
$ kongtrol --help`,
    install10: <>macOS may ask for confirmation the first time you run the binary.
                    Go to <strong>System Settings → Privacy &amp; Security</strong> and click "Allow".</>,
    install11: `$ curl -L https://github.com/DerotLuna/vpn-kongtrol/releases/latest/download/kongtrol_linux_amd64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Verify
$ kongtrol --help`,
    install12: <><strong>One step as administrator:</strong> run <IC c="kongtrol service install" /> once
                    and accept the prompt. It installs a helper that makes the network changes (routes, DNS, kill switch).
                    After that no command needs an administrator terminal or <IC c="sudo" />.</>,
    // feature:wizard
    wizard1: <>Registering your VPNs: kongtrol init</>,
    wizard2: <><IC c="kongtrol init" /> <strong>doesn't configure your VPNs</strong> — that's
                already done. What it does is tell Kongtrol where they are and how to talk to them:
                which client to use, what tunnel name it has, where the <IC c=".ovpn" /> lives,
                and it stores credentials in the <strong>OS keychain</strong> (never in the YAML).</>,
    wizard3: <>If you already have FortiClient with a working tunnel and OpenVPN with its{' '}
                <IC c=".ovpn" />, this step just collects that information.</>,
    wizard4: <>Connection files</>,
    wizard5: <><strong>If your VPNs are already configured, you don't need to do anything special here.</strong>{' '}
                Kongtrol just points to the files where they already live — in the wizard you
                simply type each file's current path. You only need to act if you receive new
                files from a coworker or IT and have nowhere to put them, or if you want to
                reorganize everything into a single directory.</>,
    wizard6: 'Adapter',
    wizard7: 'What the wizard needs',
    wizard8: 'Anything to do?',
    wizard9: os === 'windows' ? 'Tunnel name + username/password' : 'Cert path, key path, credentials',
    wizard10: <>{os === 'windows'
                        ? <>No. The tunnel already lives in the FortiClient GUI — Kongtrol activates it by name. No certs.</>
                        : <>Have the <IC c=".crt" /> and <IC c=".key" /> in some accessible path and note it down.</>}</>,
    wizard11: <>Path to the <IC c=".ovpn" /></>,
    wizard12: <>{os === 'windows'
                        ? <>No. Use the <IC c=".ovpn" /> wherever it already is (e.g. <IC c="C:\Users\YOU\OpenVPN\config\" />).</>
                        : <>No, if you already have the <IC c=".ovpn" />. Just note its current path.</>}</>,
    wizard13: <>GUI mode: Proton credentials. Automatic mode: WireGuard <IC c=".conf" /> file.</>,
    wizard14: <>GUI: no files. WireGuard: export the config from Proton and save the path.</>,
    wizard15: <>WireGuard / others</>,
    wizard16: <>Path to the config file</>,
    wizard17: <>No, if you already have the file. Just note its path.</>,
    wizard18: <><strong>Optional organization:</strong> if you want to centralize things, you can create{' '}
                <IC c={os === 'windows' ? '%USERPROFILE%\\.kongtrol\\certs\\' : '~/.kongtrol/certs/'} />{' '}
                and copy the files there. Useful when onboarding a new teammate or receiving
                fresh certs. Not required if everything's already in place.</>,
    wizard19: <>Running the wizard</>,
    wizard20: <>You first pick a language, then the wizard shows the detected clients
                and asks whether you want to add a profile:</>,
    wizard21: `▸  VPN clients detected on this system:
    ✓  FortiClient             6.4.10.1821
    ✓  OpenVPN                 2.6.8
    ✓  ProtonVPN               4.3.14

Add a new VPN profile? [y/N]: y`,
    wizard22: <>The wizard uses <strong>numbered menus</strong> where it makes sense — you don't
                have to memorize valid values:</>,
    wizard23: `  Profile name: office

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

  Choose [1]: 1`,
    wizard24: <>Every field comes with a hint for where to find the value:</>,
    wizard25: <>See the full field-by-field flow (FortiClient)</>,
    wizard26: os === 'windows'
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
  Password for office (stored in keychain): ****`,
    wizard27: <>For OpenVPN, the wizard just points to your existing <IC c=".ovpn" /> file:</>,
    wizard28: <>See the full field-by-field flow (OpenVPN)</>,
    wizard29: os === 'windows'
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
  Private key path (blank if embedded in the .ovpn):`,
    wizard30: <>ProtonVPN: manual GUI vs automatic WireGuard</>,
    wizard31: <>There are two valid paths with ProtonVPN depending on how much you want to automate.</>,
    wizard32: <><strong>Manual GUI (fast):</strong> you operate Proton from its app and Kongtrol
                  routes policies over that profile.</>,
    wizard33: <><strong>Automatic WireGuard (recommended):</strong> export a <IC c=".conf" /> file from Proton,
                  create a <IC c="type: wireguard" /> profile and Kongtrol can bring the connection up/down by command.</>,
    wizard34: <><strong>ProtonVPN alongside other adapters (Forti/OpenVPN/etc.):</strong> use it in a
                separate profile and only route target domains/IPs (e.g. streaming). Keep
                corporate traffic in its dedicated profiles to avoid route conflicts.</>,
    wizard35: <><strong>Where do I find FortiClient's Host and Tunnel Name?</strong><br />
                Open FortiClient → <strong>Remote Access</strong> tab. You'll see a list of saved connections.
                The <strong>connection name</strong> (e.g. "Office VPN") is the <em>tunnel name</em>.
                Click the pencil (edit) icon next to that connection — there you'll see the{' '}
                <strong>Remote Gateway</strong> or <strong>Server</strong> field: that's the <em>host</em>.</>,
    wizard36: <>If a client isn't auto-detected, the wizard tells you and asks for the
                binary path. You can also set it later in the YAML with <IC c="binary_path" />.</>,
    wizard37: <>At the end: security options, routing policies and confirmation:</>,
    wizard38: <>See the security and policy flow</>,
    wizard39: `Enable kill switch? [Y/n]:
Enable DNS guard? [Y/n]:
Enable signed audit log? [Y/n]:
Enable web dashboard? (http://127.0.0.1:9741) [Y/n]:
Show your public IP? (asks an external service) [y/N]:
  Uses an external service (api.ipify.org) that sees your public IP

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
[✓] Configuration is valid.`,
    wizard40: <><strong>You can type several domains separated by commas</strong> in a single prompt —
                the wizard splits them automatically. <IC c="claude.ai, *.anthropic.com" /> → two entries.</>,
    wizard41: <>To add another profile or policy later: run <IC c="kongtrol init" /> again.
                The wizard detects the existing config and won't overwrite anything without confirmation.</>,
    groups1: <>Groups</>,
    groups2: <>Groups bundle profiles so you can bring them up together with a single command.
                Edit <IC c={os === 'windows' ? 'C:\\Users\\YOU\\.kongtrol\\kongtrol.yaml' : '~/.kongtrol/kongtrol.yaml'} /> and append:</>,
    groups3: `kongtrol up --group work     # brings up office + dev-server + aws in parallel
kongtrol down --group work`,
    groups4: <>A profile can belong to several groups at once — <IC c="office" /> can be
                in <IC c="work" /> and also in <IC c="full" /> without any conflict.</>,
    groups5: <>If a daemon is already running (<IC c="kongtrol up" /> in the background, or via the
                dashboard) and you hand-edit the group, you don't need to restart it —{' '}
                <IC c="kongtrol reload --group work" /> re-reads <IC c="kongtrol.yaml" /> from disk and
                reconnects only that group's active tunnels. For a single tunnel,{' '}
                <IC c="kongtrol reload --tunnel office" />.</>,
    // feature:policies feature:routes
    policies1: <>Routing policies</>,
    policies2: <>When you have several tunnels active at once, how does your machine know
                which traffic goes over which VPN? That's what <strong>policies</strong> define.</>,
    policies3: `  request                     policy                            tunnel
  ──────────────────────────────────────────────────────────────────────────
  claude.ai            ──▶   domains: claude.ai            ──▶  us-content (ProtonVPN)
  10.10.4.12           ──▶   ip_ranges: 10.10.0.0/16        ──▶  office     (FortiClient)
  api.yourcompany.com  ──▶   no match                       ──▶  direct connection (no VPN)`,
    policies4: <>Without policies, each VPN installs its own routes on connect — the last
                one to connect usually wins general traffic, and you can't control what goes
                where. With policies, Kongtrol routes precisely:</>,
    policies5: <><IC c="claude.ai" /> and <IC c="*.anthropic.com" /> → go through <IC c="us-content" /> (ProtonVPN)</>,
    policies6: <><IC c="10.10.0.0/16" /> (office network) → goes through <IC c="office" /> (FortiClient)</>,
    policies7: <><IC c="*.yourcompany.com" /> → goes through <IC c="office" /></>,
    policies8: <>Everything else → direct connection, no VPN</>,
    policies9: <>The wizard (<IC c="kongtrol init" />) lets you create policies at the end of the flow.
                You can also add or edit them directly in the YAML under the <IC c="policies:" /> section:</>,
    policies10: `policies:

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
  #   via: office`,
    policies11: <><strong>How does matching work?</strong><br />
                Domains: exact glob or with a <IC c="*." /> prefix — <IC c="*.company.com" /> covers all subdomains but not <IC c="company.com" /> alone (add both if you need them).<br />
                IPs: longest-prefix match — the most specific rule wins. <IC c="10.10.1.0/24" /> takes priority over <IC c="10.0.0.0/8" />.</>,
    policies12: <><strong>App rules (experimental):</strong> you can write <IC c="apps" /> in a rule,
                but today it does not change routing: only <IC c="domains" /> and <IC c="ip_ranges" /> count.
                For now <IC c="apps" /> only shows up in <IC c="kongtrol map app:<exe>" />.</>,
    policies13: <><strong>Transparent split-DNS:</strong> with <IC c="monitor.split_dns.enabled: true" />,
                Kongtrol injects policy domains into system resolution (hosts) so regular apps
                resolve through the right tunnel without calling the API.</>,
    policies14: `# Validate after editing
$ kongtrol config validate
# [OK] Config is valid.`,
    policies15: <>If no daemon is running yet, policies are read when you run <IC c="kongtrol up" />.
                If one is already active, apply the change without restarting it via{' '}
                <IC c="kongtrol reload --policy" /> — reloads the policy engine from the edited{' '}
                <IC c="kongtrol.yaml" />, without touching any tunnels.</>,
    doctor1: <>Verify with doctor</>,
    doctor2: <>Before connecting for the first time, let Kongtrol validate everything's in order:</>,
    doctor3: os === 'windows'
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
  ✓  us-content: password  found in OS keychain (Credential Manager)

  Permissions
  ✓  kill switch         platform implementation available
  ✓  dns guard           platform implementation available

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
  ✓  us-content: password  found in OS keychain

  Permissions
  ✓  kill switch         platform implementation available
  ✓  dns guard           platform implementation available

All checks passed. You're good to go.`,
    doctor4: <>If any line shows ✗, the message tells you exactly what's missing.</>,
    // feature:daemon_lifecycle
    connect1: <>First connection</>,
    connect2: <><strong>A whole group:</strong></>,
    connect3: `$ kongtrol up --group work
[+] office      connected
[+] dev-server  connected
[+] aws         connected`,
    connect4: <><strong>Several profiles without a group:</strong></>,
    connect5: <>All three tunnels come up in parallel. The routing policy decides which traffic goes through which.</>,
    connect6: <><strong>A single profile:</strong></>,
    connect7: <>Kongtrol keeps running in the background: closing the view disconnects nothing. When you're done:</>,
    connect8: `# q or Ctrl+C  →  closes the view; the VPNs stay connected

# To disconnect (from any terminal):
$ kongtrol down --group work
$ kongtrol down --all`,
    connect9: <><strong>Check status:</strong></>,
    connect10: `$ kongtrol status

PROFILE      STATUS        IP           UPTIME
office       connected     10.10.0.5    1h 23m
dev-server   connected     185.x.x.x    1h 23m
aws          connected     172.31.4.7   1h 23m
us-content   disconnected  —            —

Kill switch: ARMED (protecting 3 tunnel(s))

# Live view (refreshes every 2s):
$ kongtrol status --watch`,
    // feature:dashboard
    dashboard1: <>Web dashboard</>,
    dashboard2: <>Open <a href="http://localhost:9741" target="_blank" rel="noreferrer">http://localhost:9741</a> in
                your browser. It's a full management console, with sidebar navigation:</>,
    dashboard3: <>Page</>,
    dashboard4: <>What it does</>,
    dashboard5: <>Live tunnels, per-tunnel traffic charts, active routes, policy resolver, group quick-launch, connect/disconnect all</>,
    dashboard6: <>Tabs: routing policies (CRUD + rule tester), VPN profiles (CRUD, config + keychain), and groups (CRUD + connect/disconnect)</>,
    dashboard7: <>Live Kill Switch/DNS Guard toggles, per-profile kill switch override, leak check status, optional public IP lookup (off by default) with a Check button</>,
    dashboard8: <>Audit events filterable by profile/level, with an HMAC-signed indicator</>,
    dashboard9: <>Health check, scheduler + rules, split DNS, kill switch/DNS guard tuning, public IP services, audit log</>,
    dashboard10: `# Useful endpoints
GET  /api/v1/metrics/history
GET  /api/v1/dns/resolve?domain=claude.ai&via=us-content
GET  /api/v1/resolve?target=portal.company.com&app=chrome.exe
POST /api/v1/security/killswitch   {"enabled": true}
POST /api/v1/security/dnsguard     {"enabled": true}
POST /api/v1/security/publicip     {"enabled": true}   # optional: look up your public IP
POST /api/v1/security/publicip/check                    # Check now
GET  /api/v1/vpns                  # profiles · POST/PUT/DELETE for CRUD
GET  /api/v1/groups                # groups · POST/PUT/DELETE + /connect /disconnect
GET  /api/v1/settings              # PUT to save
GET  /api/v1/audit                 # ?profile=&level=&limit=`,
    dashboard11: <>The dashboard is compiled into the binary — nothing extra to install.
                It's full management now (create/edit profiles, policies, groups, toggle the
                kill switch live), not just read-only monitoring. The dashboard's own port/bind
                is still CLI-only (<code>kongtrol config dashboard set-port &lt;port&gt;</code>)
                — changing it from the page serving the request would break the connection.</>,
    // feature:i18n
    daily1: <>Daily usage</>,
    daily2: <>Command</>,
    daily3: <>What it does</>,
    daily4: <>Start the day (office + dev-server + aws)</>,
    daily5: <>End the day</>,
    daily6: <>Shut everything down</>,
    daily7: <>See what's connected</>,
    daily8: <>Live monitoring (2s)</>,
    daily9: <>Live monitoring + starts the embedded dashboard at the same time</>,
    daily10: <>Look up your public IP and test for leaks now (if you turned the lookup on)</>,
    daily11: <>Open the web UI</>,
    daily12: <>Full diagnostics</>,
    daily13: <>Export config without passwords (to share)</>,
    daily14: <><strong>Optional time-based scheduler:</strong></>,
    daily15: <><strong>Per-profile kill switch override:</strong></>,
    daily16: <><strong>Update a password:</strong></>,
    daily17: `$ kongtrol init
# → pick the profile → "Update credentials?" → enter the new one`,
    // Command reference: same list and order as `kongtrol --help`
    // The rows come from docs/cli.json (generated from the CLI: go generate ./cmd/kongtrol).
    cmdref1: <>Command reference</>,
    cmdref2: <>The same groups as <IC c="kongtrol --help" />. Every command has its own help with examples: <IC c="kongtrol <command> --help" />.</>,
    // feature:killswitch feature:dns_guard
    trouble1: <>Troubleshooting</>,
    trouble2: <><strong>DNS not restored after a crash:</strong></>,
    trouble3: <><strong>Kill switch still active with no internet after disconnecting:</strong></>,
    trouble4: `$ kongtrol down --all    # disables the kill switch automatically`,
    trouble5: <>If Kongtrol closed abruptly: with the helper installed, its blocks lift on their own in about
                20 seconds (the network stays open, never cut). If they're still there, remove <strong>only</strong> Kongtrol's rules.
                Don't use <IC c="netsh advfirewall reset" />: it deletes all your firewall rules.</>,
    trouble6: os === 'windows'
? `# PowerShell as Administrator:
> netsh advfirewall firewall delete rule name=KongtrolBlock
> netsh advfirewall firewall delete rule name=KongtrolKillSwitchBlock`
: os === 'macos'
? `$ sudo pfctl -a com.kongtrol -F all`
: `$ sudo iptables -F KONGTROL_BLOCK
$ sudo ip6tables -F KONGTROL_BLOCK
$ sudo iptables -F KONGTROL`,
    trouble7: <><strong>The VPN client isn't showing up in auto-detection:</strong></>,
    trouble8: os === 'windows'
? `# The wizard asks for the binary path if it can't detect it:
> kongtrol init
#   !  This VPN client wasn't auto-detected.
#   Binary path (blank = auto-detect): C:\\custom\\path\\FortiClient.exe

# Or add it manually in %USERPROFILE%\\.kongtrol\\kongtrol.yaml:
# office:
#   type: forticlient
#   app_path: "C:\\\\custom\\\\path\\\\FortiClient.exe"`
: `# The wizard asks for the binary path if it can't detect it:
$ kongtrol init
#   !  This VPN client wasn't auto-detected.
#   Binary path (blank = auto-detect): /path/to/binary

# Or add it manually in ~/.kongtrol/kongtrol.yaml:
# office:
#   type: forticlient
#   app_path: "/custom/path/FortiClient"`,
    trouble9: <><strong>Wrong or expired password:</strong></>,
    trouble10: `$ kongtrol init
# Select the profile → "Refresh credentials" → enter the new one`,
    trouble11: <>If a session/token/credential failure is detected during connect/reconnect,
                Kongtrol raises a <strong>re-auth required</strong> alert with an adapter-specific hint.</>,
    trouble12: <><strong>File layout:</strong></>,
    trouble13: `~/.kongtrol/
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
  Linux   → libsecret / GNOME Keyring`,
  }
}
