package filepicker

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/ncruces/zenity"
)

// fake replaces the dialog (tests never open a window).
func fake(t *testing.T, avail bool, fn func(...zenity.Option) (string, error)) {
	t.Helper()
	oldA, oldS := availableFn, selectFile
	availableFn = func() bool { return avail }
	selectFile = fn
	t.Cleanup(func() { availableFn, selectFile = oldA, oldS })
}

func TestPick(t *testing.T) {
	ctx := context.Background()
	o := Options{Title: "t", FilterName: "VPN", Patterns: []string{"*.ovpn"}}

	fake(t, false, nil)
	if _, err := Pick(ctx, o); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}

	fake(t, true, func(...zenity.Option) (string, error) { return `C:\vpn\office.ovpn`, nil })
	if p, err := Pick(ctx, o); err != nil || p != `C:\vpn\office.ovpn` {
		t.Fatal(p, err)
	}

	fake(t, true, func(...zenity.Option) (string, error) { return "", zenity.ErrCanceled })
	if _, err := Pick(ctx, o); !errors.Is(err, ErrCanceled) {
		t.Fatal(err)
	}

	fake(t, true, func(...zenity.Option) (string, error) { return "", errors.New("exec: zenity: not found") })
	if _, err := Pick(ctx, o); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}

func TestPick_OneAtATime(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	fake(t, true, func(...zenity.Option) (string, error) {
		once.Do(func() { close(started) })
		<-release
		return "/a.ovpn", nil
	})
	done := make(chan error)
	go func() {
		_, err := Pick(context.Background(), Options{})
		done <- err
	}()
	<-started
	if _, err := Pick(context.Background(), Options{}); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// A second click while a dialog is open brings that dialog to the front
// (it may have dropped behind the browser) and still answers ErrBusy; the
// open dialog is kept in front for its whole life.
func TestPick_BusyBringsTheOpenDialogToFront(t *testing.T) {
	var mu sync.Mutex
	var raised, brought []string
	oldR, oldB := raiseFn, bringFn
	raiseFn = func(_ context.Context, title string) func() {
		mu.Lock()
		raised = append(raised, title)
		mu.Unlock()
		return func() {}
	}
	bringFn = func(title string) {
		mu.Lock()
		brought = append(brought, title)
		mu.Unlock()
	}
	t.Cleanup(func() { raiseFn, bringFn = oldR, oldB })

	release := make(chan struct{})
	started := make(chan struct{})
	fake(t, true, func(...zenity.Option) (string, error) {
		close(started)
		<-release
		return "/a.ovpn", nil
	})
	done := make(chan error)
	go func() {
		_, err := Pick(context.Background(), Options{Title: "Elige el archivo"})
		done <- err
	}()
	<-started
	if _, err := PickFolder(context.Background(), Options{Title: "Elige la carpeta"}); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(raised) != 1 || raised[0] != "Elige el archivo" {
		t.Fatalf("raised = %q, want the open dialog kept in front", raised)
	}
	if len(brought) != 1 || brought[0] != "Elige el archivo" {
		t.Fatalf("brought = %q, want the open dialog's title", brought)
	}
	// Closed: nothing is left to bring to the front.
	shown.Lock()
	left := shown.title
	shown.Unlock()
	if left != "" {
		t.Fatalf("title kept after close: %q", left)
	}
}
