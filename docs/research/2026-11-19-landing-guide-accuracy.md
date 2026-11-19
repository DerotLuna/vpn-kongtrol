<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: setup guide accuracy and step format

Version 1.0.0 · 2026-11-19

## Question

For ADR-0061: which statements on the landing guide (`/guia`, `/en/guide`) no longer match the
product, and how should install steps be written so a non-technical user knows what to do and
what they should see.

## Sources

| # | Source | URL / path | Accessed |
|---|---|---|---|
| 1 | Microsoft Writing Style Guide — Writing step-by-step instructions: numbered list, one instruction per step, say where the action happens first, start with an imperative verb, fit the steps on one screen | https://learn.microsoft.com/en-us/style-guide/procedures-instructions/writing-step-by-step-instructions | 2026-11-19, verified (fetched) |
| 2 | Cloudflare WARP client on Linux: "Register the client `warp-cli registration new`" | https://developers.cloudflare.com/warp-client/get-started/linux/ | 2026-11-19, verified (fetched); the page does not mention the old `warp-cli register` |
| 3 | Release assets and names | `landing/src/links.ts`, `.goreleaser.yaml` | 2026-11-19, read |
| 4 | Install scripts: macOS app to `~/Applications`, CLI to `/usr/local/bin` or `~/.local/bin` without sudo on a fresh install, opens Kongtrol at the end, helper only updated when outdated | `install.sh`, `install.ps1` | 2026-11-19, read |
| 5 | Windows installer: per user, "Install the network helper" task (one UAC prompt), "Open Kongtrol and create my first VPN" opens `/profiles` | `packaging/windows/*.iss` | 2026-11-19, read |
| 6 | Wizard flow: language never asked, profiles → policies → security → review, then helper offer and "Connect now?"; an existing config opens an action menu (`menu.*`) | `cmd/kongtrol/wizard*.go`, `internal/i18n/i18n.go` | 2026-11-19, read |
| 7 | Tray and panel labels used in the guide (`tray.helper_install`, `tray.header.stopped`, `tray.start`, `attach.waiting_user`, `diag.fix`, `diag.export`, `nav.*`, `add.*`, `profiles.action.save`) | `internal/i18n/i18n.go`, `web/ui/src/i18n/**` | 2026-11-19, read |
| 8 | Firewall names for manual recovery: `KongtrolBlock`, `KongtrolKillSwitchBlock`, `KONGTROL_BLOCK`, `KONGTROL`, pf anchor `com.kongtrol`; `/etc/resolv.conf.kongtrol.bak` | `internal/privops/*.go` | 2026-11-19, read |
| 9 | Supported VPN types and modes | `internal/config/schema.go` (`oneof`), README "Supported VPNs" | 2026-11-19, read |

## Findings: what was stale

- Windows install told users to download `kongtrol_windows_amd64.zip`, unzip it to `C:\tools`
  and edit PATH by hand; the product ships `Kongtrol-Setup-windows-amd64.exe` (per user, helper
  task, opens the panel). macOS/Linux used `curl … kongtrol_<os>_<arch>.tar.gz | tar xz` +
  `sudo mv`; the documented path is the one-line script from `INSTALL_CMD` (and `.deb`/`.rpm`).
- The wizard section described a language question (removed: the language comes from the
  system), numbered `Elige [1]:` prompts (the wizard uses interactive menus) and long sample
  transcripts that no longer match the strings.
- Sample `doctor` and `status` outputs used labels that are not the current ones.
- The dashboard section listed raw API endpoints; they need the session the panel gets, so a
  user cannot call them as shown, and they are not a user task.
- `warp-cli register` (source 2: now `warp-cli registration new`).
- Typo "kongtrolar", "la tabla de rutas del equipo las conexiones" (missing comma).

## Findings: step format

Source 1 matches the AGENTS.md "User guidance" rule: each step is one imperative instruction,
says where it happens, and is followed by what the user sees ("Verás" / "You'll see"), so
they can tell it worked. Problems use the same shape: symptom → one "Qué hacer" line → an
optional command per OS.

## Not changed here (follow-up)

- `internal/vpn/cloudflarewarp/cli.go` still tells users `warp-cli register` (source 2).
- `docs/SETUP.md` §11 still suggests `warp-cli register`, `netsh advfirewall reset` (deletes
  every firewall rule) and `sudo pfctl -d` (turns pf off for everyone); the guide uses the
  narrow commands from source 8.
