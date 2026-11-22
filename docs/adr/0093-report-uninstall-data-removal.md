# ADR-0093: Report Windows uninstall data removal failures

Status: Accepted · Date: 2026-11-22

The uninstall command skips full configuration loading so it can handle a
broken file. The shutdown path still needs the local API address, so
it reads the on-disk listener settings and machine preferences explicitly,
falling back to defaults when the configuration cannot be read. This prevents
the cleanup from aborting before any data is removed.

Inno Setup runs the data removal before removing the executables. It records
the CLI output in the local uninstall log when logging is enabled. A launch
failure or nonzero exit gets a separate message about user data, with a way to
retry from Settings. The confirmation defaults to keeping data.

During a dashboard reset, preserve the WebView2 runtime folder so the tray
can stay open. WebView2 may hold a lock on it. Saved profiles, preferences and
credentials are still removed. The standalone purge targets the runtime
folder after stopping the tray.

[Research](../research/2026-11-22-uninstall-data-removal.md)
