package privhelper

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

const (
	readTimeout   = 10 * time.Second
	writeTimeout  = 10 * time.Second
	maxConcurrent = 16
)

// Authenticator identifies and authorizes the peer of conn. release frees
// OS resources tied to the caller (e.g. an impersonation token).
type Authenticator func(conn net.Conn) (caller privops.Caller, release func(), err error)

// Server serves one request per connection.
type Server struct {
	Ops          privops.Ops
	Authenticate Authenticator
	Audit        *Audit
}

// Serve accepts connections until ctx is done or l fails.
func (s *Server) Serve(ctx context.Context, l net.Listener) error {
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		conn, err := l.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return err
		}
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() { <-sem; wg.Done() }()
			s.HandleConn(ctx, conn)
		}()
	}
}

// HandleConn authenticates the peer, reads one request, runs it under the
// op's timeout and writes the response. Every request is audited.
func (s *Server) HandleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			s.Audit.Record(AuditEntry{Time: start, Op: "panic", Error: "internal error"})
		}
	}()
	caller, release, err := s.Authenticate(conn)
	if err != nil {
		s.Audit.Record(AuditEntry{Time: start, Op: "connect", Code: "denied", Error: err.Error()})
		// Drain the request (bounded, discarded) so the client's write
		// completes and it reads the denial instead of a broken pipe.
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var discard privops.Request
		_ = privops.ReadFrame(conn, &discard)
		_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		_ = privops.WriteFrame(conn, privops.Response{V: privops.ProtocolVersion, Code: privops.ErrorCode(privops.ErrDenied), Error: "not allowed to use the Kongtrol helper"})
		return
	}
	if release != nil {
		defer release()
	}
	_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
	var req privops.Request
	if err := privops.ReadFrame(conn, &req); err != nil {
		s.Audit.Record(AuditEntry{Time: start, Caller: caller.Name, CallerID: caller.ID, Op: "read", Code: "protocol", Error: err.Error()})
		_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		_ = privops.WriteFrame(conn, privops.Response{V: privops.ProtocolVersion, Code: privops.ErrorCode(privops.ErrInvalidRequest), Error: "malformed request"})
		return
	}
	opCtx, cancel := context.WithTimeout(privops.WithCaller(ctx, caller), privops.OpTimeout(req.Op))
	resp := privops.Dispatch(opCtx, s.Ops, req)
	cancel()

	s.Audit.Record(AuditEntry{
		Time: start, Caller: caller.Name, CallerID: caller.ID, Op: req.Op, Args: req.Args,
		OK: resp.OK, Code: resp.Code, Error: resp.Error, Millis: time.Since(start).Milliseconds(),
	})
	_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	_ = privops.WriteFrame(conn, resp)
}

// authorize applies the access rule shared by all platforms: the installing
// user, or an elevated administrator / root.
func authorize(cfg Config, c privops.Caller) error {
	if c.Admin || (c.ID != "" && c.ID == cfg.UserID) {
		return nil
	}
	return privops.ErrDenied
}
