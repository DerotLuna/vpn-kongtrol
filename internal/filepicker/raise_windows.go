package filepicker

import (
	"context"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procIsIconic            = user32.NewProc("IsIconic")
	procShowWindow          = user32.NewProc("ShowWindow")
)

const (
	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoActivate = 0x0010
	swpShowWindow = 0x0040
	swRestore     = 9
)

var hwndTopmost = ^uintptr(0) // HWND_TOPMOST (-1)

// raiseEvery is how often the dialog is put back on top while it is open.
const raiseEvery = 250 * time.Millisecond

// raise keeps the dialog in front for as long as it is open. The daemon
// is a hidden, detached process: Windows refuses it the foreground while
// the user works in the browser (SetForegroundWindow rules), and a dialog
// raised only once (topmost and straight back) dropped behind the browser
// the moment the browser repainted or took a click — it seemed to close at
// once while it was still open, so the next "Elegir archivo" answered
// "busy". So the dialog stays HWND_TOPMOST for its whole life, put back on
// top every raiseEvery (another topmost window may cover it), until stop
// runs or ctx ends; it closes with the dialog, so nothing stays on top
// afterwards. The foreground is asked for once, when the dialog first
// appears (it may still be refused). Only a window of this process with
// the dialog's own title is touched.
func raise(ctx context.Context, title string) func() {
	t, err := windows.UTF16PtrFromString(title)
	if err != nil || title == "" {
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		tick := time.NewTicker(raiseEvery)
		defer tick.Stop()
		first := true
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			h := ownWindow(t)
			if h == 0 {
				continue
			}
			if first {
				bring(h)
				first = false
				continue
			}
			_, _, _ = procSetWindowPos.Call(h, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
		}
	}()
	return func() { close(done) }
}

// bringToFront raises the dialog already open (a second click on "Elegir
// archivo" while it is open): restored if minimized, put on top, and the
// foreground asked for.
func bringToFront(title string) {
	t, err := windows.UTF16PtrFromString(title)
	if err != nil || title == "" {
		return
	}
	if h := ownWindow(t); h != 0 {
		bring(h)
	}
}

func bring(h uintptr) {
	if r, _, _ := procIsIconic.Call(h); r != 0 {
		_, _, _ = procShowWindow.Call(h, swRestore)
	}
	_, _, _ = procSetWindowPos.Call(h, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow)
	_, _, _ = procSetForegroundWindow.Call(h)
}

// ownWindow is the top-level window with exactly this title, only when it
// belongs to this process (0 otherwise).
func ownWindow(title *uint16) uintptr {
	h, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	if h == 0 {
		return 0
	}
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(windows.HWND(h), &pid); err != nil || int(pid) != os.Getpid() {
		return 0
	}
	return h
}
