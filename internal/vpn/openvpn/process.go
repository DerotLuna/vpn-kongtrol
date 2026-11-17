package openvpn

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// helperPollEvery is how often a helper-run openvpn is checked for exit
// and its recent log lines fetched.
var helperPollEvery = 2 * time.Second

// opsFunc returns the privileged implementation used when Kongtrol is not
// elevated (the helper; ErrHelperNotInstalled when missing). Tests replace it.
var opsFunc = privops.Default

// isElevated decides between spawning openvpn directly (admin terminal,
// root) and asking the helper. Tests replace it.
var isElevated = privops.IsElevated

// process manages one openvpn instance: a child process when Kongtrol is
// elevated, else a session run by the Kongtrol helper. Either way Kongtrol
// itself talks to openvpn's management interface.
type process struct {
	mgmtAddr string // 127.0.0.1:<port>
	mgmt     *mgmtClient
	log      *logState
	exited   chan struct{} // closed when the process exits
	exitOnce sync.Once
	exitErr  error // valid after exited is closed

	// Direct child (elevated).
	cmd      *exec.Cmd
	lifetime lifetimeHandle // OS handle tying openvpn's life to Kongtrol's

	// Helper session (not elevated). The helper ties the session to this
	// process's lease, so it is stopped if Kongtrol dies.
	ops      privops.Ops
	session  string
	stopPoll chan struct{}
}

// startOptions are the inputs needed to launch one openvpn instance.
type startOptions struct {
	Binary     string // resolved openvpn executable (direct spawn only)
	ConfigPath string // .ovpn/.conf path (made absolute by buildArgs)
	CertPath   string
	KeyPath    string
}

// buildArgs returns the openvpn command line for opts and a management port.
//
// --cd is set to the config file's directory so relative paths inside the
// config (ca ca.crt, tls-crypt ta.key, ...) resolve exactly as they do when
// OpenVPN GUI launches the same file. It must precede --config.
func buildArgs(opts startOptions, mgmtPort int) ([]string, error) {
	cfgPath, err := filepath.Abs(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}
	args := []string{
		"--cd", filepath.Dir(cfgPath),
		"--config", cfgPath,
		"--management", "127.0.0.1", fmt.Sprintf("%d", mgmtPort),
		"--management-query-passwords",
		"--verb", "3",
	}
	if opts.CertPath != "" {
		args = append(args, "--cert", opts.CertPath, "--key", opts.KeyPath)
	}
	// --route-nopull is deliberately NOT used: the server-pushed routes are the
	// tunnel's own destinations (the user never has to list them). Kongtrol's
	// policy routes are added on top by monitor.PolicyResolver. See ADR 0007.
	return args, nil
}

// start launches openvpn and connects to its management interface: as a
// child process when Kongtrol is elevated, else through the helper.
func start(ctx context.Context, opts startOptions, username, password string) (*process, error) {
	var (
		p   *process
		err error
	)
	if isElevated() {
		p, err = startDirect(opts)
	} else {
		p, err = startViaHelper(ctx, opts)
	}
	if err != nil {
		return nil, err
	}
	if username != "" || password != "" {
		if err := p.mgmt.SendAuth(username, password); err != nil {
			_ = p.stop()
			return nil, fmt.Errorf("send credentials: %w", err)
		}
	}
	return p, nil
}

// startDirect spawns openvpn as a child. A free management port is
// allocated so several OpenVPN instances can run simultaneously.
func startDirect(opts startOptions) (*process, error) {
	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("allocate management port: %w", err)
	}
	args, err := buildArgs(opts, port)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(opts.Binary, args...)
	setProcAttr(cmd)

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	p := &process{
		cmd:      cmd,
		mgmtAddr: fmt.Sprintf("127.0.0.1:%d", port),
		log:      &logState{},
		exited:   make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		return nil, fmt.Errorf("start %s: %w", opts.Binary, err)
	}
	// Best effort: make openvpn die with Kongtrol even on a crash. A failure
	// only means a crash could orphan openvpn; graceful stop still works.
	_ = attachLifetime(p)

	go func() {
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			p.log.consume(sc.Text())
		}
		_, _ = io.Copy(io.Discard, pr) // keep draining if a line was too long
	}()
	go func() {
		err := cmd.Wait()
		releaseLifetime(p)
		_ = pw.Close()
		p.markExited(err)
	}()

	mgmt, err := p.waitForMgmt("", 15*time.Second)
	if err != nil {
		p.kill()
		return nil, err
	}
	p.mgmt = mgmt
	return p, nil
}

// startViaHelper asks the helper to run openvpn on the user's config. The
// helper resolves its own trusted openvpn binary (binary_path is not used)
// and returns the management address and password; Kongtrol answers the
// "ENTER PASSWORD:" prompt and keeps using the management interface. The
// helper's log tail (OpenVPNStatus) replaces openvpn's stdout.
func startViaHelper(ctx context.Context, opts startOptions) (*process, error) {
	ops, err := opsFunc(ctx)
	if err != nil {
		return nil, err
	}
	req := privops.OpenVPNStartRequest{}
	if req.ConfigPath, err = filepath.Abs(opts.ConfigPath); err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}
	if opts.CertPath != "" {
		if req.CertPath, err = filepath.Abs(opts.CertPath); err != nil {
			return nil, err
		}
		if req.KeyPath, err = filepath.Abs(opts.KeyPath); err != nil {
			return nil, err
		}
	}
	sctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	sess, err := ops.OpenVPNStart(sctx, req)
	cancel()
	if err != nil {
		return nil, err
	}
	p := &process{
		mgmtAddr: sess.MgmtAddr,
		log:      &logState{},
		exited:   make(chan struct{}),
		ops:      ops,
		session:  sess.ID,
		stopPoll: make(chan struct{}),
	}
	go p.pollHelper()
	mgmt, err := p.waitForMgmt(sess.MgmtPassword, 15*time.Second)
	sess.MgmtPassword = ""
	if err != nil {
		p.stopHelperSession()
		return nil, err
	}
	p.mgmt = mgmt
	return p, nil
}

// pollHelper mirrors the helper session into p: recent log lines (tunnel
// interface, pushed DNS, errors) and the exit.
func (p *process) pollHelper() {
	t := time.NewTicker(helperPollEvery)
	defer t.Stop()
	for {
		p.refreshHelperState()
		select {
		case <-p.stopPoll:
			return
		case <-p.exited:
			return
		case <-t.C:
		}
	}
}

func (p *process) refreshHelperState() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st, err := p.ops.OpenVPNStatus(ctx, p.session)
	switch {
	case errors.Is(err, privops.ErrNotFound):
		// The helper restarted or the lease expired: the session is gone.
		p.markExited(errors.New("openvpn session ended in the Kongtrol helper"))
	case err != nil:
		// Helper briefly unreachable: keep the last known state.
	default:
		p.log.replaceTail(st.LogTail, st.PushReply)
		if !st.Running {
			var exitErr error
			if st.ExitError != "" {
				exitErr = errors.New(st.ExitError)
			}
			p.markExited(exitErr)
		}
	}
}

// markExited records the exit once.
func (p *process) markExited(err error) {
	p.exitOnce.Do(func() {
		p.exitErr = err
		close(p.exited)
	})
}

// hasExited reports whether the openvpn process is gone.
func (p *process) hasExited() bool {
	select {
	case <-p.exited:
		return true
	default:
		return false
	}
}

// exitError describes why openvpn exited, using its own log lines.
func (p *process) exitError() error {
	return fmt.Errorf("openvpn exited (%v): %s", p.exitErr, p.log.lastErrors())
}

// stop signals the openvpn process to disconnect and waits for exit.
// OpenVPN serves one management client at a time, so the SIGTERM goes
// through our connection; the helper then waits for the exit (or kills).
func (p *process) stop() error {
	if p.mgmt != nil {
		_ = p.mgmt.Signal("SIGTERM")
		p.mgmt.Close()
	}
	if p.ops != nil {
		return p.stopHelperSession()
	}
	select {
	case <-p.exited:
	case <-time.After(10 * time.Second):
		p.kill()
	}
	return nil
}

// stopHelperSession ends the helper session and stops polling.
func (p *process) stopHelperSession() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := p.ops.OpenVPNStop(ctx, p.session)
	if errors.Is(err, privops.ErrNotFound) {
		err = nil
	}
	if p.stopPoll != nil {
		select {
		case <-p.stopPoll:
		default:
			close(p.stopPoll)
		}
	}
	p.markExited(nil)
	return err
}

func (p *process) kill() {
	if p.ops != nil {
		_ = p.stopHelperSession()
		return
	}
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	<-p.exited
}

// waitForMgmt polls until the management interface is reachable, failing
// fast (with openvpn's own error lines) if the process dies first.
func (p *process) waitForMgmt(password string, timeout time.Duration) (*mgmtClient, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p.hasExited() {
			return nil, p.exitError()
		}
		if c, err := newMgmtClientAuth(p.mgmtAddr, password); err == nil {
			return c, nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for management interface at %s: %s", p.mgmtAddr, p.log.lastErrors())
}

// freePort finds a free TCP port on localhost by binding to :0.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}
