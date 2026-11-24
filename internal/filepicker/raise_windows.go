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
)

const (
	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpShowWindow = 0x0040
)

var (
	hwndTopmost   = ^uintptr(0)     // HWND_TOPMOST (-1)
	hwndNoTopmost = ^uintptr(0) - 1 // HWND_NOTOPMOST (-2)
)

// raise brings the dialog in front once it appears. The daemon is a
// hidden, detached process: Windows may refuse it the foreground while the
// user works in the browser (SetForegroundWindow rules), so the dialog is
// also moved to the top of the z-order (topmost and straight back, so it
// does not stay above other windows) — visible over the browser even when
// it cannot take the keyboard focus. Only a window of this process with
// the dialog's own title is touched. It stops after 5 s or when stop runs.
func raise(ctx context.Context, title string) func() {
	t, err := windows.UTF16PtrFromString(title)
	if err != nil || title == "" {
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		deadline := time.After(5 * time.Second)
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-deadline:
				return
			case <-tick.C:
			}
			h, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(t)))
			if h == 0 {
				continue
			}
			var pid uint32
			if _, err := windows.GetWindowThreadProcessId(windows.HWND(h), &pid); err != nil || int(pid) != os.Getpid() {
				continue
			}
			flags := uintptr(swpNoMove | swpNoSize | swpShowWindow)
			_, _, _ = procSetWindowPos.Call(h, hwndTopmost, 0, 0, 0, 0, flags)
			_, _, _ = procSetWindowPos.Call(h, hwndNoTopmost, 0, 0, 0, 0, flags)
			_, _, _ = procSetForegroundWindow.Call(h)
			return
		}
	}()
	return func() { close(done) }
}
