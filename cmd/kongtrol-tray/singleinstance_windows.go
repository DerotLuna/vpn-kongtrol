//go:build windows

package main

import (
	"log"

	"golang.org/x/sys/windows"
)

// acquireSingleInstance returns false if another kongtrol-tray is already
// running. Uses a named kernel mutex, which the OS automatically releases if
// the holding process dies or is killed — no stale-lock-file cleanup needed.
func acquireSingleInstance() bool {
	name, err := windows.UTF16PtrFromString(`Global\KongtrolTraySingleInstance`)
	if err != nil {
		log.Printf("kongtrol-tray: single-instance check: %v", err)
		return true // fail open — don't block startup over this
	}
	_, err = windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		return false
	}
	if err != nil {
		log.Printf("kongtrol-tray: single-instance check: %v", err)
	}
	return true
}
