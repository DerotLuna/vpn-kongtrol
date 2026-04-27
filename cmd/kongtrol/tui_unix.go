//go:build !windows

package main

// enableTerminal is a no-op on Unix — ANSI escape codes work by default in
// any terminal emulator that sets TERM (xterm, tmux, etc.).
func init() {}
