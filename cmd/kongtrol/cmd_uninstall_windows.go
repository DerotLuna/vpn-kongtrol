//go:build windows

package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

// stopTray closes a running tray app (it would otherwise restart the
// daemon or keep its files locked).
func stopTray() {
	_ = exec.Command("taskkill", "/f", "/im", "kongtrol-tray.exe").Run()
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
	cmd := exec.Command("cmd.exe", "/c", "ping -n 4 127.0.0.1 >nul & del /f /q \""+path+"\"")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008}
	return cmd.Start()
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
