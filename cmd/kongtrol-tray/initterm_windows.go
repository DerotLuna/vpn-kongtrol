//go:build windows

package main

import (
	"errors"
	"os/exec"
	"strings"
	"syscall"
)

const createNewConsole = 0x00000010

// canRunInit reports whether the tray can open a terminal running
// `kongtrol init` on this OS.
const canRunInit = true

// runInitInTerminal opens a new console window running `kongtrol init`
// (cmd /k keeps it open afterwards so the user can read the result).
// exe comes from daemonctl.FindKongtrol; a path with characters cmd.exe
// would interpret is refused rather than escaped.
func runInitInTerminal(exe string) error {
	if exe == "" || strings.ContainsAny(exe, "\"%^&|<>") {
		return errors.New("unsafe kongtrol path for cmd.exe")
	}
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// /k with an outer pair of quotes: cmd strips the first and last
		// quote, leaving `"<exe>" init`.
		CmdLine:       `cmd.exe /k ""` + exe + `" init"`,
		CreationFlags: createNewConsole,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
