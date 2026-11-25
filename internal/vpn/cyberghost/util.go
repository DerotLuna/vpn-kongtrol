package cyberghost

import "runtime"

func detectPaths() []string {
	if runtime.GOOS == "linux" {
		return []string{"cyberghostvpn", "/usr/local/bin/cyberghostvpn"}
	}
	return nil
}
