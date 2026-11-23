//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// systemDirectory is %SystemRoot%\System32, where uninstall's helper
// programs are taken from by full path.
func systemDirectory() string {
	sys, err := windows.GetSystemDirectory()
	if err != nil || sys == "" {
		return `C:\Windows\System32`
	}
	return sys
}

// stopTray closes a running tray app (it would otherwise restart the
// daemon or keep its files locked).
func stopTray() {
	_ = exec.Command(filepath.Join(systemDirectory(), "taskkill.exe"), "/f", "/im", "kongtrol-tray.exe").Run()
}

// startDetached starts the Windows installer's uninstaller and returns at
// once, so this kongtrol.exe exits before it deletes the files.
func startDetached(path string) error {
	cmd := exec.Command(path)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008} // DETACHED_PROCESS
	return cmd.Start()
}

// deleteAfterExit removes the running kongtrol.exe once this process exits
// (Windows locks a running executable).
func deleteAfterExit(path string) error {
	return deleteAfterExitCmd(path, os.Getpid()).Start()
}

// Environment variables that carry the delayed delete's inputs.
const (
	deletePathEnv = "KONGTROL_DELETE_PATH"
	deleteWaitEnv = "KONGTROL_WAIT_PID"
)

// deleteAfterExitScript is constant PowerShell: the path never becomes
// part of any command text. It is read from the environment and used only
// with -LiteralPath, where no character is interpreted (cmd.exe expanded
// %VAR% inside the old `del "<path>"` even within quotes). It waits for
// the given process, then retries for up to ~10 s while the image is
// still mapped. No double quotes, so Go's argument quoting stays trivial.
const deleteAfterExitScript = `$p = $env:` + deletePathEnv + `; $w = $env:` + deleteWaitEnv + `; ` +
	`if (-not $p) { exit 2 }; ` +
	`if ($w) { Wait-Process -Id $w -Timeout 30 -ErrorAction SilentlyContinue }; ` +
	`for ($i = 0; $i -lt 20; $i++) { ` +
	`if (-not (Test-Path -LiteralPath $p)) { exit 0 }; ` +
	`Remove-Item -LiteralPath $p -Force -ErrorAction SilentlyContinue; ` +
	`Start-Sleep -Milliseconds 500 }; ` +
	`if (Test-Path -LiteralPath $p) { exit 1 }`

// deleteAfterExitCmd builds the delayed delete (ADR-0100): Windows
// PowerShell by full path under System32, started in System32, no profile,
// with the path and the process to wait for (0 = none) in its environment.
func deleteAfterExitCmd(path string, waitPID int) *exec.Cmd {
	sys := systemDirectory()
	ps := filepath.Join(sys, "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.Command(ps, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", deleteAfterExitScript)
	cmd.Dir = sys
	wait := ""
	if waitPID > 0 {
		wait = strconv.Itoa(waitPID)
	}
	// os/exec keeps the last value of a duplicated variable.
	cmd.Env = append(os.Environ(), deletePathEnv+"="+path, deleteWaitEnv+"="+wait)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x00000008, // DETACHED_PROCESS
	}
	return cmd
}

// removeRunValue deletes HKCU\...\Run "Kongtrol" (written by install.ps1)
// only while it still starts this tray: the Windows installer writes the
// same value for its own copy, and that one is never removed here.
func removeRunValue(tray string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	data, _, err := k.GetStringValue("Kongtrol")
	if err != nil || !runValueIsFor(data, tray) {
		return errRunValueNotOurs
	}
	return k.DeleteValue("Kongtrol")
}

// removeUserPathEntry removes dir from HKCU\Environment\Path (the entry
// install.ps1 added).
func removeUserPathEntry(dir string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	cur, typ, err := k.GetStringValue("Path")
	if err != nil {
		return err
	}
	parts := strings.Split(cur, ";")
	kept := parts[:0]
	removed := false
	for _, p := range parts {
		if p != "" && strings.EqualFold(filepath.Clean(p), filepath.Clean(dir)) {
			removed = true
			continue
		}
		kept = append(kept, p)
	}
	if !removed {
		return errNotInPath
	}
	val := strings.Join(kept, ";")
	if typ == registry.EXPAND_SZ {
		return k.SetExpandStringValue("Path", val)
	}
	return k.SetStringValue("Path", val)
}
