<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: a console program opened with a double click on Windows

Version 1.0.0 · 2026-11-19

## Question

Users download `kongtrol-windows-amd64.exe` (the terminal command) instead of the installer and
double-click it: a console window opens and closes at once, with no message. How do Go CLIs
handle this, and can Kongtrol reuse it with a translated message?

## Findings

- `github.com/inconshreveable/mousetrap` v1.1.0, `StartedByExplorer()`: true when the parent
  process is `explorer.exe` (Windows only; always false elsewhere). Source read in the module
  cache, accessed 2026-11-19: https://github.com/inconshreveable/mousetrap
- cobra v1.8.0 (`go.mod`), `command_win.go` `preExecHook`, accessed 2026-11-19
  (https://github.com/spf13/cobra/blob/v1.8.0/command_win.go): runs on every `ExecuteC` when
  `MousetrapHelpText != ""` and `mousetrap.StartedByExplorer()`. It prints the text, then either
  sleeps `MousetrapDisplayDuration` (default 5 s) or, with duration 0, prints the fixed English
  line `Press return to continue...`, waits for Enter, and exits with code 1.
- cobra 1.8.0 does not look at the arguments: a shortcut that starts `kongtrol.exe dashboard`
  from Explorer would also be stopped (reasoned from the source above; not reproduced).

## Conclusion

Reuse mousetrap (the same check cobra uses) but not cobra's hook: duration 0 adds an English-only
line, which breaks the ES+EN rule. Turn the hook off (`cobra.MousetrapHelpText = ""`) and, only
with no arguments, print the `internal/i18n` text and wait for Enter (EOF also closes).
Not verified by a real double click in this session (**no verificado**); unit tests cover the
text and the argument rule.
