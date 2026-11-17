package openvpn

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
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

// mgmtPasswordPrompt is what openvpn prints (without a newline) to a new
// management client when it was started with a management password file,
// as the Kongtrol helper does.
const mgmtPasswordPrompt = "ENTER PASSWORD:"

// newMgmtClient connects to the OpenVPN management interface at addr.
// OpenVPN must have been started with --management <addr> <port>.
func newMgmtClient(addr string) (*mgmtClient, error) {
	return newMgmtClientAuth(addr, "")
}

// newMgmtClientAuth connects to the management interface and, when openvpn
// asks for it, answers the "ENTER PASSWORD:" prompt with password before
// waiting for the >INFO banner.
func newMgmtClientAuth(addr, password string) (*mgmtClient, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("mgmt: connect to %s: %w", addr, err)
	}
	c := &mgmtClient{
		conn: conn,
		r:    bufio.NewReader(conn),
	}
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	head, err := c.r.Peek(len(mgmtPasswordPrompt))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("mgmt: waiting for banner: %w", err)
	}
	if string(head) == mgmtPasswordPrompt {
		if password == "" {
			conn.Close()
			return nil, fmt.Errorf("mgmt: %s asks for a management password", addr)
		}
		_, _ = c.r.Discard(len(mgmtPasswordPrompt))
		if err := c.send(password); err != nil {
			conn.Close()
			return nil, fmt.Errorf("mgmt: send password: %w", err)
		}
	}
	// Consume the welcome banner ("SUCCESS: password is correct" first when
	// a password was asked; a wrong one gets "ERROR" and a closed socket).
	if err := c.readUntil(">INFO"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("mgmt: waiting for banner: %w", err)
	}
	return c, nil
}

// State returns the current VPN state line from OpenVPN.
// Response format: TIMESTAMP,STATE,DESCRIPTION,LOCAL_IP,REMOTE_IP,...
// followed by "END". Real-time notifications (">PASSWORD:", ">HOLD:", …)
// that arrive in between are skipped.
func (c *mgmtClient) State() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.send("state"); err != nil {
		return "", err
	}
	state := ""
	for {
		line, err := c.readLine()
		if err != nil {
			return state, err
		}
		switch {
		case line == "END":
			return state, nil
		case strings.HasPrefix(line, "ERROR"):
			return "", fmt.Errorf("mgmt: %s", line)
		case state == "" && line != "" && line[0] >= '0' && line[0] <= '9':
			// State lines start with a unix timestamp. Anything else is a
			// real-time ">..." notification or a leftover "SUCCESS: ..."
			// reply to an earlier username/password/signal command.
			state = line
		}
	}
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
	return c.statusOutput("status 2")
}

// statusOutput sends a status command and returns its lines up to END.
// Real-time ">..." notifications that arrive in between are dropped.
func (c *mgmtClient) statusOutput(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.send(cmd); err != nil {
		return "", err
	}
	var sb strings.Builder
	for {
		line, err := c.readLine()
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(line, ">") {
			continue
		}
		if strings.HasPrefix(line, "ERROR") {
			return "", fmt.Errorf("mgmt: %s", line)
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
		if strings.HasPrefix(line, "END") {
			break
		}
	}
	return sb.String(), nil
}

// ByteTotals returns the tunnel's link-level byte counters (sent, received)
// from the client-mode "status" statistics block — the same numbers the
// >BYTECOUNT notifications carry. Works the same when openvpn was spawned
// directly or by the helper: both share this authenticated connection.
func (c *mgmtClient) ByteTotals() (sent, received uint64, err error) {
	out, err := c.statusOutput("status")
	if err != nil {
		return 0, 0, err
	}
	sent, received, ok := parseStatusBytes(out)
	if !ok {
		return 0, 0, fmt.Errorf("mgmt: no byte counters in status output")
	}
	return sent, received, nil
}

// parseStatusBytes reads "TCP/UDP write bytes,N" (sent) and
// "TCP/UDP read bytes,N" (received) from client-mode status output.
func parseStatusBytes(out string) (sent, received uint64, ok bool) {
	var haveSent, haveRecv bool
	for _, line := range strings.Split(out, "\n") {
		key, val, found := strings.Cut(strings.TrimSpace(line), ",")
		if !found {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimSpace(val), 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "TCP/UDP write bytes":
			sent, haveSent = n, true
		case "TCP/UDP read bytes":
			received, haveRecv = n, true
		}
	}
	return sent, received, haveSent && haveRecv
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
