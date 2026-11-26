//go:build windows

package filepicker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// TestDesktopPicker requires an interactive desktop and explicit opt-in.
func TestDesktopPicker(t *testing.T) {
	if os.Getenv("KONGTROL_TEST_DESKTOP_PICKER") != "1" {
		t.Skip("requires an interactive desktop")
	}

	const title = "Kongtrol selector de prueba"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := Pick(ctx, Options{Title: title, FilterName: "VPN", Patterns: []string{"*.ovpn"}})
		done <- err
	}()
	name, err := windows.UTF16PtrFromString(title)
	if err != nil {
		t.Fatal(err)
	}
	isVisible := user32.NewProc("IsWindowVisible")
	deadline := time.Now().Add(10 * time.Second)
	for {
		if h := ownWindow(name); h != 0 {
			visible, _, _ := isVisible.Call(h)
			if visible != 0 {
				break
			}
		}

		select {
		case err := <-done:
			t.Fatalf("picker closed before becoming visible: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("picker did not become visible")
		}
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)
	h := ownWindow(name)
	visible, _, _ := isVisible.Call(h)
	if h == 0 || visible == 0 {
		t.Fatal("picker did not remain visible")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, ErrCanceled) && !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel picker: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("picker did not close after cancellation")
	}
}

func TestDesktopPickerSelect(t *testing.T) {
	if os.Getenv("KONGTROL_TEST_DESKTOP_PICKER") != "select" {
		t.Skip("requires an interactive desktop")
	}
	path := filepath.Join(t.TempDir(), "sample.ovpn")
	if err := os.WriteFile(path, []byte("test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("Select this test file: %s", path)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	got, err := Pick(ctx, Options{Title: "Kongtrol selector de prueba", FilterName: "VPN", Patterns: []string{"*.ovpn"}})
	if err != nil || got != path {
		t.Fatalf("Pick = %q, %v; want %q", got, err, path)
	}
}
