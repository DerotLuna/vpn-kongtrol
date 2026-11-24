# Research: native file picker and OpenVPN files that name other files

Version 1.0.0 · 2026-11-24

## Question

An imported `.ovpn` that names sibling files (`ca ca.crt`, `tls-auth ta.key 1`) was refused with
`api.err.import_file_ref`: the browser gives the page only a file name and its content, never a
path, so the daemon cannot find `ca.crt`. How do other tools solve it, what is the standard way to
get a real path from a local web panel, and does an OS dialog opened by Kongtrol's hidden daemon
come to the front?

## Sources

| Source | Accessed |
|---|---|
| [OpenVPN 2.6 reference manual (`--cd`, inline files, `key-direction`)](https://openvpn.net/community-resources/reference-manual-for-openvpn-2-6/) | 2026-11-24 |
| [Microsoft: SetForegroundWindow (foreground rules)](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setforegroundwindow) | 2026-11-24 |
| [pkg.go.dev: github.com/ncruces/zenity v0.10.15 (MIT)](https://pkg.go.dev/github.com/ncruces/zenity) | 2026-11-24 |
| zenity v0.10.15 source (`file_windows.go`, `util_windows.go`, `internal/zenutil/run_unix.go`, `run_darwin.go`), read from the Go module cache | 2026-11-24 |
| Kongtrol code: `internal/privops/ovpnconf.go` (`SanitizeOpenVPNConfig`, `ovpnFileRefs`, `inlineBlock`), `internal/privops/local.go` (`OpenVPNStart` resolves references next to the config) | 2026-11-24 |

## Findings

### How OpenVPN and its GUIs find referenced files

- OpenVPN reads key and certificate files relative to its working directory; `--cd dir` changes
  to `dir` "prior to reading any files such as configuration files, key files". OpenVPN GUI on
  Windows starts openvpn with `--cd` set to the config's folder, so `ca ca.crt` next to the
  `.ovpn` works. Kongtrol's adapter does the same at connect time (`--cd <config dir>`).
- Every file directive can be inlined instead (`<ca>…</ca>`), and `key-direction` carries the
  direction that `tls-auth file 1` would give. Kongtrol's helper already turns references into
  inline blocks (`SanitizeOpenVPNConfig` with a reader) so openvpn never opens a user path.
- So the fix needs no new format: give the existing sanitizer a reader at import time and store the
  self-contained result.

### Getting a real path from a local web panel

- Browsers never expose a picked file's path to a page (the File API gives name, size and
  content). The File System Access API's `showOpenFilePicker` also hides paths and is
  Chromium-only.
- Desktop apps built on a web view (Electron `dialog.showOpenDialog`, Tauri's dialog plugin, Wails
  `OpenFileDialog`; known from their docs, not re-fetched for this note) ask the native side to
  open the OS dialog and get the path back. Kongtrol's
  native side is the daemon (or the tray); the same pattern fits: an authenticated local endpoint
  opens the dialog and returns the path.
- Library: `github.com/ncruces/zenity` (MIT, listed on Awesome Go). Verified in its source and
  docs: no cgo; Windows uses the Common Item Dialog (`IFileOpenDialog`) in process, with
  `GetOpenFileName` as fallback; macOS runs `osascript -l JavaScript`; other Unixes run `qarma`,
  `zenity` or `matedialog` (first found in `PATH`, looked up once); `IsAvailable()` is always
  true on Windows and macOS; `Context` closes the dialog and returns `ctx.Err()`; `ErrCanceled`
  on cancel; `FileFilter{Name, Patterns, CaseFold}`. A `CGO_ENABLED=0` Windows build of a probe
  and of `cmd/kongtrol` succeeded. Not covered: KDE without zenity (no kdialog), and the XDG
  desktop portal (`org.freedesktop.portal.FileChooser`), which would need a D-Bus client of our
  own. Those sessions fall back to the browser's input, which now takes several files at once.

### Foreground from the hidden, detached daemon (Windows)

- Microsoft's rules: a process may set the foreground window only if, among others, it is the
  foreground process, was started by it, received the last input event, or the foreground lock
  timeout expired; otherwise "an application cannot force a window to the foreground while the
  user is working with another window. Instead, Windows flashes the taskbar button".
- When the user clicks "Elegir archivo" the browser is the foreground process. Neither the daemon
  (detached, `CREATE_NO_WINDOW`) nor the tray started the browser or received that click, so by
  the documented rules both are in the same position: routing the dialog through the tray does not
  gain the foreground right. The browser cannot call `AllowSetForegroundWindow` for us.
- zenity itself calls `SetForegroundWindow` on a window of its own process before showing the
  dialog; the daemon has none, so that step does nothing there.
- Mitigation used: once the dialog appears, Kongtrol moves it to the top of the z-order
  (`SetWindowPos HWND_TOPMOST` then `HWND_NOTOPMOST`, which the foreground rules do not cover) and
  tries `SetForegroundWindow`. The panel says "Elige el archivo en la ventana que se abrió (si no
  la ves, búscala en la barra de tareas)".
- Probe (2026-11-24, Windows 11 Pro 10.0.26300, one bounded run, each dialog closed itself after
  4 s): a process started with `DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW`
  (like the daemon) opened the dialog; it was visible and was the foreground window, both without
  and with the z-order step (`SetWindowPos` and `SetForegroundWindow` returned success). Caveat:
  the probe was started by an automated session while the owner was not clicking in a browser, so
  the foreground lock may have expired or the starter chain may have counted; it does not prove
  the case "the user just clicked in the browser". **No verificado** for that case: manual test
  by the owner (below).

### Manual test (owner)

1. Run a build of this branch; open the panel in the default browser from the tray.
2. Agregar › Importar un archivo de VPN › Elegir archivo, clicking quickly so the browser is
   clearly the foreground window.
3. Expected: the "Kongtrol · Elige el archivo de tu VPN" dialog appears above the browser (with
   keyboard focus, or at least visible on top with a flashing taskbar button).
4. Pick an `.ovpn` that names `ca.crt` in the same folder; Guardar y activar must succeed and
   "Más opciones" lists "Guardados dentro del perfil: ca.crt".
5. Repeat on macOS and on a Linux desktop with zenity installed, and once on KDE without zenity
   (expected: the browser's file picker opens instead).

## Confidence

High for OpenVPN behaviour and for zenity's platform support (primary docs plus source read).
Medium for the Windows foreground behaviour: documented rules plus one probe in conditions that
may not match a real click. Low (unverified) for macOS and Linux dialog placement.

## Open questions

- Whether the z-order step is enough on Windows when the user is active in the browser (manual
  test above).
- XDG portal support for KDE/Flatpak sessions without zenity: would need a small D-Bus client
  (`godbus/dbus` is already a dependency); left out until users ask.

## Used by

- ADR-0105
