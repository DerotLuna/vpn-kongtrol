//go:build windows

package routing

import (
	"os/exec"
	"syscall"
)

func listAllOS() ([]Route, error) {
	ps := `Get-NetRoute -ErrorAction SilentlyContinue | ForEach-Object { ` +
		`"$($_.DestinationPrefix)|$($_.NextHop)|$($_.InterfaceAlias)|$($_.RouteMetric)" }`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseNetRouteLines(string(out)), nil
}
