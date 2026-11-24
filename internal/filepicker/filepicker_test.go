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
