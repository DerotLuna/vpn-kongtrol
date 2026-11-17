//go:build !windows && !linux && !darwin

package netstats

func counters(string) (sent, received uint64, ok bool) { return 0, 0, false }
