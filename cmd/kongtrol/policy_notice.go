package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// policyNoticeTTL is how long the live view keeps showing the latest
// policy notice ("claude.ai sale por internet normal · brainbox está
// apagado").
const policyNoticeTTL = 30 * time.Minute

// fetchDaemonEvents reads the daemon notices newer than after.
func fetchDaemonEvents(client *http.Client, base string, after int64) ([]monitor.Event, int64, error) {
	req, err := daemonRequest(http.MethodGet, base+"/api/v1/events?after="+strconv.FormatInt(after, 10), nil)
	if err != nil {
		return nil, after, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, after, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, after, fmt.Errorf("events unavailable")
	}
	var body struct {
		Events []monitor.Event `json:"events"`
		Seq    int64           `json:"seq"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, after, err
	}
	return body.Events, body.Seq, nil
}

// policyNoticeLine renders an event in the CLI language, with the action
// the user can take, or "" when there is none or it is too old.
func policyNoticeLine(e *monitor.Event, now time.Time) string {
	if e == nil || now.Sub(e.Time) > policyNoticeTTL {
		return ""
	}
	args := make([]any, len(e.Args))
	for i, a := range e.Args {
		args[i] = a
	}
	line := e.Time.Format("15:04") + "  " + cf(e.Key, args...)
	if e.CanConnect && e.Profile != "" && e.Kind != monitor.EventTunnelWatchdog { // watchdog notices name the command already
		line += "  ·  " + cf("policy.event.hint_cli", e.Profile)
	}
	if hint := offerHintCLI(e); hint != "" {
		line += "  ·  " + hint
	}
	return line
}

// offerHintCLI is the command that does what a notice's Offer button does
// in the panel and the tray (ADR-0102), or "".
func offerHintCLI(e *monitor.Event) string {
	switch e.Offer {
	case offerEnableKillSwitch:
		return cf("offer.cli.enable_killswitch")
	case offerReleaseBlock:
		if e.Profile == "" || strings.Contains(cf(e.Key, anyArgs(e.Args)...), "kongtrol down "+e.Profile) {
			return "" // the notice names the command already
		}
		return cf("offer.cli.release_block", e.Profile)
	case offerAddProvider:
		if len(e.Args) == 0 {
			return ""
		}
		return cf("offer.cli.add_provider", e.Args[len(e.Args)-1])
	}
	return ""
}

func anyArgs(args []string) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = a
	}
	return out
}
