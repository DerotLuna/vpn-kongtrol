//go:build windows

package main

import (
	"os/exec"
	"path/filepath"
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

// deleteAfterExit removes the running kongtrol.exe a few seconds after this
// process exits (Windows locks a running executable).
func deleteAfterExit(path string) error {
	return deleteAfterExitCmd(path).Start()
}

// deleteAfterExitCmd builds the delayed delete. cmd.exe looks up a bare
// "ping" in the current directory first, so both programs are given by
// full path under System32 and the working directory is System32 too.
func deleteAfterExitCmd(path string) *exec.Cmd {
	sys := systemDirectory()
	shell := filepath.Join(sys, "cmd.exe")
	ping := filepath.Join(sys, "PING.EXE")
	cmd := exec.Command(shell)
	cmd.Dir = sys
	// Raw command line: Go's argument escaping (\") is not cmd.exe's. /d
	// skips AutoRun; the outer quotes keep cmd's /c quote stripping from
	// eating the inner ones.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x00000008, // DETACHED_PROCESS
		CmdLine:       `"` + shell + `" /d /c ""` + ping + `" -n 4 127.0.0.1 >nul & del /f /q "` + path + `""`,
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
