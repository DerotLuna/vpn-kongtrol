package privops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DialFunc opens one connection to the helper.
type DialFunc func(ctx context.Context) (net.Conn, error)

// Client implements Ops by sending each call to the helper service. It is
// safe for concurrent use: every call uses its own connection.
type Client struct {
	dial DialFunc
	// DefaultTimeout bounds a call whose ctx has no deadline.
	DefaultTimeout time.Duration
	// RenewEvery is how often the lease is renewed once this client owns
	// state in the helper (default LeaseRenewEvery).
	RenewEvery time.Duration

	lease      string
	renewOnce  sync.Once
	stopRenew  chan struct{}
	closeOnce  sync.Once
	renewCount atomic.Int64
}

// NewClient returns a client for the installed helper (named pipe on
// Windows, Unix socket elsewhere). The server's identity (SYSTEM/root) is
// verified on every dial. Each Client has its own lease: keep one per
// process (Default does).
func NewClient() *Client { return NewClientWithDialer(dialHelper) }

// NewClientWithDialer returns a client over a custom transport (tests).
func NewClientWithDialer(d DialFunc) *Client {
	return &Client{dial: d, DefaultTimeout: 60 * time.Second, RenewEvery: LeaseRenewEvery,
		lease: randomHex(16), stopRenew: make(chan struct{})}
}

// Close stops lease renewal. The helper then undoes this client's state
// after LeaseTTL — call it only when the process is about to exit without
// cleaning up, or in tests.
func (c *Client) Close() {
	c.closeOnce.Do(func() { close(c.stopRenew) })
}

// leasedOp reports whether op can leave state owned by the lease.
func leasedOp(op string) bool {
	switch op {
	case OpOpenVPNStart, OpKillSwitchOn, OpDNSSet, OpHostsWrite, OpBlockSet:
		return true
	}
	return false
}

// startRenewal renews the lease every RenewEvery for the rest of the
// process (or until Close). Failures are ignored: a missed renewal only
// matters after LeaseTTL, i.e. four in a row.
func (c *Client) startRenewal() {
	c.renewOnce.Do(func() {
		every := c.RenewEvery
		if every <= 0 {
			every = LeaseRenewEvery
		}
		go func() {
			t := time.NewTicker(every)
			defer t.Stop()
			for {
				select {
				case <-c.stopRenew:
					return
				case <-t.C:
					ctx, cancel := context.WithTimeout(context.Background(), every)
					if c.RenewLease(ctx) == nil {
						c.renewCount.Add(1)
					}
					cancel()
				}
			}
		}()
	})
}

// Call sends one request and decodes the result into out (may be nil).
func (c *Client) Call(ctx context.Context, op string, args, out any) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.DefaultTimeout)
		defer cancel()
	}
	conn, err := c.dial(ctx)
	if err != nil {
		if errors.Is(err, ErrDenied) {
			return err
		}
		return fmt.Errorf("%w: %v", ErrHelperNotInstalled, err)
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	req := Request{V: ProtocolVersion, Op: op, Lease: c.lease}
	if args != nil {
		if req.Args, err = json.Marshal(args); err != nil {
			return err
		}
	}
	if err := WriteFrame(conn, req); err != nil {
		return fmt.Errorf("%w: send: %v", ErrFailed, err)
	}
	var resp Response
	if err := ReadFrame(conn, &resp); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %v", ErrFailed, ctx.Err())
		}
		return fmt.Errorf("%w: receive: %v", ErrFailed, err)
	}
	if resp.V != ProtocolVersion {
		return ErrHelperOutdated
	}
	if !resp.OK {
		return errorFromWire(resp.Code, resp.Error, resp.Reason)
	}
	if leasedOp(op) {
		c.startRenewal()
	}
	if out != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, out); err != nil {
			return fmt.Errorf("%w: decode result: %v", ErrFailed, err)
		}
	}
	return nil
}

func (c *Client) Ping(ctx context.Context) (PingResult, error) {
	var r PingResult
	err := c.Call(ctx, OpPing, nil, &r)
	return r, err
}

func (c *Client) AddRoute(ctx context.Context, r Route) error {
	return c.Call(ctx, OpRouteAdd, r, nil)
}

func (c *Client) DeleteRoute(ctx context.Context, r Route) error {
	return c.Call(ctx, OpRouteDelete, r, nil)
}

func (c *Client) FlushRoutes(ctx context.Context, iface string) error {
	return c.Call(ctx, OpRouteFlush, IfaceArgs{Interface: iface}, nil)
}

func (c *Client) SetDNS(ctx context.Context, req DNSRequest) error {
	return c.Call(ctx, OpDNSSet, req, nil)
}

func (c *Client) RestoreDNS(ctx context.Context, iface string) error {
	return c.Call(ctx, OpDNSRestore, IfaceArgs{Interface: iface}, nil)
}

func (c *Client) KillSwitchEnable(ctx context.Context, req KillSwitchRequest) error {
	return c.Call(ctx, OpKillSwitchOn, req, nil)
}

func (c *Client) KillSwitchDisable(ctx context.Context) error {
	return c.Call(ctx, OpKillSwitchOff, nil, nil)
}

func (c *Client) WGInstallTunnel(ctx context.Context, req WGInstallRequest) (WGTunnel, error) {
	var t WGTunnel
	err := c.Call(ctx, OpWGInstall, req, &t)
	return t, err
}

func (c *Client) WGRemoveTunnel(ctx context.Context, req WGRemoveRequest) error {
	return c.Call(ctx, OpWGRemove, req, nil)
}

func (c *Client) WGSetAllowedIPs(ctx context.Context, req WGAllowedIPsRequest) error {
	return c.Call(ctx, OpWGAllowedIPs, req, nil)
}

// WGStats reads a tunnel's handshake and transfer counters. A helper
// installed by an older Kongtrol rejects the newer protocol version, or
// does not know the op: both map to ErrHelperOutdated.
func (c *Client) WGStats(ctx context.Context, req WGStatsRequest) (WGStats, error) {
	var s WGStats
	err := c.Call(ctx, OpWGStats, req, &s)
	if errors.Is(err, ErrInvalidRequest) && strings.Contains(err.Error(), "unknown operation") {
		return WGStats{}, ErrHelperOutdated
	}
	return s, err
}

func (c *Client) OpenVPNStart(ctx context.Context, req OpenVPNStartRequest) (OpenVPNSession, error) {
	var s OpenVPNSession
	err := c.Call(ctx, OpOpenVPNStart, req, &s)
	return s, err
}

func (c *Client) OpenVPNStop(ctx context.Context, id string) error {
	return c.Call(ctx, OpOpenVPNStop, SessionArgs{ID: id}, nil)
}

func (c *Client) OpenVPNStatus(ctx context.Context, id string) (OpenVPNState, error) {
	var s OpenVPNState
	err := c.Call(ctx, OpOpenVPNStatus, SessionArgs{ID: id}, &s)
	return s, err
}

func (c *Client) WriteHostsBlock(ctx context.Context, entries []HostEntry) error {
	return c.Call(ctx, OpHostsWrite, HostsArgs{Entries: entries}, nil)
}

func (c *Client) RenewLease(ctx context.Context) error {
	return c.Call(ctx, OpLeaseRenew, nil, nil)
}

// SetBlockedDestinations replaces the blocked destination set. A helper
// installed by an older Kongtrol does not know the op ("unknown
// operation"): that maps to ErrHelperOutdated so the user is told to
// update the helper instead of seeing a raw error.
func (c *Client) SetBlockedDestinations(ctx context.Context, req BlockRequest) error {
	err := c.Call(ctx, OpBlockSet, req, nil)
	if errors.Is(err, ErrInvalidRequest) && strings.Contains(err.Error(), "unknown operation") {
		return ErrHelperOutdated
	}
	return err
}

var _ Ops = (*Client)(nil)
