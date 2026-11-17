//go:build windows

package privhelper

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// ServiceName is the Windows service name.
const ServiceName = "KongtrolHelper"

func programFilesDir() string {
	p, err := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, 0)
	if err != nil || p == "" {
		p = `C:\Program Files`
	}
	return filepath.Join(p, "Kongtrol")
}

// BinaryPath is where `service install` copies the helper executable.
func BinaryPath() string { return filepath.Join(programFilesDir(), "kongtrol-helper.exe") }

// ConfigPath is helper.json (admin-only directory).
func ConfigPath() string { return filepath.Join(privops.ProgramDataDir(), "helper.json") }

// AuditPath is the helper audit log.
func AuditPath() string { return filepath.Join(privops.ProgramDataDir(), "helper-audit.log") }

func validateUserID(id string) error {
	sid, err := windows.StringToSid(id)
	if err != nil || !sid.IsValid() {
		return fmt.Errorf("privhelper: invalid user SID %q", id)
	}
	return nil
}

func checkConfigFileTrusted(path string) error {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil ||
		!(owner.IsWellKnown(windows.WinLocalSystemSid) || owner.IsWellKnown(windows.WinBuiltinAdministratorsSid)) {
		return errors.New("privhelper: helper.json is not owned by Administrators/SYSTEM")
	}
	return nil
}

// pipeSDDL: owner Administrators; deny network logons; full access for
// SYSTEM and Administrators; read/write (connect) for the installing user.
func pipeSDDL(userSID string) string {
	return "O:BAG:BAD:P(D;;GA;;;NU)(A;;GA;;;SY)(A;;GA;;;BA)(A;;GRGW;;;" + userSID + ")"
}

func listen(cfg Config) (net.Listener, error) {
	return winio.ListenPipe(privops.PipeName, &winio.PipeConfig{
		SecurityDescriptor: pipeSDDL(cfg.UserID),
		InputBufferSize:    64 << 10,
		OutputBufferSize:   64 << 10,
	})
}

// authenticator identifies the client process's user from its token and
// hands Local an impersonation token for file reads.
func authenticator(cfg Config) Authenticator {
	return func(conn net.Conn) (privops.Caller, func(), error) {
		fd, ok := conn.(interface{ Fd() uintptr })
		if !ok {
			return privops.Caller{}, nil, errors.New("not a pipe")
		}
		var pid uint32
		if err := windows.GetNamedPipeClientProcessId(windows.Handle(fd.Fd()), &pid); err != nil {
			return privops.Caller{}, nil, err
		}
		proc, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
		if err != nil {
			return privops.Caller{}, nil, err
		}
		defer windows.CloseHandle(proc)
		var tok windows.Token
		if err := windows.OpenProcessToken(proc, windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE, &tok); err != nil {
			return privops.Caller{}, nil, err
		}
		defer tok.Close()
		u, err := tok.GetTokenUser()
		if err != nil {
			return privops.Caller{}, nil, err
		}
		c := privops.Caller{ID: u.User.Sid.String(), UID: -1,
			Admin: tok.IsElevated() || u.User.Sid.IsWellKnown(windows.WinLocalSystemSid)}
		c.Name = c.ID
		if acct, dom, _, err := u.User.Sid.LookupAccount(""); err == nil {
			c.Name = dom + `\` + acct
		}
		if err := authorize(cfg, c); err != nil {
			return c, nil, fmt.Errorf("%w: %s (pid %d)", err, c.Name, pid)
		}
		var imp windows.Token
		if err := windows.DuplicateTokenEx(tok, windows.TOKEN_IMPERSONATE|windows.TOKEN_QUERY, nil,
			windows.SecurityImpersonation, windows.TokenImpersonation, &imp); err != nil {
			return c, nil, err
		}
		c.Token = uintptr(imp)
		return c, func() { _ = imp.Close() }, nil
	}
}

// ── Service entrypoint ──────────────────────────────────────────────────

type handler struct{}

func (handler) Execute(_ []string, req <-chan svc.ChangeRequest, st chan<- svc.Status) (bool, uint32) {
	st <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx) }()
	st <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case err := <-done:
			cancel()
			if err != nil {
				return true, 1
			}
			return false, 0
		case c := <-req:
			switch c.Cmd {
			case svc.Interrogate:
				st <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				st <- svc.Status{State: svc.StopPending}
				cancel()
				select {
				case <-done:
				case <-time.After(20 * time.Second):
				}
				return false, 0
			}
		}
	}
}

// Run is the `kongtrol helper run` entrypoint: under the SCM it runs as a
// service; otherwise (debugging from an elevated terminal) in the foreground.
func Run(ctx context.Context) error {
	if is, err := svc.IsWindowsService(); err == nil && is {
		return svc.Run(ServiceName, handler{})
	}
	if !privops.IsElevated() {
		return errors.New("privhelper: run requires administrator rights")
	}
	return serve(ctx)
}

// ── Install / uninstall / status ────────────────────────────────────────

// Install copies the executable to Program Files, writes helper.json,
// registers the LocalSystem service and starts it. Must run elevated.
func Install(o InstallOptions) error {
	if !privops.IsElevated() {
		return errors.New("privhelper: install requires administrator rights")
	}
	data, err := encodeConfig(Config{Format: ConfigFormat, UserID: o.UserID, UserName: o.UserName, GID: -1,
		Version: o.Version, InstalledAt: time.Now().UTC()})
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	if s, err := m.OpenService(ServiceName); err == nil {
		stopService(s)
		s.Close()
	}
	if err := privops.EnsurePrivateDir(programFilesDir()); err != nil {
		return fmt.Errorf("privhelper: %s: %w", programFilesDir(), err)
	}
	if err := copyExecutable(o.SourceExe, BinaryPath()); err != nil {
		return err
	}
	if err := privops.EnsurePrivateDir(privops.ProgramDataDir()); err != nil {
		return err
	}
	if err := os.WriteFile(ConfigPath(), data, 0o600); err != nil {
		return err
	}
	bin := BinaryPath()
	cmdline := windows.EscapeArg(bin) + " helper run"
	s, err := m.OpenService(ServiceName)
	if err == nil {
		c, cerr := s.Config()
		if cerr == nil {
			c.BinaryPathName, c.StartType, c.ServiceStartName = cmdline, mgr.StartAutomatic, "LocalSystem"
			cerr = s.UpdateConfig(c)
		}
		if cerr != nil {
			s.Close()
			return cerr
		}
	} else {
		s, err = m.CreateService(ServiceName, bin, mgr.Config{
			DisplayName:  "Kongtrol Helper",
			Description:  "Runs Kongtrol's network changes (routes, DNS, firewall, VPN tunnels) so no admin terminal is needed.",
			StartType:    mgr.StartAutomatic,
			ErrorControl: mgr.ErrorNormal,
		}, "helper", "run")
		if err != nil {
			return err
		}
	}
	defer s.Close()
	_ = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}, 86400)
	if err := s.Start(); err != nil {
		return fmt.Errorf("privhelper: start service: %w", err)
	}
	return waitState(s, svc.Running, 15*time.Second)
}

func stopService(s *mgr.Service) {
	if st, err := s.Query(); err == nil && st.State != svc.Stopped {
		_, _ = s.Control(svc.Stop)
		_ = waitState(s, svc.Stopped, 20*time.Second)
	}
}

func waitState(s *mgr.Service, want svc.State, d time.Duration) error {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		st, err := s.Query()
		if err != nil {
			return err
		}
		if st.State == want {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("privhelper: service did not reach state %d", want)
}

// Uninstall stops and deletes the service and removes its files.
func Uninstall() error {
	if !privops.IsElevated() {
		return errors.New("privhelper: uninstall requires administrator rights")
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	if s, err := m.OpenService(ServiceName); err == nil {
		stopService(s)
		derr := s.Delete()
		s.Close()
		if derr != nil {
			m.Disconnect()
			return derr
		}
	}
	m.Disconnect()
	var errs []error
	for _, p := range []string{programFilesDir(), privops.ProgramDataDir()} {
		if err := os.RemoveAll(p); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// QueryStatus inspects the SCM (works unprivileged).
func QueryStatus() State {
	m, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return State{}
	}
	defer windows.CloseServiceHandle(m)
	name, _ := windows.UTF16PtrFromString(ServiceName)
	h, err := windows.OpenService(m, name, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return State{}
	}
	s := &mgr.Service{Name: ServiceName, Handle: h}
	defer s.Close()
	st := State{Installed: true, BinaryPath: BinaryPath()}
	if q, err := s.Query(); err == nil {
		st.Running = q.State == svc.Running
	}
	return st
}

// chownRoot is a no-op on Windows: the directory's protected DACL applies.
func chownRoot(string, os.FileMode) error { return nil }
