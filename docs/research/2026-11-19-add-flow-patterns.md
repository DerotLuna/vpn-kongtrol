<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# "＋ Agregar": how others add a VPN and send a site through it

Version 1.0.0 · 2026-11-19 · Backs [ADR-0055](../adr/0055-add-sheet-one-entry-point.md)

Access date for every source: 2026-11-19 (fetched with a plain HTTP client; the text quoted is
short and paraphrased where possible).

## Questions

1. Where do mature apps put "add a VPN" and "send this through the VPN / not through it", and how
   many fields does the first step ask for?
2. Do they build the rule from what the user is looking at (a connection, a destination)?
3. What do they match on: apps, domains, IPs/CIDRs, wildcards?

## Findings

| Source | What it shows | Status |
|---|---|---|
| Little Snitch 6, Connection alert — https://help.obdev.at/littlesnitch6/alert-overview | A connection not covered by a rule raises one alert; "answering the alert creates a rule which matches the connection", prefilled from it (process + remote server); the remote can be matched as **domain or host** from a menu on the name; lifetime and profile are secondary options. | verified |
| Little Snitch 6, Connection list — https://help.obdev.at/littlesnitch6/lsm-connection-list | Each row of the live list has a context menu, and rules "can be used to create or delete rules with a single click" from the list. | verified |
| Mullvad split tunneling — https://mullvad.net/en/help/split-tunneling-with-the-mullvad-app | One list in Settings › Split tunneling; "+" next to an app, "Find another app" when it is missing. By **app** only, exclude-only. | verified |
| NordVPN split tunneling — https://support.nordvpn.com/hc/en-us/articles/19618692366865 | Settings › Split tunneling › "Add apps": the list shows the apps that are open, "Browse apps" for others. By **app**; on Linux an "Allowlist" of ports/subnets instead. | verified |
| Proton VPN split tunneling — https://protonvpn.com/support/protonvpn-split-tunneling | Exclude mode / Include mode; apps from a list or "Add" (EXE), and **IP addresses or CIDR ranges** ("46.36.198.0/24"). No domains. | verified |
| Tailscale app connectors — https://tailscale.com/docs/features/app-connectors/how-to/setup | Traffic for named **domains** (`"github.com", "*.github.com"`) goes through a connector node; "any DNS request to the configured app domains triggers route discovery" (routes learned from DNS). | verified |
| UniFi traffic routes — https://help.ui.com/hc/en-us/articles/12566175125783 | Policy-based routes by domain / IP / device through a chosen WAN or VPN, with an optional kill switch. | **no verificado** (the page answered 403 to our client; from prior knowledge only) |
| Microsoft, Add a VPN connection — https://support.microsoft.com/help/20510 | One entry ("Add VPN") in Settings › Network & internet › VPN; the form asks provider, a **connection name you'll recognize**, server name or address, VPN type, sign-in type; the rest is optional and editable later. | verified |

## What we take

- **One entry point, many doors.** Windows has one "Add VPN"; Little Snitch builds rules from
  the alert and from the connection list. Kongtrol: one "＋ Agregar" sheet, opened from the top
  bar and, prefilled, from where the user already looks (a tunnel row, a lookup result, a
  notice). Not invented: the deep link is just the URL of the same sheet.
- **Ask only what is needed first.** Windows asks a recognizable name + server; Mullvad/Nord
  only "which app". Kongtrol: a name (prefilled from the found app) and, for corporate
  clients, the gateway; the rest under "Más opciones".
- **Found apps first.** Nord lists the apps that are open, Mullvad the installed ones.
  Kongtrol already detects installed VPN clients (GET /api/v1/vpns/detect): they come first
  with "Usar".
- **Match on domains with subdomains, IPs/CIDRs.** Tailscale's `domain` + `*.domain` pair and
  Proton's CIDR entry are exactly the policy engine's forms; "incluye subdominios" (default on)
  writes the pair. Apps stay diagnostic-only in Kongtrol (the engine does not steer by app;
  the sheet says so), unlike Mullvad/Nord/Proton.
- **Domain vs host** (Little Snitch) maps to the subdomains checkbox.
- **Where we improve:** a "Hoy / Después" preview from the daemon's own resolver before saving
  (none of the sources shows what changes for that destination), and the same input rules as
  the CLI (`kongtrol policy add`), tested on one shared table.

## Open

- UniFi's page could not be fetched (403); re-check from a browser before citing details.
