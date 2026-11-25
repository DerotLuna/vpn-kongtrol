package pia

import (
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

func connectArgs(cfg vpn.AdapterConfig) ([][]string, error) {
	target, err := appattach.ExactLocation(cfg.Extra["server"])
	if err != nil {
		return nil, err
	}
	if target == "" {
		return [][]string{{"connect"}}, nil
	}
	return [][]string{{"set", "region", target}, {"connect"}}, nil
}
