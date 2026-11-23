# Research: Windows setup dashboard locks the installed executable

Sources checked 2026-11-22.

- [Inno Setup event functions](https://jrsoftware.org/inno-setup/documentation/topic_scriptevents.htm)
  specifies that `PrepareToInstall` runs before the files-in-use check and can
  return a message that stops installation.
- [Inno Setup CloseApplications](https://jrsoftware.org/inno-setup/documentation/topic_setup_closeapplications.htm)
  describes Windows Restart Manager's role when replacing files.
- [Inno Setup ExecAndLogOutput](https://jrsoftware.org/inno-setup/documentation/topic_isxfunc_execandlogoutput.htm)
  distinguishes launch errors from exit codes and captures command output in
  the installer log.

During the reported error, the original installed daemon was still running,
the tray had exited, and the user's profile file was absent. The v0.8.0-rc.3
installer ignored the exit code of its fallback command. Both commands went
through the CLI's mandatory profile config load before reaching shutdown.
