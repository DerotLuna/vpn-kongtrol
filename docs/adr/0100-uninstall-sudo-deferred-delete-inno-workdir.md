<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0100: Uninstall under sudo, deferred self-delete and Inno working directory

Version 1.0.0 · 2026-11-23

Status: Accepted · Date: 2026-11-23 · Decision-makers: Rubén Luna

## Context and Problem Statement

The 2026-11-18 review ([ADR-0040](0040-local-hardening-review-2026-11.md)) left points to verify
([#39](https://github.com/DerotLuna/vpn-kongtrol/issues/39)):

1. `kongtrol uninstall` refuses to run under sudo. Could it resolve the real user instead?
2. Which working directory the Windows uninstaller gives `kongtrol.exe`, and whether a program is
   found by name in a folder others can write.
4. The deferred delete of the running `kongtrol.exe` passed the path to `cmd.exe /c del "<path>"`;
   cmd expands `%VAR%` even inside quotes, so a path with `%` could delete another file.

## Decision Outcome

1. **Keep refusing, detect more ways.** Resolving the user with `SUDO_UID` + `getpwuid` gives the
   right folder, but the process is still root: it would stop every user's tray, fail the daemon
   listener-owner check ([ADR-0062](0062-origin-bound-sessions-token-rotation-listener-owner.md)),
   miss the user's keychain and delete as root inside a folder the user controls. Dropping to the
   user is the same as running without sudo, and the helper removal already asks for permission
   itself. `runViaSudo` now also detects `SUDO_UID`, `DOAS_USER` (doas) and `PKEXEC_UID` (pkexec);
   a root shell of its own (`su -`) is still allowed. The message names sudo, doas and pkexec and
   says what to do (ES + EN).
2. **No change needed.** Every `[Run]`, `[UninstallRun]` and `Exec` entry in
   `packaging/windows/kongtrol.iss` names an absolute program (`{app}`, `{sys}`, `{tmp}`); Inno then
   starts it in that program's folder. `{app}` is the user's own `%LOCALAPPDATA%\Programs\Kongtrol`.
   Go 1.19+ never runs a program from the current directory.
4. **No command text carries the path.** `deleteAfterExit` starts Windows PowerShell from System32
   (full path, working directory System32, `-NoProfile -NonInteractive`) with a constant script.
   The path and the process to wait for are environment variables, read with `$env:` and used only
   with `-LiteralPath`. It waits for the uninstall process to exit, then retries for about 10 s.
   `MoveFileEx(MOVEFILE_DELAY_UNTIL_REBOOT)` was rejected: it needs an administrator.

Items 3 (`#k=` reaching each browser) and 5 (landing guide) are outside this decision: 3 needs a
manual test with Edge, Chrome and Firefox; 5 was already fixed (the guide says `kongtrol dashboard`
or the tray icon).

### Consequences

- Good: a path with `%`, `&`, `^`, `!` or `[ ]` deletes exactly that file (tested).
- Good: the self-delete waits for the process instead of a fixed 3 s sleep.
- Bad: the self-delete needs Windows PowerShell. If it is missing or blocked, the existing
  `cli.uninstall.remove_failed` line tells the user to delete the file by hand.

### Confirmation

`cmd/kongtrol/cmd_uninstall_windows_test.go` (literal path with `%USERNAME%`, wait for a process,
no path on the command line), `TestRunViaSudo` in `cmd/kongtrol/cli_fixes_test.go`.

## More Information

Research and sources: [uninstall under sudo, deferred delete and Inno working directory](../research/2026-11-23-uninstall-sudo-and-deferred-delete.md).
Code: `cmd/kongtrol/cmd_uninstall.go`, `cmd/kongtrol/cmd_uninstall_windows.go`,
`internal/i18n/lifecycle.go`.
