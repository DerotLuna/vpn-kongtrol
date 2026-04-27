package openvpn

import (
	"fmt"
	"net"
	"os/exec"
	"time"
)

// process manages the lifecycle of an openvpn subprocess.
type process struct {
	cmd      *exec.Cmd
	mgmtAddr string // 127.0.0.1:<port>
	mgmt     *mgmtClient
}

// start launches the openvpn process with the given .ovpn config file.
// A free management port is allocated automatically to avoid conflicts
// when multiple OpenVPN instances run simultaneously.
func start(configPath, certPath, keyPath, username, password string) (*process, error) {
	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("openvpn: allocate management port: %w", err)
	}

	mgmtAddr := fmt.Sprintf("127.0.0.1:%d", port)

	args := []string{
		"--config", configPath,
		"--management", "127.0.0.1", fmt.Sprintf("%d", port),
		"--management-query-passwords",
		"--verb", "3",
	}

	if certPath != "" {
		args = append(args, "--cert", certPath, "--key", keyPath)
	}

	// Route-nopull: Kongtrol manages routes itself via the policy engine.
	// Remove this flag if you want OpenVPN to manage its own routes.
	args = append(args, "--route-nopull")

	cmd := exec.Command("openvpn", args...)
	setProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("openvpn: start process: %w", err)
	}

	p := &process{
		cmd:      cmd,
		mgmtAddr: mgmtAddr,
	}

	// Give the process time to bind the management port.
	mgmt, err := waitForMgmt(mgmtAddr, 15*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("openvpn: management interface not ready: %w", err)
	}
	p.mgmt = mgmt

	// Send credentials if provided.
	if username != "" || password != "" {
		if err := mgmt.SendAuth(username, password); err != nil {
			return nil, fmt.Errorf("openvpn: send credentials: %w", err)
		}
	}

	return p, nil
}

// stop signals the openvpn process to disconnect and waits for exit.
func (p *process) stop() error {
	if p.mgmt != nil {
		_ = p.mgmt.Signal("SIGTERM")
		p.mgmt.Close()
	}
	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	select {
	case <-time.After(10 * time.Second):
		_ = p.cmd.Process.Kill()
		return <-done
	case err := <-done:
		return err
	}
}

// waitForMgmt polls until the management interface is reachable.
func waitForMgmt(addr string, timeout time.Duration) (*mgmtClient, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c, err := newMgmtClient(addr)
		if err == nil {
			return c, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for management interface at %s", addr)
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
