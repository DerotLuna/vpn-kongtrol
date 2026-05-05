//go:build windows

package wireguard

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// wgUp installs and starts a WireGuard tunnel on Windows using wireguard.exe.
// Windows WireGuard does not use wg-quick; it manages tunnels as Windows services.
// /installtunnelservice requires admin — uses ShellExecuteEx runas to trigger UAC.
func wgUp(configPath string) error {
	iface := interfaceFromConfig(configPath)

	// If service exists AND interface is live, already running — nothing to do.
	if tunnelServiceExists(iface) && ifaceExists(iface) {
		return nil
	}

	// Remove stale kongtrol-managed tunnels left over from previous runs
	// (e.g. process crash before graceful disconnect). These use temp dir
	// names like "kongtrol-wg-281123272" and can block new tunnel creation.
	cleanStaleKongtrolTunnels(iface)

	// Service exists but interface is down (stopped/failed) — remove stale service
	// before reinstalling so wireguard.exe gets a clean slate.
	if tunnelServiceExists(iface) {
		_ = wgDown(iface)
	}

	binary := wireguardBinary()
	if err := runElevated(binary, "/installtunnelservice", configPath); err != nil {
		return fmt.Errorf("wireguard /installtunnelservice: %w", err)
	}

	// Wait up to 8s for the service to reach Running (or fail early).
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if tunnelServiceRunning(iface) {
			// Prevent the service from auto-starting on reboot — kongtrol
			// owns the lifecycle, not Windows services.
			setServiceManualStart(iface)
			return nil // service is up; caller will waitForInterface
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Service didn't reach Running — query its actual state for a useful error.
	m2, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("wireguard service did not start (SCM query failed: %w)", err)
	}
	defer m2.Disconnect()
	s2, err := m2.OpenService("WireGuardTunnel$" + iface)
	if err != nil {
		return fmt.Errorf("wireguard service disappeared after install (check Event Viewer)")
	}
	defer s2.Close()
	st, err := s2.Query()
	if err != nil {
		return fmt.Errorf("wireguard service state unknown after install")
	}
	return fmt.Errorf("wireguard service did not reach Running state (state=%v); check Event Viewer → Windows Logs → Application for WireGuard errors", st.State)
}

// cleanStaleKongtrolTunnels removes WireGuard tunnel services left behind by
// previous kongtrol runs that exited without graceful disconnect. These have
// names matching "WireGuardTunnel$kongtrol-wg-*" from temp-dir-based configs.
// The currentIface is excluded (handled separately by the caller).
func cleanStaleKongtrolTunnels(currentIface string) {
	m, err := mgr.Connect()
	if err != nil {
		return
	}
	defer m.Disconnect()

	services, err := m.ListServices()
	if err != nil {
		return
	}

	const prefix = "WireGuardTunnel$kongtrol-wg-"
	for _, svcName := range services {
		if !strings.HasPrefix(svcName, prefix) {
			continue
		}
		ifaceName := strings.TrimPrefix(svcName, "WireGuardTunnel$")
		if ifaceName == currentIface {
			continue
		}
		_ = wgDown(ifaceName)
	}
}

// tunnelServiceExists returns true if the WireGuard tunnel Windows service exists.
func tunnelServiceExists(ifaceName string) bool {
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer m.Disconnect()
	s, err := m.OpenService("WireGuardTunnel$" + ifaceName)
	if err != nil {
		return false
	}
	s.Close()
	return true
}

// wgDown stops and removes the WireGuard tunnel service on Windows.
// WireGuard tunnel services are named "WireGuardTunnel$<tunnel_name>".
// We manage the service directly via the Windows SCM rather than relying on
// wireguard.exe CLI flags, which require the manager service to be running.
func wgDown(ifaceName string) error {
	if ifaceName == "" {
		return fmt.Errorf("wgDown: tunnel name is empty")
	}
	serviceName := "WireGuardTunnel$" + ifaceName

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("open SCM: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		// Service doesn't exist — already stopped.
		return nil
	}
	defer s.Close()

	// Stop the service if it's running.
	status, err := s.Query()
	if err == nil && status.State != svc.Stopped && status.State != svc.StopPending {
		if _, err := s.Control(svc.Stop); err != nil {
			return fmt.Errorf("stop service %q: %w", serviceName, err)
		}
		// Wait up to 10s for the service to stop.
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			status, err = s.Query()
			if err != nil || status.State == svc.Stopped {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
	}

	if err := s.Delete(); err != nil {
		return fmt.Errorf("delete service %q: %w", serviceName, err)
	}
	// Close handles before polling, otherwise the SCM entry may linger.
	s.Close()
	m.Disconnect()

	// Wait for SCM to fully remove the entry so a subsequent /installtunnelservice
	// does not collide with a service marked-for-deletion.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !tunnelServiceExists(ifaceName) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

// wgShow returns the output of `wg show <iface>` on Windows.
// The `wg` CLI on Windows communicates over a named pipe managed by wireguard.exe.
func wgShow(ifaceName string) (string, error) {
	return runCmd("wg", "show", ifaceName)
}

// setServiceManualStart changes the tunnel service start type to manual so
// it does not auto-start on reboot. Best-effort; errors are ignored.
func setServiceManualStart(ifaceName string) {
	m, err := mgr.Connect()
	if err != nil {
		return
	}
	defer m.Disconnect()
	s, err := m.OpenService("WireGuardTunnel$" + ifaceName)
	if err != nil {
		return
	}
	defer s.Close()
	c, err := s.Config()
	if err != nil {
		return
	}
	c.StartType = mgr.StartManual
	_ = s.UpdateConfig(c)
}

// tunnelServiceRunning returns true if the WireGuard tunnel Windows service
// is in the Running state. Used by Status() to avoid false error reports
// during the brief window between service start and interface appearance.
func tunnelServiceRunning(ifaceName string) bool {
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer m.Disconnect()
	s, err := m.OpenService("WireGuardTunnel$" + ifaceName)
	if err != nil {
		return false
	}
	defer s.Close()
	status, err := s.Query()
	if err != nil {
		return false
	}
	return status.State == svc.Running || status.State == svc.StartPending
}

// wgTransfer returns byte counts for a tunnel.
func wgTransfer(ifaceName string) (string, error) {
	return runCmd("wg", "show", ifaceName, "transfer")
}

func wireguardBinary() string {
	return filepath.Join("C:\\", "Program Files", "WireGuard", "wireguard.exe")
}

func wgBinary() string {
	return filepath.Join("C:\\", "Program Files", "WireGuard", "wg.exe")
}

// WgSetAllowedIPs updates AllowedIPs for a peer on a live WireGuard tunnel.
// On Windows, the WireGuard service updates both crypto-routing and OS routes
// automatically — no separate routeMgr.Add() needed.
// Requires Administrator (communicates via named pipe).
func WgSetAllowedIPs(ifaceName, peerPubKey string, cidrs []string) error {
	_, err := runCmd(wgBinary(), "set", ifaceName, "peer", peerPubKey,
		"allowed-ips", strings.Join(cidrs, ","))
	return err
}

// shellExecuteInfo mirrors the Win32 SHELLEXECUTEINFOW struct.
// https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-shellexecuteinfow
type shellExecuteInfo struct {
	cbSize         uint32
	fMask          uint32
	hwnd           uintptr
	lpVerb         *uint16
	lpFile         *uint16
	lpParameters   *uint16
	lpDirectory    *uint16
	nShow          int32
	hInstApp       uintptr
	lpIDList       uintptr
	lpClass        *uint16
	hkeyClass      uintptr
	dwHotKey       uint32
	hIconOrMonitor uintptr
	hProcess       windows.Handle
}

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	shellExecuteExW  = shell32.NewProc("ShellExecuteExW")
)

const (
	seeMaskNocloseprocess = 0x00000040
	swHide                = 0
)

// runElevated runs exe with args via ShellExecuteExW using the "runas" verb,
// which triggers a UAC prompt if the process is not already elevated.
// It waits for the child to exit and returns an error if it exits non-zero.
func runElevated(exe string, args ...string) error {
	verb, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	params, err := syscall.UTF16PtrFromString(strings.Join(args, " "))
	if err != nil {
		return err
	}

	info := shellExecuteInfo{
		fMask:      seeMaskNocloseprocess,
		lpVerb:     verb,
		lpFile:     file,
		lpParameters: params,
		nShow:      swHide,
	}
	info.cbSize = uint32(unsafe.Sizeof(info))

	ret, _, callErr := shellExecuteExW.Call(uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return fmt.Errorf("ShellExecuteExW: %w", callErr)
	}
	if info.hProcess == 0 {
		// Already elevated or instantaneous — treat as success.
		return nil
	}
	defer windows.CloseHandle(info.hProcess)

	if _, err := windows.WaitForSingleObject(info.hProcess, windows.INFINITE); err != nil {
		return fmt.Errorf("wait: %w", err)
	}

	var exitCode uint32
	if err := windows.GetExitCodeProcess(info.hProcess, &exitCode); err != nil {
		return fmt.Errorf("exit code: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("exit status %d", exitCode)
	}
	return nil
}
