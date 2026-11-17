//go:build linux

package netstats

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// counters reads /sys/class/net/<if>/statistics, world-readable.
func counters(name string) (sent, received uint64, ok bool) {
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return 0, 0, false
	}
	dir := filepath.Join("/sys/class/net", name, "statistics")
	tx, errTx := readUint(filepath.Join(dir, "tx_bytes"))
	rx, errRx := readUint(filepath.Join(dir, "rx_bytes"))
	if errTx != nil || errRx != nil {
		return 0, 0, false
	}
	return tx, rx, true
}

func readUint(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}
