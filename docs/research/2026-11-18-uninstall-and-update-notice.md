<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Uninstall that keeps data, and an opt-in update notice

Version 1.0.0 · 2026-11-18 · Backs [ADR-0034](../adr/0034-uninstall-and-update-notice.md)

All sources accessed 2026-11-18.

## Uninstall: keep data unless asked

| Precedent | What it does | Source |
|---|---|---|
| Debian `apt-get remove` vs `purge` | `remove` "leaves its configuration files on the system"; `purge` is remove "and purged (any configuration files are deleted too)". We reuse the word **purge**. | https://manpages.debian.org/bookworm/apt/apt-get.8.en.html |
| Homebrew casks `zap` | "The `zap` procedures will never be performed by default, but only if the user uses `--zap` on `uninstall`". | https://docs.brew.sh/Cask-Cookbook |
| Inno Setup `MsgBox` | Flags include `MB_YESNO` and `MB_DEFBUTTON2`; the docs' example `MB_YESNO or MB_DEFBUTTON2` is "a Yes/No question, defaulting to No". | https://jrsoftware.org/ishelp/topic_isxfunc_msgbox.htm |
| Inno Setup `UninstallSilent` | "Returns True if Uninstall is running silently". Silent uninstall → never delete data. | https://jrsoftware.org/ishelp/topic_isxfunc_uninstallsilent.htm |
| Inno Setup event functions | `InitializeUninstall` ("Return False to abort Uninstall"), `CurUninstallStepChanged` with `usAppMutexCheck, usUninstall, usPostUninstall, usDone`. The page does not describe each step. | https://jrsoftware.org/ishelp/topic_scriptevents.htm |
| Inno Setup `[UninstallRun]` | Entries run "as the first step of uninstallation"; `RunOnceId` runs an entry once. Its order relative to `usUninstall` is **no verificado**, so the purge (`kongtrol uninstall --data-only`) stops the tray and daemon itself and works in either order. | https://jrsoftware.org/ishelp/topic_runsection.htm |

Keychain: `github.com/zalando/go-keyring` offers `Get`, `Set`, `Delete` and
`DeleteAll(service)` but no listing of item names (reasoned from its API at v0.2.8 in
`go.mod`; `DeleteAll` was not used because its behavior per backend is **no verificado** and it
could reach entries that are not ours). So the purge reads profile names from `kongtrol.yaml`,
`config.yaml` and their backups (`kongtrol.yaml.bak-*`) and deletes exactly
`<profile>.username`, `<profile>.password` and `_system.audit_hmac_key` under service
`vpn-kongtrol`.

Package managers own their files: Kongtrol never deletes a binary installed by Homebrew, npm,
Scoop, winget, `.deb`/`.rpm` or AUR; it prints the manager's command (names from
`.goreleaser.yaml`: Scoop `kongtrol`, winget `Kongtrol.Kongtrol`; Homebrew formula `kongtrol`;
npm `kongtrol`).

## Update notice: opt-in, at most once a day

| Fact | Source |
|---|---|
| `GET /repos/{owner}/{repo}/releases/latest` returns "the most recent non-prerelease, non-draft release, sorted by the created_at attribute". | https://docs.github.com/en/rest/releases/releases#get-the-latest-release |
| "All API requests must include a valid `User-Agent` header"; GitHub recommends the application name. Most endpoints expect `Accept: application/vnd.github+json`. | https://docs.github.com/en/rest/using-the-rest-api/getting-started-with-the-rest-api |
| Unauthenticated primary rate limit: 60 requests per hour, per originating IP. One request a day is far below it. | https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api |
| GitHub CLI precedent: checks for a new version "once every 24 hours" and shows a notice; `GH_NO_UPDATE_NOTIFIER` turns it off. Kongtrol goes further: off unless the user turns it on (Local only). | https://cli.github.com/manual/gh_help_environment |
| Semver comparison with `golang.org/x/mod/semver` (Go team module, already in the module graph) instead of our own parser. | https://pkg.go.dev/golang.org/x/mod/semver |

What GitHub sees: the requester's public IP, the time and `User-Agent: kongtrol/<version>`.
GitHub's own handling of that data is governed by its privacy statement
(https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement — not
reviewed in detail, **no verificado**).
