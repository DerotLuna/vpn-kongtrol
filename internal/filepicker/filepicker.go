// Package filepicker opens the operating system's own "Open file" dialog
// from the daemon, so the panel's "Elegir archivo" gets a real path (the
// browser's file input never gives one) and an imported .ovpn can find the
// certificates next to it (ADR-0105).
//
// It wraps github.com/ncruces/zenity (MIT, pure Go, no cgo): Windows uses
// the Common Item Dialog (IFileOpenDialog) in-process; macOS runs
// osascript (JavaScript for Automation, "choose file"); Linux and the BSDs
// run qarma, zenity or matedialog, whichever is installed. Available says
// whether this session can show one; when it cannot, the panel falls back
// to the browser's file input. Research and sources:
// docs/research/2026-11-24-native-file-picker.md.
//
// Only the path the user picked leaves this package (nothing is read
// here); one dialog at a time; the dialog closes when ctx ends (the page
// went away or the request timed out).
package filepicker

import (
	"context"
	"errors"
	"sync"

	"github.com/ncruces/zenity"
)

var (
	// ErrUnavailable: no dialog can be shown in this session (no desktop,
	// no dialog program installed).
	ErrUnavailable = errors.New("file picker unavailable")
	// ErrCanceled: the user closed the dialog without picking a file.
	ErrCanceled = errors.New("file picker canceled")
	// ErrBusy: another dialog is already open.
	ErrBusy = errors.New("file picker already open")
)

var open sync.Mutex

// Options are the dialog's translated texts and the file patterns.
type Options struct {
	Title      string
	FilterName string
	Patterns   []string // e.g. "*.ovpn", "*.conf"
}

// selectFile is zenity.SelectFile (a variable so tests never open a window).
var selectFile = zenity.SelectFile

// availableFn is Available's OS check (a variable for tests).
var availableFn = available

// Available reports whether a native dialog can be shown now.
func Available() bool { return availableFn() }

// Pick shows the dialog and returns the absolute path of the file picked.
func Pick(ctx context.Context, o Options) (string, error) {
	if !Available() {
		return "", ErrUnavailable
	}
	if !open.TryLock() {
		return "", ErrBusy
	}
	defer open.Unlock()
	opts := []zenity.Option{zenity.Context(ctx), zenity.Title(o.Title)}
	if len(o.Patterns) > 0 {
		opts = append(opts, zenity.FileFilter{Name: o.FilterName, Patterns: o.Patterns, CaseFold: true})
	}
	stop := raise(ctx, o.Title)
	defer stop()
	path, err := selectFile(opts...)
	switch {
	case errors.Is(err, zenity.ErrCanceled):
		return "", ErrCanceled
	case err != nil && ctx.Err() != nil:
		return "", ctx.Err()
	case err != nil:
		// The dialog program failed to start (e.g. removed meanwhile).
		return "", errors.Join(ErrUnavailable, err)
	case path == "":
		return "", ErrCanceled
	}
	return path, nil
}
