// Setup guide content — English.
// Version 3.9.0 · 2026-11-25.
//
// Rendered by components/sections/Guide.tsx, which owns the markup; this file
// holds only the words. Keys and shapes must match guide.es.tsx exactly
// (enforced by the GuideContent type and scripts/check-i18n.mjs). Every fact
// is checked against the code and docs/SETUP.md (research note
// docs/research/2026-11-19-landing-guide-accuracy.md). File names come only
// from links.ts.
import type { OS } from '../hooks/useOS'
import IC from '../components/ui/InlineCode'
import { pathFor } from '../site'
import { DEFENDER_FALSE_POSITIVE_URL, INSTALL_CMD, UNINSTALL_SCRIPT_BASE, WINDOWS_INSTALLER_FILE } from '../links'
import type { GuideContent, GuideFix, GuideStep } from './guide.es'

const tray = (os: OS) => os === 'windows'
  ? 'in the tray, next to the clock (if you don\'t see it, open the ^ arrow)'
  : os === 'macos'
    ? 'in the menu bar, top right'
    : 'in your desktop\'s system tray'

export default function guideEN(os: OS): GuideContent {
  return {
    labels: {
      toc: 'Contents',
      jump: 'Go to',
      see: 'You\'ll see',
      fix: 'What to do',
      cmd: 'Command',
      does: 'What it does',
    },
    // feature:first_run
    qs: {
      badge: 'Quickstart',
      lead: <>Four steps, no terminal. Kongtrol works with the VPNs you already use: it doesn't replace them.</>,
      steps: [
        {
          t: <>Install Kongtrol</>,
          d: os === 'windows'
            ? <>Download the installer from <a href="/en/#install">downloads</a> and open it. It installs for your user only.</>
            : <>Paste the line from <a href="#install">Install</a> into Terminal, or download from <a href="/en/#install">downloads</a>.</>,
          see: <>The Kongtrol icon {tray(os)}.</>,
        },
        {
          t: <>Give permission once</>,
          d: <>Kongtrol installs a <strong>helper</strong>: a small service that changes routes and DNS for you.
            {os === 'windows'
              ? <> The installer asks: click <strong>Yes</strong> on the Windows prompt.</>
              : <> In the icon's menu choose <strong>Install helper (once)</strong> and type your password.</>}</>,
          see: <>"Helper ready". It never asks again.</>,
        },
        {
          t: <>Create your first VPN</>,
          d: <>The panel opens in your browser, on <strong>Profiles</strong>. Pick your VPN, fill in the details and click <strong>Save</strong>.</>,
          see: <>Your VPN in the list. Kongtrol restarts by itself to use it.</>,
        },
        {
          t: <>Connect</>,
          d: <>From the icon or from <strong>Home</strong> in the panel. If you use your VPN's app, connect there as usual: Kongtrol detects it.</>,
          see: <>"your-vpn · connected" with a green dot, and the icon turns green.</>,
        },
      ] as GuideStep[],
      cli: <>Prefer the terminal? <IC c="kongtrol init" /> does the same with step-by-step questions.</>,
    },
    how: {
      title: <>How it works</>,
      lead: <>Kongtrol is not a VPN. It sits between your apps and the VPNs you already have, and decides which one each destination uses.
        Anything without a rule uses your normal internet.</>,
      apps: <>your apps</>,
      appsSub: <>browser, Slack, terminal…</>,
      engine: <>rules: sites · networks</>,
      vpns: <>your VPNs</>,
      foot: <>kill switch · DNS protection · signed log · all local</>,
    },
    // feature:providers feature:rule_move
    prereqs: {
      title: <>Before you start</>,
      lead: <>Have your VPN working as usual. Kongtrol uses it; it doesn't set it up for you.</>,
      col1: 'Your VPN',
      col2: 'What you need',
      rows: [
        { vpn: <>NordVPN, Proton VPN, Surfshark, FortiClient, Mullvad, PIA, ExpressVPN, Windscribe, CyberGhost</>, need: <>Its official app installed and signed in. If Kongtrol cannot detect the tunnel, specify its interface in advanced settings. Connect ExpressVPN on Windows and CyberGhost from the provider client.</> },
        { vpn: <>WireGuard, OpenVPN</>, need: <>The <IC c=".conf" /> or <IC c=".ovpn" /> file from your provider or your IT team, and the{' '}
          {os === 'windows' ? 'WireGuard or OpenVPN for Windows' : os === 'macos' ? 'wireguard-tools or OpenVPN' : 'wireguard-tools or openvpn'} program.</> },
        { vpn: <>Tailscale, Cloudflare WARP, Cisco Secure Client, GlobalProtect</>, need: <>Its official program installed and signed in.</> },
      ],
      note: <>Full list and what Kongtrol does with each: <a href="/en/#vpns">supported VPNs</a>. Proton VPN also works without its app,
        as a <IC c="wireguard" /> profile with the file you download from your account. Use one or the other, not both
        at once: they share your account and clash. Kongtrol will not connect that file while the Proton app is connected,
        and if you connect the app, it disconnects the file and tells you. If you add the Proton app when that file already has rules,
        Kongtrol offers to move them to the app (or later: <IC c="kongtrol policy move" />).</>,
    },
    // feature:helper feature:windows_installer_shutdown
    install: {
      title: <>Install</>,
      steps: (os === 'windows'
        ? [
          { t: <>Download the installer</>, d: <>In <a href="/en/#install">downloads</a>, the Windows button gets <IC c={WINDOWS_INSTALLER_FILE} />.</>, see: <>The file in your Downloads folder.</> },
          { t: <>Open it</>, d: <>If a blue Windows warning (SmartScreen) shows up, click <strong>More info</strong>, then <strong>Run anyway</strong>. It shows because the file isn't signed yet.
            {' '}If Microsoft Defender flags it as a threat (for example <IC c="Trojan:Win32/Bearfoos.A!ml" />) and quarantines it, it's a false positive common for new unsigned programs: check it on the <a href="/en/security">Security</a> page, allow it in Windows Security → Protection history only if you trust it, and report it at{' '}
            <a href={DEFENDER_FALSE_POSITIVE_URL}>microsoft.com/wdsi/filesubmission</a>.</>, see: <>The Kongtrol setup wizard.</> },
          { t: <>Keep the recommended options</>, d: <>Open at sign-in, install the network helper and add the <IC c="kongtrol" /> command. No administrator needed, except for the helper (once).</>, see: <>A Windows prompt for the helper: click <strong>Yes</strong>.</> },
          { t: <>Finish</>, d: <>With "Open Kongtrol and create my first VPN" checked, go on to <a href="#wizard">Your first VPN</a>.</>, see: <>The icon in the tray and the panel in your browser.</> },
        ]
        : [
          { t: <>Open Terminal</>, d: os === 'macos' ? <>Search with <strong>Cmd + Space</strong> → "Terminal".</> : <>From your desktop's applications menu.</>, see: <>A window with a line to type in.</> },
          { t: <>Paste this line and press Enter</>, code: `$ ${INSTALL_CMD.unix}`, d: <>It downloads the latest version, checks it wasn't tampered with and installs it.</>, see: os === 'macos' ? <><IC c="~/Applications/Kongtrol.app" /> installed, and Kongtrol opens.</> : <>The <IC c="kongtrol" /> command, the icon and a menu entry; Kongtrol opens.</> },
          { t: <>Install the helper</>, d: <>In the icon's menu, <strong>Install helper (once)</strong>.</>, see: <>It asks for your password once; then "Helper ready".</> },
        ]) as GuideStep[],
      scriptNote: os === 'macos'
        ? <>Installed this way, macOS shows no unverified-app warning. If you download from the browser, the first time:
          right-click → <strong>Open</strong>, or System Settings → Privacy &amp; Security → <strong>Open Anyway</strong>.
          {' '}Only the <IC c="kongtrol" /> command, without the app? End the line with <IC c="| sh -s -- --cli-only" />.</>
        : os === 'linux'
          ? <>Another option: the <IC c=".deb" /> (Ubuntu, Debian) or <IC c=".rpm" /> (Fedora) package from{' '}
            <a href="/en/#install">downloads</a>, with a double click. On GNOME the icon needs the AppIndicator extension (Ubuntu ships it).
            {' '}Only the <IC c="kongtrol" /> command? End the line with <IC c="| sh -s -- --cli-only" />; with no desktop (a server, SSH) it does that by itself and tells you.</>
          : <>No installer? This line in PowerShell (no administrator) installs the same: the app with its icon and the <IC c="kongtrol" /> command.
            {' '}Only the command: type <IC c="$env:KONGTROL_CLI_ONLY=1" /> first. If you used the installer, update with the new installer.
            {' '}If Windows says it could not close the applications, cancel setup, choose <strong>Quit and disconnect all</strong> from the tray, and run the installer again.</>,
      winScript: INSTALL_CMD.windows,
      helper: <><strong>One permission, once.</strong> After that no step needs an administrator terminal or{' '}
        <IC c="sudo" />. If the helper is missing, the panel says so with a <strong>Fix</strong> button; in the terminal:{' '}
        <IC c="kongtrol service install" />.</>,
    },
    // feature:wizard feature:profile_import feature:add_sheet
    wizard: {
      title: <>Your first VPN</>,
      lead: <>In the panel, <strong>Profiles</strong> (or the <strong>＋ Add</strong> button at the top). Pick one of these:</>,
      ways: [
        <><strong>A VPN you already use</strong> (NordVPN, Proton, FortiClient…): pick the app and give it a name, for example <IC c="office" />.</>,
        <><strong>A <IC c=".conf" /> or <IC c=".ovpn" /> file</strong>: import it with <strong>Choose file</strong>, or drop it on the panel. If it comes with separate certificates (<IC c="ca.crt" />…), Kongtrol takes them from the same folder or from the path the file names; if you dropped it, Kongtrol looks for the original in your usual folders (OpenVPN, Downloads, Desktop, Documents). If some are still missing, it names them: choose the folder they are in, or add them one by one. It checks it and keeps a protected copy.</>,
        <><strong>Username and password</strong>: they go to the system keychain, never to a text file.</>,
        <><strong>Save and activate</strong> saves it and connects it; <strong>Save</strong> only saves it, so you connect it whenever you want.</>,
      ],
      attach: <><strong>If you use your VPN's app:</strong> when you connect, if the tunnel is already up, Kongtrol attaches to it.
        If not, it opens the app and connects for you when it can. If it can't, it tells you what to do and waits for you to connect.</>,
      cliTitle: <>From the terminal</>,
      cli: <>To create your first profile, run <IC c="kongtrol init" />.</>,
      cliSteps: [
        <>Your VPNs: it finds the installed apps (✓) and asks for each profile's name and details.</>,
        <>Save the profile and exit, or continue to rules and security.</>,
        <>If you continue, choose which sites or networks use the VPN and set options like the kill switch and DNS protection.</>,
        <>Kongtrol saves your changes to <IC c="~/.kongtrol/kongtrol.yaml" /> and offers to install the helper if needed.</>,
      ],
      cliMore: <>If you already have profiles, <IC c="kongtrol init" /> shows a menu: add a profile, update credentials,
        rules, security or save. For scripts: <IC c="kongtrol profile add" /> and <IC c="kongtrol profile import" />.
        The language follows your system; change it with <IC c="kongtrol config lang es" />.</>,
      code: `$ kongtrol init
$ kongtrol profile import ~/Downloads/proton-us.conf`,
      forti: <><strong>FortiClient: host and tunnel name?</strong> In FortiClient, <strong>Remote Access</strong> tab:
        the connection name is the tunnel; the pencil (edit) shows <strong>Remote Gateway</strong>, which is the host.</>,
    },
    // feature:daemon_lifecycle
    connect: {
      title: <>Connect and disconnect</>,
      lead: <>From the icon: click the VPN. From the panel: <strong>Home</strong> → <strong>Connect</strong>.</>,
      bg: <>Kongtrol keeps working in the background. Closing the panel, the terminal or the icon disconnects nothing.
        After your first VPN, Kongtrol asks whether to reconnect the ones you left connected when the computer starts; it waits for internet and never opens a VPN app on its own. You can change it in <strong>Settings</strong> → <strong>When starting</strong>.</>,
      stop: <>To disconnect everything: <strong>Quit and disconnect all</strong> in the icon, or <IC c="kongtrol down --all" />.
        The icon stays with "Kongtrol stopped"; <strong>Start Kongtrol</strong> starts it again.</>,
      code: `$ kongtrol up office           # connect one VPN
$ kongtrol up office aws       # several at once
$ kongtrol status --watch      # live status · q closes the view, VPNs stay up
$ kongtrol tray                # open the tray icon if the tray app is installed
$ kongtrol down office         # disconnect one
$ kongtrol down --all          # disconnect all`,
      waiting: <><strong>"Waiting for connection in …"</strong> means Kongtrol opened your VPN's app and waits for you to connect there.
        Connect in the app; Kongtrol detects it.</>,
      drop: <><strong>If a VPN drops,</strong> Kongtrol reconnects it on its own (up to 3 attempts); if it can't, it tells you once with <strong>Retry</strong> and tries again when your network changes.
        It only tries another VPN if you declared it as a backup (<IC c="failover" />).</>,
    },
    // feature:policies feature:routes
    policies: {
      title: <>What goes through each VPN</>,
      lead: <>A <strong>rule</strong> chooses which destinations use a VPN profile. A <strong>route</strong> is the path your device takes to reach a destination. Other traffic follows the device's existing routes.</>,
      diagram: `  destination          rule                         goes through
  ─────────────────────────────────────────────────────────────────
  claude.ai        ──▶  domains: claude.ai      ──▶  us-content
  10.10.4.12       ──▶  ip_ranges: 10.10.0.0/16 ──▶  office
  example.com      ──▶  no rule                 ──▶  normal internet`,
      panel: <>In the panel, open <strong>Rules</strong> → <strong>＋ Add</strong>. Type a site or a network, pick the VPN and test before saving.
        After saving the first profile, you can stay in the same tab and add a rule for that VPN.
        You can also create the rule in the terminal.</>,
      code: `$ kongtrol policy add github.com --via office
$ kongtrol policy show github.com     # which VPN it uses and why
$ kongtrol policy list
$ kongtrol policy remove <name>`,
      down: <><strong>When the VPN is off,</strong> the profile can allow the regular connection, block known destinations, or block them and try to connect when Kongtrol sees TCP activity. Each rule can choose its own action.
        Use <IC c="kongtrol profile on-tunnel-down office block" /> to change it. Detection can miss quick or rejected attempts. On macOS, connect manually.</>,
      // feature:default_egress
      defaultEgress: <>On Windows or Linux, choose <strong>WireGuard default egress</strong> on a compatible, disconnected direct profile. You can also use <IC c="kongtrol routes check-default office" /> and <IC c="kongtrol routes default-egress office on" />. When connected, the profile carries traffic without a more specific route.</>,
      // feature:vpn_routes
      vpnRoutes: <><strong>Routes each VPN brings.</strong> Below your rules, the dashboard shows the routes that come from the VPN’s file or its server. Kongtrol doesn’t change them: to do that, edit the file or ask your provider.
        The most specific address wins, so a rule for <IC c="10.1.2.3" /> through another VPN wins over <IC c="10.0.0.0/8" /> or over “All traffic” from an .ovpn or .conf file. In the terminal: <IC c="kongtrol routes vpn" />.</>,
      // feature:profile_priority
      order: <><strong>When two rules overlap.</strong> If two rules with different VPNs cover the same site or the same IP range, the VPN with the higher <strong>priority</strong> (the larger number) wins. If neither has one, the rule higher up goes first.
        The dashboard marks those rules (“Priority 10” or “by position”) and says who decided, with a “Change priority” link. In the terminal: <IC c="kongtrol profile priority office 20" /> and <IC c="kongtrol policy list" />.</>,
      yamlLead: <>You can also write them in <IC c="kongtrol.yaml" />, under <IC c="policies:" />:</>,
      yaml: `policies:
  - name: Claude AI
    match:
      domains: [claude.com, "*.claude.com", api.anthropic.com, "*.anthropic.com"]
    via: us-content

  - name: Office network
    match:
      ip_ranges: [10.10.0.0/16]
      domains: ["*.yourcompany.com"]
    via: office
    on_tunnel_down: block   # optional: if office is off, block`,
      match: <><strong>Multiple destinations.</strong> Separate them with commas or line breaks, for example <IC c="claude.com, api.anthropic.com, *.anthropic.com, 192.0.2.42" />. The wildcard covers one level of subdomains. Save the rule paused and activate it when you want it to take effect.
        You can use <IC c="café.example" />. Names also match if you paste uppercase letters or a trailing dot.
        Use <IC c="192.0.2.42" /> for one IP, <IC c="2001:db8::42" /> for IPv6, or <IC c="192.0.2.0/24" /> for a network. The most specific network wins when they overlap.
        App rules (<IC c="apps" />) only explain for now; they don't route.</>,
      apply: <>After editing the file: <IC c="kongtrol config validate" /> and <IC c="kongtrol reload --policy" /> (applies without touching the tunnels).
        Changes in the panel apply by themselves.</>,
    },
    groups: {
      title: <>Groups</>,
      lead: <>A group connects several VPNs with one click. Create it in the panel: <strong>Profiles</strong> → <strong>Groups</strong> tab. Or in the file:</>,
      yaml: `groups:
  work:
    profiles: [office, dev-server, aws]`,
      code: `$ kongtrol up --group work
$ kongtrol down --group work`,
      note: <>A VPN can be in several groups. If you edit the file while Kongtrol runs:{' '}
        <IC c="kongtrol reload" /> applies the changes without a restart.</>,
    },
    faq: {
      title: 'Frequently asked questions',
      items: [
        { question: 'Do I need WireGuard or my VPN provider’s app?', answer: <>A direct WireGuard profile requires your system’s WireGuard tools. If you choose a provider that uses its own app, install that app and connect there when Kongtrol asks.</> },
        { question: 'Can I use several VPNs at once?', answer: <>You can split destinations across compatible profiles. Some apps control the entire connection, DNS, or the firewall. If one VPN fails when you connect another, open <a href="#doctor">Diagnostics</a> to check what your device sees.</> },
        { question: 'What happens if I close the dashboard?', answer: <>Your connections stay active. Open the dashboard or tray to see them. Use <IC c="kongtrol down --all" /> to disconnect all of them.</> },
      ],
    },
    // feature:wizard_doctor feature:direct_diagnostics
    doctor: {
      title: <>Check that all is well</>,
      lead: <>In the panel: <strong>Settings</strong> → <strong>Diagnostics</strong>. It checks your VPNs, files, saved passwords and the helper.
        Each problem says what to do. Diagnostics also warns when two detected VPNs announce full routes for IPv4 or IPv6.</>,
      code: `$ kongtrol dashboard --diagnostics  # open Diagnostics directly
$ kongtrol doctor          # the same in the terminal
$ kongtrol doctor --fix    # and repairs what it can`,
      export: <><strong>Export configuration</strong> (in Diagnostics, or <IC c="kongtrol export" />) gives you a file without passwords,
        useful to ask for help or to set up a teammate's computer.</>,
    },
    // feature:dashboard
    dashboard: {
      title: <>The panel</>,
      lead: <>Open it from the icon (<strong>Open panel</strong>) or with <IC c="kongtrol dashboard" />. It opens already signed in:
        typing the address by hand is not enough, for security. It runs inside your computer; nothing leaves it.</>,
      rows: [
        { name: <>Home</>, desc: <>Your VPNs live (status, IP, time, traffic), connect and disconnect, notices and quick-access groups.</> },
        { name: <>Profiles</>, desc: <>Create, edit, rename and delete VPNs; backup (failover); Groups tab.</> },
        { name: <>Rules</>, desc: <>Which sites and networks use each VPN, with a test before saving; Schedules tab.</> },
        { name: <>Traffic</>, desc: <>What goes through each VPN, where a site goes out, recent connections (in memory only) and the system routes.</> },
        { name: <>Security</>, desc: <>Kill switch and DNS protection, per VPN, and the optional public IP check.</> },
        { name: <>Activity</>, desc: <>Signed log with filters (10 entries and <strong>Show 20 more</strong>; new ones wait behind <strong>Show</strong>), and Kongtrol's notices.</> },
        { name: <>Settings</>, desc: <>Language, theme, diagnostics, new version notice, restart or stop, and delete your data.</> },
      ],
      port: <>The panel's port changes only from the terminal (<IC c="kongtrol config dashboard set-port <port>" />):
        changing it from the panel itself would cut the connection.</>,
    },
    // feature:i18n feature:killswitch feature:dns_guard
    daily: {
      title: <>Day to day</>,
      rows: [
        { cmd: 'kongtrol up --group work', desc: <>Start the day</> },
        { cmd: 'kongtrol down --all', desc: <>Disconnect everything</> },
        { cmd: 'kongtrol status', desc: <>What's connected, kill switch and DNS</> },
        { cmd: 'kongtrol policy show github.com', desc: <>Which VPN a destination uses and why</> },
        { cmd: 'kongtrol ip', desc: <>Your public IP now (if you turned the check on)</> },
        { cmd: 'kongtrol logs', desc: <>Log of connections and notices</> },
        { cmd: 'kongtrol restart', desc: <>Restart Kongtrol; reconnects the VPNs that were up</> },
      ],
      ks: <><strong>Kill switch:</strong> if a VPN drops on its own, it blocks only that VPN's destinations; everything else keeps working.
        If that VPN carried all your traffic, internet stays blocked until it returns, and Kongtrol tells you.
        With <strong>Allow the local network</strong> off (<IC c="allow_lan: false" />) your printer, NAS and other
        devices on your network are cut too; turn it on to keep them.
        If another VPN takes over all your traffic (say you connect Proton VPN in its app), it does not block:
        your traffic keeps going through that VPN. And Kongtrol never connects two VPNs that carry all your
        traffic at once: it tells you which one to disconnect first.
        Turn it on in <strong>Security</strong> or with <IC c="kongtrol config killswitch on" />; per VPN, in the file:</>,
      ksYaml: `vpns:
  office:
    kill_switch: true
  us-content:
    kill_switch: false`,
      sched: <><strong>Schedules</strong> (panel: Rules → Schedules) connect and disconnect by themselves:</>,
      schedYaml: `monitor:
  scheduler:
    enabled: true
    rules:
      - name: office-hours
        profiles: [office]
        weekdays: [mon, tue, wed, thu, fri]
        start: "09:00"
        end: "18:00"`,
      pass: <><strong>Changed your password?</strong> In the panel, edit the profile. In the terminal:{' '}
        <IC c="kongtrol init" /> → "Update credentials for a profile".</>,
      refTitle: <>All commands</>,
      ref: <>The same groups as <IC c="kongtrol --help" />. Each command has examples: <IC c="kongtrol <command> --help" />.</>,
    },
    // feature:killswitch feature:dns_guard
    trouble: {
      title: <>If something fails</>,
      lead: <>First, <strong>Diagnostics</strong> in the panel or <IC c="kongtrol doctor" />: it almost always says what to do.</>,
      items: [
        {
          q: <>No site opens after Kongtrol closed abruptly</>,
          fix: <>Wait about 20 seconds: the helper removes Kongtrol's blocks by itself. If they stay, run{' '}
            <IC c="kongtrol down --all" />. Last resort, remove only Kongtrol's rules
            {os === 'windows' ? <> (never <IC c="netsh advfirewall reset" />: it deletes all your rules)</> : null}:</>,
          code: os === 'windows'
            ? `# PowerShell as administrator
> netsh advfirewall firewall delete rule name=KongtrolBlock
> netsh advfirewall firewall delete rule name=KongtrolKillSwitchBlock`
            : os === 'macos'
              ? `$ sudo pfctl -a com.kongtrol -F all`
              : `$ sudo iptables -F KONGTROL_BLOCK
$ sudo ip6tables -F KONGTROL_BLOCK
$ sudo iptables -F KONGTROL`,
        },
        {
          q: <>A VPN dropped and nothing loads</>,
          fix: <>That VPN carried all your traffic and the kill switch protects it. To browse now, press "Get internet back" in the notice
            (it disconnects it), or create rules
            so only some sites depend on it. If your printer or NAS is unreachable too, turn on
            "Allow the local network" in Security.</>,
          code: `$ kongtrol down <profile>`,
        },
        {
          q: <>Site names don't resolve after an unexpected shutdown</>,
          fix: <>Put your system's DNS back as it was:</>,
          code: os === 'windows'
            ? `> netsh interface ip set dns "Ethernet" dhcp     # or "Wi-Fi"`
            : os === 'macos'
              ? `$ networksetup -setdnsservers Wi-Fi empty`
              : `# with systemd-resolved (Ubuntu, Fedora): the tunnel's name
$ resolvectl revert tun0
# without systemd-resolved
$ sudo cp /etc/resolv.conf.kongtrol.bak /etc/resolv.conf`,
        },
        {
          q: <>Kongtrol can't find my VPN's app</>,
          fix: <>Say where it is: in the panel, edit the profile → advanced options (app path). In the file:</>,
          code: os === 'windows'
            ? `vpns:
  office:
    type: forticlient
    app_path: "C:\\\\Path\\\\FortiClient\\\\FortiClient.exe"`
            : `vpns:
  office:
    type: forticlient
    app_path: "/path/to/FortiClient"`,
        },
        {
          q: <>Wrong or expired password</>,
          fix: <>Save the new one: edit the profile in the panel, or <IC c="kongtrol init" /> → "Update credentials for a profile".</>,
        },
        {
          q: <>"Helper missing" or "helper out of date"</>,
          fix: <><strong>Fix</strong> button in Diagnostics, or <IC c="kongtrol service install" />. It asks for permission once.</>,
        },
        {
          q: <>Cloudflare WARP: "not registered"</>,
          fix: <>Register the device once in the WARP app, or in the terminal:</>,
          code: `$ warp-cli registration new`,
        },
        {
          q: <><IC c="kongtrol" /> is not recognized in the terminal</>,
          fix: <>Open a new terminal. If it persists, <IC c="kongtrol doctor" /> says which copy the terminal uses and whether an old one is in PATH.
            The icon and the panel don't need it.</>,
        },
      ] as GuideFix[],
      files: <><strong>Where your things live:</strong> the configuration in{' '}
        <IC c={os === 'windows' ? '%USERPROFILE%\\.kongtrol\\' : '~/.kongtrol/'} /> (no passwords). Passwords in{' '}
        {os === 'windows' ? 'Windows Credential Manager' : os === 'macos' ? 'the macOS Keychain' : 'the system keyring (libsecret / GNOME Keyring)'}.</>,
    },
    // feature:update_check
    update: {
      title: <>Update</>,
      lead: <>Update the same way you installed. Your configuration, passwords and the helper stay,
        and the VPNs you had connected reconnect.</>,
      how: os === 'windows'
        ? <>Download the new installer and open it: it says "Update Kongtrol" and uses your previous options.</>
        : os === 'macos'
          ? <>Paste the install line again: it says "Updating Kongtrol" and restarts on the new version.</>
          : <>Paste the install line again. With <IC c=".deb" /> or <IC c=".rpm" />: install the new package and run{' '}
            <IC c="kongtrol restart" />.</>,
      notice: <><strong>New version notice (optional, off):</strong> if you turn it on, Kongtrol checks at most once a day
        for a new version and tells you once per version. It never downloads or installs anything. Turn it on in <strong>Settings</strong> or:</>,
      code: `$ kongtrol config update-check on       # turn on
$ kongtrol config update-check off      # turn off: no request at all
$ kongtrol config update-check status`,
      privacy: <>What is asked and to whom: <a href={`${pathFor('privacidad', 'en')}#conexiones`}>privacy policy</a>.</>,
    },
    // feature:uninstall
    uninstall: {
      title: <>Uninstall</>,
      keep: <><strong>Your data stays on this computer:</strong> the <IC c="~/.kongtrol" /> folder (configuration, logs) and the
        passwords in the system keychain. Kongtrol keeps nothing off your computer. If you reinstall, everything is still there.
        It's deleted only if you ask, and that cannot be undone.</>,
      how: os === 'windows'
        ? <>Settings → Apps → Kongtrol → <strong>Uninstall</strong>. It asks whether to delete your data too (default No).
          It disconnects your VPNs, closes Kongtrol and removes the helper.</>
        : <>In the terminal. It disconnects your VPNs, closes Kongtrol and removes the helper. With a <IC c=".deb" />/<IC c=".rpm" /> package,
          it tells you how to finish (<IC c="sudo apt remove kongtrol" /> or <IC c="sudo dnf remove kongtrol" />).</>,
      code: `$ kongtrol uninstall                    # removes the app; your data stays on this computer
$ kongtrol uninstall --purge            # also deletes data and passwords from this computer (asks)
$ kongtrol uninstall --purge --dry-run  # shows what it would do, touches nothing`,
      scriptLead: <>With the install script (add <IC c={os === 'windows' ? '-Purge' : '--purge'} /> to delete your data too):</>,
      script: os === 'windows'
        ? `> & ([scriptblock]::Create((iwr ${UNINSTALL_SCRIPT_BASE}install.ps1 -useb))) -Uninstall`
        : `$ curl -fsSL ${UNINSTALL_SCRIPT_BASE}install.sh | sh -s -- --uninstall`,
      data: <><strong>Only delete your data:</strong> panel → Settings → "Delete all my data", or{' '}
        <IC c="kongtrol uninstall --data-only" />. The app stays and goes back to first run. The icon has no uninstall item, on purpose.</>,
    },
  }
}
