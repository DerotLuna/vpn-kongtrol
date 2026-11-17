//go:build !windows

package routing

// listOS: Linux and macOS report only the routes Kongtrol added.
func listOS() ([]Route, bool) { return nil, false }
