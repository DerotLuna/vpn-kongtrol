//go:build windows

package main

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/windows"

	"github.com/vpn-kongtrol/kongtrol/internal/privhelper"
)

func elevationPromptKey() string { return "service.install.uac" }

// currentUser returns this process's user SID and DOMAIN\name.
func currentUser() (sid, name string, err error) {
	u, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", "", err
	}
	sid = u.User.Sid.String()
	if acct, dom, _, lerr := u.User.Sid.LookupAccount(""); lerr == nil {
		name = dom + `\` + acct
	}
	return sid, name, nil
}

func identityArgs() []string {
	sid, name, err := currentUser()
	if err != nil {
		return nil
	}
	return []string{"--user-sid", sid, "--user-name", name}
}

// elevatedInstallOptions takes the installing user from the flags sent by
// the unprivileged parent, or the current user in an admin terminal.
func elevatedInstallOptions() (privhelper.InstallOptions, error) {
	sid, name := svcUserSID, svcUserName
	if sid == "" {
		var err error
		if sid, name, err = currentUser(); err != nil {
			return privhelper.InstallOptions{}, err
		}
	}
	if _, err := windows.StringToSid(sid); err != nil {
		return privhelper.InstallOptions{}, fmt.Errorf("invalid user SID %q", sid)
	}
	return privhelper.InstallOptions{UserID: sid, UserName: name, GID: -1}, nil
}

// runElevatedStep re-runs this executable through one UAC prompt.
func runElevatedStep(action string, extra []string) error {
	exe, err := currentExecutable()
	if err != nil {
		return err
	}
	args := append([]string{"service", action, "--elevated-step", "--quiet"}, extra...)
	code, err := privhelper.RunElevated(exe, args, 3*time.Minute)
	if err != nil {
		return err
	}
	if code != 0 {
		return errors.New(cf("service.elevated.exit_code", code))
	}
	return nil
}
