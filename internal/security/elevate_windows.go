//go:build windows

package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// shellExecuteInfo mirrors the Win32 SHELLEXECUTEINFOW struct.
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

var (
	shell32         = syscall.NewLazyDLL("shell32.dll")
	shellExecuteExW = shell32.NewProc("ShellExecuteExW")
)

const (
	seeMaskNocloseprocess = 0x00000040
	swHide                = 0
)

// runElevated runs exe with args via ShellExecuteExW using the "runas" verb,
// triggering a UAC prompt if the process is not already elevated.
// Waits for the child to exit and returns an error if it exits non-zero.
func runElevated(exe string, args ...string) error {
	verb, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	params, err := syscall.UTF16PtrFromString(strings.Join(args, " "))
	if err != nil {
		return err
	}

	info := shellExecuteInfo{
		fMask:        seeMaskNocloseprocess,
		lpVerb:       verb,
		lpFile:       file,
		lpParameters: params,
		nShow:        swHide,
	}
	info.cbSize = uint32(unsafe.Sizeof(info))

	ret, _, callErr := shellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return fmt.Errorf("ShellExecuteExW: %w", callErr)
	}
	if info.hProcess == 0 {
		return nil
	}
	defer windows.CloseHandle(info.hProcess)

	if _, err := windows.WaitForSingleObject(info.hProcess, windows.INFINITE); err != nil {
		return fmt.Errorf("wait: %w", err)
	}

	var exitCode uint32
	if err := windows.GetExitCodeProcess(info.hProcess, &exitCode); err != nil {
		return fmt.Errorf("exit code: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("exit status %d", exitCode)
	}
	return nil
}

// runNetshElevated executes multiple netsh commands in a single elevated
// PowerShell process, requiring only one UAC prompt.
func runNetshElevated(commands [][]string) error {
	if len(commands) == 0 {
		return nil
	}

	// Build a PowerShell script with all netsh commands.
	// Use SilentlyContinue so delete-rule commands don't abort on "not found".
	var script strings.Builder
	script.WriteString("$ErrorActionPreference = 'SilentlyContinue'\n")
	for _, args := range commands {
		script.WriteString("& netsh ")
		for i, a := range args {
			if i > 0 {
				script.WriteString(" ")
			}
			// Quote args that contain spaces or = signs.
			if strings.Contains(a, " ") || strings.Contains(a, "=") {
				fmt.Fprintf(&script, "'%s'", strings.ReplaceAll(a, "'", "''"))
			} else {
				script.WriteString(a)
			}
		}
		script.WriteString("\n")
	}

	// Write to temp file — PowerShell -File is more reliable than -Command
	// for multi-line scripts with special characters.
	tmpDir, err := os.MkdirTemp("", "kongtrol-elev-*")
	if err != nil {
		return fmt.Errorf("elevate: temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "netsh_commands.ps1")
	if err := os.WriteFile(scriptPath, []byte(script.String()), 0600); err != nil {
		return fmt.Errorf("elevate: write script: %w", err)
	}

	return runElevated("powershell.exe",
		"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath)
}

// isElevated returns true if the current process is running with admin privileges.
func isElevated() bool {
	token := windows.GetCurrentProcessToken()
	return token.IsElevated()
}
