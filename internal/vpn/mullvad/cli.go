package mullvad

import (
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

func connectArgs(cfg vpn.AdapterConfig) ([][]string, error) {
	var location []string
	if server := strings.TrimSpace(cfg.Extra["server"]); server != "" {
		location = []string{server}
	} else {
		for _, value := range []string{cfg.Extra["country"], cfg.Extra["city"]} {
			if strings.TrimSpace(value) != "" {
				location = append(location, value)
			}
		}
	}
	if len(location) == 0 {
		return [][]string{{"connect"}}, nil
	}
	for i, value := range location {
		target, err := appattach.ExactLocation(value)
		if err != nil {
			return nil, err
		}
		location[i] = strings.ToLower(target)
	}
	return [][]string{append([]string{"relay", "set", "location"}, location...), {"connect"}}, nil
}
