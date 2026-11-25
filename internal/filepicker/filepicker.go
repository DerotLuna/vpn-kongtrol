// Package filepicker opens the operating system's own "Open file" dialog
// from the daemon, so the panel's "Elegir archivo" gets a real path (the
// browser's file input never gives one) and an imported .ovpn can find the
// certificates next to it (ADR-0105). PickFolder opens the same dialog in
// folder mode, for "Elegir la carpeta donde están" when a dropped .ovpn's
// files are still missing (ADR-0112).
//
// It wraps github.com/ncruces/zenity (MIT, pure Go, no cgo): Windows uses
// the Common Item Dialog (IFileOpenDialog) in-process; macOS runs
// osascript (JavaScript for Automation, "choose file" / "choose folder");
// Linux and the BSDs run qarma, zenity or matedialog, whichever is
// installed. Available says whether this session can show one; when it
// cannot, the panel falls back to the browser's file input. Research and
// sources: docs/research/2026-11-24-native-file-picker.md.
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

// shown is the title of the dialog open now ("" when none), so a second
// click can bring that dialog to the front instead of only refusing.
var shown struct {
	sync.Mutex
	title string
}

// raiseFn keeps the open dialog in front for its whole life; bringFn
// raises an already-open dialog once (variables so tests never touch a
// window; raise_windows.go / raise_other.go).
var (
	raiseFn = raise
	bringFn = bringToFront
)

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
	opts := []zenity.Option{}
	if len(o.Patterns) > 0 {
		opts = append(opts, zenity.FileFilter{Name: o.FilterName, Patterns: o.Patterns, CaseFold: true})
	}
	return pick(ctx, o.Title, opts)
}

// PickFolder shows the dialog in folder mode (zenity.Directory) and
// returns the absolute path of the folder picked.
func PickFolder(ctx context.Context, o Options) (string, error) {
	return pick(ctx, o.Title, []zenity.Option{zenity.Directory()})
}

func pick(ctx context.Context, title string, extra []zenity.Option) (string, error) {
	if !Available() {
		return "", ErrUnavailable
	}
	if !open.TryLock() {
		// The dialog may sit behind the browser (the daemon cannot take
		// the foreground): show the user the one already open.
		shown.Lock()
		t := shown.title
		shown.Unlock()
		if t != "" {
			bringFn(t)
		}
		return "", ErrBusy
	}
	defer open.Unlock()
	shown.Lock()
	shown.title = title
	shown.Unlock()
	defer func() {
		shown.Lock()
		shown.title = ""
		shown.Unlock()
	}()
	opts := append([]zenity.Option{zenity.Context(ctx), zenity.Title(title)}, extra...)
	stop := raiseFn(ctx, title)
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
