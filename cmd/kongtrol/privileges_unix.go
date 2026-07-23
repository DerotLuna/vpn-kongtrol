//go:build !windows

package main

import "os"

func hasNetworkAdminPrivileges() bool {
	return os.Geteuid() == 0
}
