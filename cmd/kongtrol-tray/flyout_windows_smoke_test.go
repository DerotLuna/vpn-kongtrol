//go:build windows

package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// TestFlyoutWindowSmoke opens the real WebView2 flyout against a local test
// server. It shows a window for a few seconds, so it runs only on request:
//
//	KONGTROL_FLYOUT_SMOKE=1 go test -run TestFlyoutWindowSmoke ./cmd/kongtrol-tray/
func TestFlyoutWindowSmoke(t *testing.T) {
	if os.Getenv("KONGTROL_FLYOUT_SMOKE") != "1" {
		t.Skip("set KONGTROL_FLYOUT_SMOKE=1 to open a real WebView2 window")
	}

	// Keep WebView2's profile out of the real ~/.kongtrol. Not t.TempDir():
	// WebView2's helper processes may still hold files at cleanup.
	home, err := os.MkdirTemp("", "kongtrol-flyout-smoke-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("USERPROFILE", home)

	// A second origin the page tries to reach: the request filter must stop it.
	var leaks atomic.Int32
	outsideLn, err := net.Listen("tcp", "127.0.0.2:0")
	if err != nil {
		t.Skipf("127.0.0.2 not available: %v", err)
	}
	outside := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { leaks.Add(1) }))
	outside.Listener.Close()
	outside.Listener = outsideLn
	outside.Start()
	defer outside.Close()

	var pageLoads, bridgeLeaks atomic.Int32
	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case flyoutPath:
			// First load: try the outside origin, say ready, then leave the
			// page. Later loads only say ready.
			leave := ""
			if pageLoads.Add(1) == 1 {
				leave = `setTimeout(function () { location.href = "/app/other"; }, 300);`
			}
			fmt.Fprintf(w, `<!doctype html><title>t</title><script>
fetch(%q).catch(function () {}).finally(function () {
  if (window.kongtrolTray) window.kongtrolTray.post("ready");
  %s
});
</script>`, outside.URL+"/leak", leave)
		case "/app/other":
			fmt.Fprint(w, `<!doctype html><title>o</title><script>
if (window.kongtrolTray) fetch("/bridge-leaked");
</script>`)
		case "/bridge-leaked":
			bridgeLeaks.Add(1)
		}
	}))
	defer daemon.Close()

	pageURL, err := flyoutURL(daemon.URL)
	if err != nil {
		t.Fatal(err)
	}
	fly.url, fly.nonce = pageURL, "smoke-nonce"
	fly.pendingShow.Store(true)
	go fly.run()

	waitFor(t, "page ready", fly.ready.Load)
	if !fly.visible.Load() {
		t.Error("flyout not visible after the first show")
	}
	// Leaving the page makes the injected script ask for the flyout page again.
	waitFor(t, "flyout page loaded again", func() bool { return pageLoads.Load() >= 2 })
	if n := leaks.Load(); n != 0 {
		t.Errorf("a request to another origin got through %d time(s)", n)
	}
	if n := bridgeLeaks.Load(); n != 0 {
		t.Error("the bridge was exposed outside the flyout page")
	}
	fly.post(wmFlyoutHide)
	waitFor(t, "hidden", func() bool { return !fly.visible.Load() })
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
