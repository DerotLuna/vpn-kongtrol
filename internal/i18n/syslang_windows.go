//go:build windows

package i18n

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetUserDefaultLocaleName = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")

// systemLocale returns the user's default locale name ("es-MX", "en-US")
// via kernel32!GetUserDefaultLocaleName.
func systemLocale() string {
	const localeNameMaxLength = 85 // LOCALE_NAME_MAX_LENGTH
	buf := make([]uint16, localeNameMaxLength)
	if err := procGetUserDefaultLocaleName.Find(); err != nil {
		return ""
	}
	n, _, _ := procGetUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return windows.UTF16ToString(buf)
}
