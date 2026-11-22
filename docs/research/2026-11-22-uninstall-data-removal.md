# Research: Windows uninstall and saved profiles

Sources checked 2026-11-22.

- [Inno Setup Exec](https://jrsoftware.org/inno-setup/documentation/topic_isxfunc_exec.htm)
  distinguishes failure to launch from the process exit code.
- [Inno Setup ExecAndLogOutput](https://jrsoftware.org/inno-setup/documentation/topic_isxfunc_execandlogoutput.htm)
  captures output from a hidden console command in the uninstall log.
- [Inno Setup UninstallRun](https://jrsoftware.org/inno-setup/documentation/topic_runsection.htm)
  describes its generic remaining-elements warning. It does not report the
  status of Kongtrol's separate profile and credential removal.
- [Microsoft WebView2 process lifecycle](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/process-related-events#clearing-the-user-data-folder)
  explains that the user data folder stays locked until the browser process
  exits. In the reported dashboard reset, profile files were already gone,
  with a WebView2 cache journal still open.

The installed command could locate the user's existing configuration. The
uninstall command skips full config loading by design, but the shared shutdown
path dereferenced that config once it found a running daemon. The previous
Inno hook discarded the launch result and exit code.
