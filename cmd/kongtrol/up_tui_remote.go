package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// watchRemoteDaemon runs for the lifetime of a status --watch session. It
// repeatedly probes for a live `kongtrol up` daemon and, once found, streams
// its live tunnel snapshot over the same WebSocket feed the dashboard uses,
// retrying on disconnect — instead of a single startup probe that goes
// stale if the daemon appears later or drops and comes back.
func watchRemoteDaemon(ctx context.Context, p *tea.Program) {
	const retryDelay = 3 * time.Second
	base := daemonAPIBase()

	for ctx.Err() == nil {
		if !probeDaemonAPI(base) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
			}
			continue
		}
		streamRemoteTunnels(ctx, p, base)
		p.Send(upRemoteStateMsg{connected: false})
		select {
		case <-ctx.Done():
			return
		case <-time.After(retryDelay):
		}
	}
}

// streamRemoteTunnels dials the daemon's live metrics WebSocket and
// forwards every snapshot into the TUI until the connection drops or ctx
// is cancelled.
func streamRemoteTunnels(ctx context.Context, p *tea.Program, base string) {
	headers := http.Header{"X-Kongtrol-Token": []string{apiToken}}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, daemonWSURL(base), headers)
	if err != nil {
		return
	}
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var snapshot map[string]monitor.TunnelMetrics
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue
		}
		p.Send(upRemoteStateMsg{connected: true, apiBase: base, snapshot: snapshot})
	}
}
