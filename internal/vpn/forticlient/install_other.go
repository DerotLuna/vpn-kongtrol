//go:build !windows

package forticlient

// DetectInstall reports the FortiClient CLI used on Linux / macOS. The
// version is left unknown: the strategy there does not depend on it, and
// running the binary here could block (doctor reads it with a timeout).
func DetectInstall(appPath string) Install {
	if appPath != "" {
		return Install{Path: appPath}
	}
	return Install{Path: binaryPath()}
}
