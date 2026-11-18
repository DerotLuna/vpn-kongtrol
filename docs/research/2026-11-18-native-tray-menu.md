<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: what a native tray menu can show with fyne.io/systray

Version 1.1.0 · 2026-11-18 (1.1.0: simplified tray glyph)

## Question

User report on v0.6.0-rc.3 (Windows 11): the tray menu showed raw English states
(`brainbox [connected]`), plain text items, no version, and an outdated helper stayed outdated
after upgrading. What can the native menu do per OS with the library we ship
(`fyne.io/systray` v1.11.0), how do we pick a tray icon that fits the taskbar, and how do other
tools handle "another copy of the CLI earlier in PATH"?

## Findings

### fyne.io/systray v1.11.0 (read from the module source, accessed 2026-11-18)

Files: `systray.go`, `systray_windows.go`, `systray_darwin.go`/`.m`, `systray_unix.go`,
`systray_menu_unix.go` in `$GOMODCACHE/fyne.io/systray@v1.11.0`.

| Feature | Windows | macOS | Linux (SNI + dbusmenu) |
| --- | --- | --- | --- |
| Tray icon `SetIcon` | `.ico` only: `LoadImage(IMAGE_ICON, LR_LOADFROMFILE \| LR_DEFAULTSIZE)` from a temp file (a PNG fails) | any NSImage format, forced to 16×16 pt | `IconPixmap`, decoded with `image.Decode` + `image/png` only (an `.ico` is **not** decodable) |
| `SetTemplateIcon` | falls back to `SetIcon` | NSImage `template` (menu bar tints it) | falls back to `SetIcon` |
| Menu item `SetIcon` | yes: `.ico` → `iconToBitmap` at `SM_CXSMICON` → `MENUITEMINFO.hbmpItem` | yes (16 pt) | yes: dbusmenu `icon-data` (PNG bytes) — the doc comment says "macOS and Windows only", the code sets it on Linux too |
| Checkbox items (`AddSubMenuItemCheckbox`, `Check`) | `MFS_CHECKED` | check mark | dbusmenu toggle |
| Separators, `Disable`, `Hide` | yes | yes | yes |
| Item tooltips | **no** (Win32 menus have no item tooltips; only the tray icon tooltip) | yes | depends on the host |
| `SetTitle` (text next to the icon) | no-op | menu bar text | SNI `Title` |
| Reorder / insert items later | no API: items are appended; only title/state change | same | same |
| Click delivery | non-blocking send on `ClickedCh` (a goroutine must be waiting) | same | same |

Consequences: the VPN list is a fixed set of slots relabelled on every refresh (connected
first); state is said in words in the title (tooltips don't exist on Windows); the tray icon is
`.ico` on Windows and PNG elsewhere, chosen with build tags.

### Win32 menus with a check mark and a bitmap

`MENUITEMINFO.hbmpItem` draws the bitmap in its own column next to the check-mark column unless
the menu has `MNS_CHECKORBMP` (systray does not set it), so a checked item shows both.
https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-menuiteminfow and
https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-menuinfo — read from
memory, **no verificado** (not reopened in this session); behavior not checked on a screen.

### Taskbar light/dark theme on Windows

`HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, DWORD
`SystemUsesLightTheme` (1 = light taskbar/Start; `AppsUseLightTheme` is for app windows).
Present on Windows 10 1903+ and Windows 11; widely read by tray apps. Microsoft documents the
app-mode setting through `UISettings`, not this value: **no verificado** in official docs.
Kongtrol reads it on every refresh and falls back to the dark taskbar (Windows 11 default).

### macOS template images

A template image uses only its alpha channel; AppKit tints it for the menu bar appearance
(`NSImage.isTemplate`, https://developer.apple.com/documentation/appkit/nsimage/istemplate —
**no verificado** in this session). Color is lost, so a state badge on macOS is a shape only.

### Linux

StatusNotifierItem has no "panel is light/dark" property
(https://www.freedesktop.org/wiki/Specifications/StatusNotifierItem/ — **no verificado** in
this session). Symbolic icons on the common dark panels (GNOME top bar) are light; Kongtrol
uses the light ink there and lets the badge carry the state.

### Rendering the mark from the SVG master

`assets/brand/source/logo-kong.svg` is potrace output: one `<g transform="translate() scale()">`
with `M/m L/l C/c Z/z` paths (SVG 2 path grammar,
https://www.w3.org/TR/SVG2/paths.html#PathDataBNF). `golang.org/x/image/vector` (already in
`go.mod`) fills such outlines with anti-aliasing; potrace emits holes with opposite winding, so
the rasterizer's accumulation keeps them open. Rendering each tray size from the vector (not
downscaling a 512 px raster) and keeping the background transparent avoids the filled-disc look.
At 16 px the concentric rings of the mark blur into grey; a simplified small-size master would
be a design task (not done here).

### A simplified tray glyph (1.1.0)

User feedback on Windows 11 (dark taskbar): the line-art mark turns into a grey blob at 16 px
next to Windows' solid glyphs. Platform guidance (all **no verificado** in this session — cited
from prior reading, to re-open before the final ADR):

- Microsoft, app icon construction
  (https://learn.microsoft.com/en-us/windows/apps/design/style/iconography/app-icon-construction):
  provide each target size (16, 20, 24, 32, 40, 48 … for 100–300 % scaling), simplify detail at
  small sizes, keep shapes on the pixel grid ("pixel snapping") instead of scaling one raster.
- Microsoft, notification area
  (https://learn.microsoft.com/en-us/windows/win32/shell/notification-area): tray icons are
  drawn at the small-icon size (`SM_CXSMICON`, 16 px at 100 %); Windows 11's own tray glyphs are
  monochrome and follow the taskbar theme.
- Apple HIG, menu bar extras
  (https://developer.apple.com/design/human-interface-guidelines/the-menu-bar): use a template
  image (black + alpha) so the system tints it; keep it simple, about 16 pt tall.

Three candidates in `assets/brand/tray-candidates/` (24-unit grid SVG, nonzero winding, single
source; `generate.go` renders 16/20/24/32/40/48 px with an edge-contrast "hint" at ≤32 px —
an approximation, not true grid-fitting): **A · silueta** (solid head and ears, face cut out),
**B · anillo** (one bold ring + solid face, echoes the contour rings), **C · cara** (the face
plate alone). Previews on dark/light taskbar strips at 16 and 24 px were made for the choice;
the full line-art logo stays for large sizes (installer, about, dashboard).

### Another copy of the CLI earlier in PATH

Windows builds the process PATH as the system PATH followed by the user PATH (documented under
"Environment Variables", https://learn.microsoft.com/en-us/windows/win32/procthread/environment-variables
— **no verificado** in this session); the first match wins. Homebrew's `brew doctor` has checks
that warn when other binaries shadow Homebrew's (`Library/Homebrew/diagnostic.rb`, e.g.
`check_user_path_1`) — the same idea as Kongtrol's doctor line. Inno Setup remembers the
previous run's task choices (`UsePreviousTasks=yes` by default,
https://jrsoftware.org/ishelp/topic_setup_useprevioustasks.htm), so removing `unchecked` from the
PATH task changes new installs; an upgrade keeps what the user chose before.

## Decision input

- ADR-0036
