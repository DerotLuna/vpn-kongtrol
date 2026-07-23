//go:build windows

package main

import "golang.org/x/sys/windows"

func hasNetworkAdminPrivileges() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
