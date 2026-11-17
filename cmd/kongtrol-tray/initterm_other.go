//go:build !windows

package main

import "errors"

// canRunInit is false where no terminal emulator can be assumed; the
// first-run message tells the user to run `kongtrol init` themselves.
const canRunInit = false

func runInitInTerminal(string) error {
	return errors.New("opening a terminal is not supported on this OS")
}
