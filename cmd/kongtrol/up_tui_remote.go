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
	go pollRemoteSecurity(ctx, p, base, done)

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

// pollRemoteSecurity keeps the live view's kill switch / DNS guard line in
// step with the daemon (the WebSocket feed carries tunnels only), so it shows
// the same state `kongtrol status` reports.
func pollRemoteSecurity(ctx context.Context, p *tea.Program, base string, done <-chan struct{}) {
	client := &http.Client{Timeout: 1200 * time.Millisecond}
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	var eventSeq int64
	for {
		if sec, err := fetchDaemonSecurity(client, base); err == nil && !isClosed(done) {
			p.Send(upRemoteStateMsg{connected: true, apiBase: base, security: sec})
		}
		// Policy and watchdog notices (one line in the live view).
		if evs, seq, err := fetchDaemonEvents(client, base, eventSeq); err == nil && !isClosed(done) {
			eventSeq = seq
			for i := len(evs) - 1; i >= 0; i-- {
				if k := evs[i].Kind; k == monitor.EventPolicyTunnelDown || k == monitor.EventTunnelWatchdog {
					e := evs[i]
					p.Send(upRemoteStateMsg{connected: true, apiBase: base, notice: &e})
					break
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-t.C:
		}
	}
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
