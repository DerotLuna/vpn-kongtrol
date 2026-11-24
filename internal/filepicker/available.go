package filepicker

import "github.com/ncruces/zenity"

// available: zenity's own check (always true on Windows and macOS; on
// other Unixes, qarma, zenity or matedialog in PATH, looked up once) and a
// graphical session to show it in.
func available() bool { return hasDesktop() && zenity.IsAvailable() }
