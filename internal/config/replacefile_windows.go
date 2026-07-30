//go:build windows

package config

import (
	"time"

	"golang.org/x/sys/windows"
)

// replaceFile retries MoveFileEx on ERROR_ACCESS_DENIED/ERROR_SHARING_VIOLATION:
// on Windows, a newly written temp file (or the destination being replaced) can
// be briefly locked by another handle — most commonly real-time antivirus
// scanning the file the instant it's created — causing a transient
// "Access is denied" even though no process is meaningfully contending for the
// file. This is a well-known Go-on-Windows gotcha for atomic file replace, not
// a real permissions problem, so a few short retries clear it without the
// caller (e.g. the dashboard's policy/group/settings save handlers) surfacing
// a spurious failure for what is, a few milliseconds later, a successful write.
func replaceFile(source, destination string) error {
	sourcePtr, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	destinationPtr, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}

	const maxAttempts = 5
	backoff := 20 * time.Millisecond
	for attempt := 1; ; attempt++ {
		err := windows.MoveFileEx(
			sourcePtr,
			destinationPtr,
			windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH,
		)
		if err == nil {
			return nil
		}
		if attempt >= maxAttempts || !isTransientReplaceError(err) {
			return err
		}
		time.Sleep(backoff)
		backoff *= 2
	}
}

func isTransientReplaceError(err error) bool {
	return err == windows.ERROR_ACCESS_DENIED || err == windows.ERROR_SHARING_VIOLATION
}
