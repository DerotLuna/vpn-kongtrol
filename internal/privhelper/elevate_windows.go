//go:build windows

package privhelper

import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// shellExecuteInfo mirrors SHELLEXECUTEINFOW (same pattern as
// internal/security/elevate_windows.go).
type shellExecuteInfo struct {
	cbSize         uint32
	fMask          uint32
	hwnd           uintptr
	lpVerb         *uint16
	lpFile         *uint16
	lpParameters   *uint16
	lpDirectory    *uint16
	nShow          int32
	hInstApp       uintptr
	lpIDList       uintptr
	lpClass        *uint16
	hkeyClass      uintptr
	dwHotKey       uint32
	hIconOrMonitor uintptr
	hProcess       windows.Handle
}

var shellExecuteExW = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteExW")

// RunElevated runs exe with args through one UAC prompt ("runas"), waits up
// to timeout and returns the child's exit code. Arguments are quoted with
// windows.EscapeArg, never joined raw.
func RunElevated(exe string, args []string, timeout time.Duration) (uint32, error) {
	verb, _ := windows.UTF16PtrFromString("runas")
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return 0, err
	}
	params, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(args))
	if err != nil {
		return 0, err
	}
	info := shellExecuteInfo{fMask: 0x40 /* SEE_MASK_NOCLOSEPROCESS */, lpVerb: verb, lpFile: file,
		lpParameters: params, nShow: 0 /* SW_HIDE */}
	info.cbSize = uint32(unsafe.Sizeof(info))
	if ret, _, callErr := shellExecuteExW.Call(uintptr(unsafe.Pointer(&info))); ret == 0 {
		if errors.Is(callErr, windows.ERROR_CANCELLED) {
			return 0, ErrElevationCancelled
		}
		return 0, fmt.Errorf("ShellExecuteExW: %w", callErr)
	}
	if info.hProcess == 0 {
		return 0, errors.New("privhelper: elevated process handle missing")
	}
	defer windows.CloseHandle(info.hProcess)
	ev, err := windows.WaitForSingleObject(info.hProcess, uint32(timeout/time.Millisecond))
	if err != nil {
		return 0, err
	}
	if ev == uint32(windows.WAIT_TIMEOUT) {
		_ = windows.TerminateProcess(info.hProcess, 1)
		return 0, fmt.Errorf("privhelper: elevated step timed out after %s", timeout)
	}
	var code uint32
	if err := windows.GetExitCodeProcess(info.hProcess, &code); err != nil {
		return 0, err
	}
	return code, nil
}
