//go:build !windows && !linux && !darwin

package routing

func listAllOS() ([]Route, error) { return nil, nil }
