//go:build darwin

package routing

import "os/exec"

func listAllOS() ([]Route, error) {
	out, err := exec.Command("netstat", "-rn").Output()
	if err != nil {
		return nil, err
	}
	return parseNetstatRoutes(string(out)), nil
}
