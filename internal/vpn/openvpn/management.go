package openvpn

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// mgmtClient communicates with the OpenVPN management interface over TCP.
// Protocol: line-based text, commands sent as strings, responses parsed by prefix.
type mgmtClient struct {
	mu   sync.Mutex
	conn net.Conn
	r    *bufio.Reader
}

// newMgmtClient connects to the OpenVPN management interface at addr.
// OpenVPN must have been started with --management <addr> <port>.
func newMgmtClient(addr string) (*mgmtClient, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("mgmt: connect to %s: %w", addr, err)
	}
	c := &mgmtClient{
		conn: conn,
		r:    bufio.NewReader(conn),
	}
	// Consume the welcome banner.
	if err := c.readUntil(">INFO"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("mgmt: waiting for banner: %w", err)
	}
	return c, nil
}

// State returns the current VPN state string from OpenVPN.
// Response format: TIMESTAMP,STATE,DESCRIPTION,LOCAL_IP,REMOTE_IP,...
func (c *mgmtClient) State() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.send("state"); err != nil {
		return "", err
	}
	return c.readLine()
}

// SendAuth sends username and password through the management interface.
func (c *mgmtClient) SendAuth(username, password string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.send(fmt.Sprintf("username %q %s", "Auth", username)); err != nil {
		return err
	}
	if err := c.send(fmt.Sprintf("password %q %s", "Auth", password)); err != nil {
		return err
	}
	return nil
}

// Signal sends a signal to the OpenVPN process via the management interface.
func (c *mgmtClient) Signal(sig string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.send("signal " + sig)
}

// ByteCounts enables periodic byte count notifications.
func (c *mgmtClient) ByteCounts(intervalSec int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.send(fmt.Sprintf("bytecount %d", intervalSec))
}

// StatusV2 returns the full status output (STATUS 2 format).
func (c *mgmtClient) StatusV2() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.send("status 2"); err != nil {
		return "", err
	}
	var sb strings.Builder
	for {
		line, err := c.readLine()
		if err != nil {
			return "", err
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
		if strings.HasPrefix(line, "END") {
			break
		}
	}
	return sb.String(), nil
}

func (c *mgmtClient) send(cmd string) error {
	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := fmt.Fprintf(c.conn, "%s\n", cmd)
	return err
}

func (c *mgmtClient) readLine() (string, error) {
	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	line, err := c.r.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

func (c *mgmtClient) readUntil(prefix string) error {
	for {
		line, err := c.readLine()
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, prefix) {
			return nil
		}
	}
}

func (c *mgmtClient) Close() error {
	return c.conn.Close()
}

// parseState parses the OpenVPN state string into its components.
// Format: TIMESTAMP,STATE,DESCRIPTION,LOCAL_IP,REMOTE_IP
func parseState(s string) (state, localIP string) {
	parts := strings.SplitN(s, ",", 6)
	if len(parts) < 2 {
		return "UNKNOWN", ""
	}
	if len(parts) >= 4 {
		return parts[1], parts[3]
	}
	return parts[1], ""
}
