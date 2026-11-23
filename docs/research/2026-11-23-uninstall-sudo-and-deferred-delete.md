<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: uninstall under sudo, deferred delete and Inno working directory

Version 1.0.0 · 2026-11-23

Sources checked 2026-11-23. Backs [ADR-0100](../adr/0100-uninstall-sudo-deferred-delete-inno-workdir.md);
issue [#39](https://github.com/DerotLuna/vpn-kongtrol/issues/39), items 1, 2 and 4.

## 1. `HOME` when root acts for another user

- [sudoers(5)](https://www.sudo.ws/docs/man/sudoers.man/): with `env_reset` (the default) "The HOME,
  MAIL, SHELL, LOGNAME and USER environment variables are initialized based on the target user";
  `always_set_home` "has no effect unless the env_reset flag has been disabled or HOME is present
  in the env_keep list".
- [sudo(8)](https://www.sudo.ws/docs/man/sudo.man/): `SUDO_UID` "Set to the user-ID of the user who
  invoked sudo", `SUDO_USER` "the login name of the user who invoked sudo".
- Ubuntu kept the caller's `HOME` until 19.10 (patch adding `HOME` to `env_keep`), then followed
  upstream: [Launchpad #1556302](https://bugs.launchpad.net/bugs/1556302). Debian and Fedora use the
  upstream default (not re-checked on a live system: **no verificado**).
- macOS ships `Defaults env_keep += "HOME MAIL"` in `/etc/sudoers`
  ([Apple open source, sudo-77](https://opensource.apple.com/source/sudo/sudo-77/src/sudoers.auto.html);
  a copy of the macOS 14 file: [gist](https://gist.github.com/keith/9061156)), so `HOME` stays the
  caller's there.
- [doas(1)](https://man.openbsd.org/doas.1): "DOAS_USER is set to the name of the user executing
  doas"; `HOME` is "set to values appropriate for the target user".
- [pkexec(1)](https://www.freedesktop.org/software/polkit/docs/latest/pkexec.1.html): a "minimal known
  and safe environment"; "PKEXEC_UID ... is set to the user id of the process invoking pkexec".

Reasoned (not a source): `SUDO_UID` + `getpwuid` would give the right home folder, but the
process is still root. It would stop every user's tray (`pkill -x`), fail the daemon client's
listener-owner check (ADR-0062: the daemon belongs to the user, not root), reach root's keychain
or none (Secret Service lives on the user's D-Bus session, which `env_reset` drops; macOS login
keychain per user session: **no verificado**), and delete as root inside a folder the user
controls (links and races there are the user's to plant, a problem when sudoers grants only this
command). Dropping to the user first makes it the same as running without sudo. The only step that
needs root, removing the helper, already asks for it on its own.

## 2. Inno Setup working directory

- [[Run] / [UninstallRun]](https://jrsoftware.org/ishelp/topic_runsection.htm) `WorkingDir`: "If
  this parameter is not specified or is blank, it uses the directory from the Filename parameter".
- [Exec](https://jrsoftware.org/ishelp/topic_isxfunc_exec.htm): with an empty `WorkingDir` it "will try
  to extract a pathname from the Filename parameter and use that as the initial current directory".
- [os/exec](https://pkg.go.dev/os/exec#hdr-Executables_in_the_current_directory) (Go 1.19+):
  `LookPath` no longer returns a program from the current directory (`ErrDot`).

Every entry in `packaging/windows/kongtrol.iss` names `{app}\kongtrol.exe`, `{sys}\taskkill.exe` or
`{tmp}\kongtrol.exe`, so the working directory is `{app}` (`%LOCALAPPDATA%\Programs\Kongtrol`, only
this user can write it, the same user who could already replace `kongtrol.exe`), `{sys}` or Setup's
own temp folder.

## 3. Deleting the running `kongtrol.exe`

- Reproduced 2026-11-23 on Windows 11: `cmd.exe /d /c "del /f /q "<dir>\a %USERNAME%\k.exe""`
  deleted `<dir>\a <user>\k.exe` and left the named file: cmd expands `%VAR%` inside quotes.
- [MoveFileExW](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw)
  `MOVEFILE_DELAY_UNTIL_REBOOT` "can be used only if the process is in the context of a user who
  belongs to the administrators group or the LocalSystem account" (writes HKLM
  `PendingFileRenameOperations`): not usable by a per-user uninstall.
- [Remove-Item](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.management/remove-item)
  `-LiteralPath`: "used exactly as it's typed. No characters are interpreted as wildcards."
- [about_PowerShell_exe](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_powershell_exe):
  `-NoProfile` "Doesn't load the PowerShell profile"; `-NonInteractive` turns prompts into errors.

Chosen: Windows PowerShell by full path, constant script, the path and the process id passed in
environment variables and used only through `$env:` + `-LiteralPath`. The repository already runs
it this way for tray notifications and FortiClient automation.
