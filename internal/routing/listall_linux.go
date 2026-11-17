//go:build linux

package routing

import "os/exec"

func listAllOS() ([]Route, error) {
	out, err := exec.Command("ip", "route", "show").Output()
	if err != nil {
		return nil, err
	}
	routes := parseIPRouteLines(string(out), false)
	if out6, err := exec.Command("ip", "-6", "route", "show").Output(); err == nil {
		routes = append(routes, parseIPRouteLines(string(out6), true)...)
	}
	return routes, nil
}
