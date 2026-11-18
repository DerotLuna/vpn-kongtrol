<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0033: Reestructura de comandos del CLI

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer · Refines
[ADR-0025](0025-desktop-release-channels-first-run.md)

## Context and Problem Statement

The CLI had overlapping commands (`map` and `policy explain` answered the same question;
`routes` and `routes list`; `config defaults show` + `set-group` for one value), a `reload` whose
default did not reload everything and needed `--tunnel` for one profile, and a first run that took
several commands plus `kongtrol service install`. The product is GUI-first and CLI-complete
(AGENTS.md): the CLI must be easy for non-technical users and safe for scripts. Research:
[2026-11-18-cli-command-structure](../research/2026-11-18-cli-command-structure.md).

## Decision Drivers

- Nothing breaks for existing scripts (backward compatibility).
- Every user-visible text translated (ES/EN); deprecation notices too.
- Never wait for input in scripts; never ask with `--json` / `-q`.
- Parity with the dashboard (kill switch / DNS guard toggles).

## Considered Options

1. Keep the commands; only improve help texts.
2. Rename and remove the old names (breaks scripts).
3. **Rename, keep the old names as hidden aliases that print a translated notice on stderr;
   ask yes/no only on an interactive terminal, with `--yes` for scripts.**

## Decision Outcome

Option 3.

| Before (hidden, still works) | Now |
|---|---|
| `kongtrol map` | `kongtrol policy list` · `kongtrol policy show <destino>` |
| `kongtrol policy explain <destino>` | `kongtrol policy show <destino>` |
| `kongtrol routes list` | `kongtrol routes --system` |
| `kongtrol check` | `kongtrol ip` |
| `kongtrol config defaults show` / `set-group <g>` | `kongtrol config default-group [g]` |
| `kongtrol reload --tunnel <p>` | `kongtrol reload <p>` |

- `reload` with no arguments applies everything (policies + connected tunnels restarted in place);
  profiles the daemon does not know need a restart: it asks (default yes), `--yes` restarts, and
  without a terminal it prints `kongtrol restart`.
- First run in one command: `kongtrol init` checks the helper and offers to install it, then asks
  "¿Conectar ahora?" and runs the `up` path; `up` offers the helper when a connect fails because it
  is missing or outdated.
- New `config killswitch on|off|status` and `config dns-guard on|off|status`: through the daemon's
  API when it runs (applied at once), otherwise editing `kongtrol.yaml` keeping comments.
- Deprecation notices use `cli.deprecated` (cobra/pflag print fixed English text).
- Out of scope: scheduler rules in the CLI (next step; dashboard only for now).

### Consequences

- Good: fewer, consistent names (`noun verb`, `on|off|status`); first run is one command.
- Good: scripts keep working; `--json` output unchanged (notices go to stderr; `map --json` keeps
  its old shape).
- Bad: hidden aliases are code to keep until a major version removes them (when: **no verificado**,
  no policy yet).
- Neutral: `reload` with no arguments now restarts every connected tunnel in place (before, only
  tunnels in a group).

### Confirmation

Tests: `cmd/kongtrol/first_run_test.go` (prompt rules, helper offer, init → connect, up retry),
`cmd/kongtrol/cli_restructure_test.go` (aliases hidden + notice, reload → restart with and
without terminal/`--yes`, `config killswitch|dns-guard` via API and via file keeping comments),
`help_test.go`, `gendocs_test.go`.

## More Information

Code: `cmd/kongtrol/{prompt,deprecated,helper_check,cmd_reload,policy_cmd,traffic_map,cmd_routes,cmd_check,cmd_config,cmd_config_security,daemon_restart,wizard}.go`,
`internal/daemonctl/security.go`. Docs: README, `docs/SETUP.md`, AGENTS.md, landing guide,
generated `docs/CLI*.md` / `docs/cli.json`.
