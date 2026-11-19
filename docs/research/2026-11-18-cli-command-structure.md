<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: CLI command structure, renames and questions in the terminal

Version 1.0.0 · 2026-11-18

## Question

The Kongtrol CLI grew overlapping commands (`map` vs `policy explain`, `routes` vs `routes list`,
`check`, `config defaults show/set-group`, `reload --tunnel`) and the first run needed several
commands plus an administrator step. How do established CLIs name commands, rename them without
breaking scripts, and ask yes/no questions safely?

## Findings

1. **Interactivity.** clig.dev: "Only use prompts or interactive elements if stdin is an
   interactive terminal (a TTY)"; "If stdin is not an interactive terminal, skip prompting and just
   require those flags/args"; "A common convention is to prompt for the user to type y or yes if
   running interactively, or requiring them to pass -f or --force otherwise" [1]. Kongtrol: ask only
   when stdin and stdout are terminals and not `--json`/`-q`; `--yes` answers for scripts; otherwise
   print the "what to do" line.
2. **Renames.** clig.dev: "interfaces don't change without a lengthy and well-documented
   deprecation process", "Keep changes additive where you can", "Warn before you make a non-additive
   change" [1]. Kongtrol: old names stay as hidden aliases that warn on stderr ("Send messaging to
   stderr" [1]) so `--json` stays parseable.
3. **Naming.** clig.dev: "Be consistent across subcommands", "noun verb seems to be more common",
   "Don't have ambiguous or similarly-named commands", "Don't have a catch-all subcommand" [1].
   GitHub CLI uses `noun list` / `noun view` (`gh pr list`, `gh pr view`) [3][4]; Tailscale has a
   short top-level `tailscale ip` ("Get a device's Tailscale IP address") [5]; systemctl uses
   `enable` / `disable` / `status` verbs on one unit [6]. Kongtrol: `policy list` / `policy show`,
   `ip`, `config killswitch on|off|status`.
4. **cobra deprecation.** `Command.Deprecated` prints `Command %q is deprecated, %s` with fixed
   English text (cobra v1.8.0 `command.go` lines 101-102, 880) [2]; pflag's `MarkDeprecated` is
   similar (**no verificado** in this note; checked in the previous change, `deprecated.go`). Since
   every user-visible string must be translated (AGENTS.md), Kongtrol prints its own `cli.deprecated`.

## Not verified

- How long each project keeps deprecated aliases (no fixed policy found): **no verificado**.

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Command Line Interface Guidelines | https://clig.dev/ | 2026-11-18 |
| 2 | spf13/cobra v1.8.0 — command.go | https://github.com/spf13/cobra/blob/v1.8.0/command.go | 2026-11-18 |
| 3 | GitHub CLI manual — gh pr list | https://cli.github.com/manual/gh_pr_list | 2026-11-18 |
| 4 | GitHub CLI manual — gh pr view | https://cli.github.com/manual/gh_pr_view | 2026-11-18 |
| 5 | Tailscale — CLI reference | https://tailscale.com/kb/1080/cli | 2026-11-18 |
| 6 | systemd — systemctl(1) | https://www.freedesktop.org/software/systemd/man/latest/systemctl.html | 2026-11-18 |

## Used by

- ADR-0033
