//go:build windows

package autostart

import (
	"errors"
	"time"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey      = `Software\Microsoft\Windows\CurrentVersion\Run`
	approvedKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	// valueName is the Run value kongtrol.iss and install.ps1 write.
	valueName = "Kongtrol"
)

func readRun() (string, bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return "", false
	}
	defer k.Close()
	v, _, err := k.GetStringValue(valueName)
	if err != nil {
		return "", false
	}
	return v, true
}

func readApproved() []byte {
	k, err := registry.OpenKey(registry.CURRENT_USER, approvedKey, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()
	v, _, err := k.GetBinaryValue(valueName)
	if err != nil {
		return nil
	}
	return v
}

// Status reads the Run value and its StartupApproved switch.
func Status() State {
	_, hasRun := readRun()
	_, hasTray := TrayPath()
	return State{Supported: hasRun || hasTray, Enabled: hasRun && approvedEnabled(readApproved())}
}

// Set turns the sign-in start on or off. On writes the Run value when it
// is missing (a CLI+tray install made with KONGTROL_NO_AUTOSTART=1, or an
// installer task left unchecked) and marks it approved; off only marks it
// not approved, like Settings > Apps > Startup: the installer that owns
// the Run value may write it again on an update and the choice holds.
func Set(on bool) error {
	_, hasRun := readRun()
	if on && !hasRun {
		tray, ok := TrayPath()
		if !ok {
			return ErrNoTray
		}
		k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
		if err != nil {
			return err
		}
		err = k.SetStringValue(valueName, runValueFor(tray))
		_ = k.Close()
		if err != nil {
			return err
		}
	}
	if !on && !hasRun {
		return nil
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, approvedKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetBinaryValue(valueName, approvalValue(on, time.Now()))
}

// Remove deletes what starts tray at sign-in (uninstall): the Run value
// only while it points to tray (the Windows installer's own copy is never
// touched), then the StartupApproved value once no Run value is left.
func Remove(tray string) error {
	var errs []error
	if data, ok := readRun(); ok && runValuePoints(data, tray) {
		if k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE); err == nil {
			errs = append(errs, k.DeleteValue(valueName))
			_ = k.Close()
		}
	}
	if _, ok := readRun(); !ok {
		if k, err := registry.OpenKey(registry.CURRENT_USER, approvedKey, registry.SET_VALUE); err == nil {
			if err := k.DeleteValue(valueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
				errs = append(errs, err)
			}
			_ = k.Close()
		}
	}
	return errors.Join(errs...)
}
