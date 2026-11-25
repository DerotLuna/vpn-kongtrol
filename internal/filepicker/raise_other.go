//go:build !windows

package filepicker

import "context"

// raise: on macOS and Linux the dialog is its own process (osascript, the
// dialog program), placed by the window manager; no extra step (not
// verified on a real desktop, docs/research/2026-11-24-native-file-picker.md).
func raise(context.Context, string) func() { return func() {} }

// bringToFront: nothing to do for the same reason.
func bringToFront(string) {}
