//go:build windows

package wireguard

import (
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

// Tunnel services are created by the helper (or in-process when elevated)
// via privops; this file only reads their state, with the minimal SCM
// rights a standard user has (connect + enumerate, query status).

const wgServicePrefix = "WireGuardTunnel$"

// openSCMReadOnly opens the service manager for reading only.
// mgr.Connect asks for SC_MANAGER_ALL_ACCESS, which a standard user lacks.
func openSCMReadOnly() (*mgr.Mgr, error) {
	h, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT|windows.SC_MANAGER_ENUMERATE_SERVICE)
	if err != nil {
		return nil, err
	}
	return &mgr.Mgr{Handle: h}, nil
}

// serviceState returns the tunnel service state and whether it exists.
func serviceState(ifaceName string) (uint32, bool) {
	m, err := openSCMReadOnly()
	if err != nil {
		return 0, false
	}
	defer m.Disconnect()
	name, err := windows.UTF16PtrFromString(wgServicePrefix + ifaceName)
	if err != nil {
		return 0, false
	}
	h, err := windows.OpenService(m.Handle, name, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return 0, false
	}
	defer windows.CloseServiceHandle(h)
	var st windows.SERVICE_STATUS
	if err := windows.QueryServiceStatus(h, &st); err != nil {
		return 0, true
	}
	return st.CurrentState, true
}

// wgPreUp reports whether the tunnel is already up (service running and
// interface present) and removes tunnels left by earlier Kongtrol runs
// ("kongtrol-wg-*" from temp-dir configs) that can block a new one.
func wgPreUp(ifaceName string) bool {
	if st, ok := serviceState(ifaceName); ok && st == windows.SERVICE_RUNNING && ifaceExists(ifaceName) {
		return true
	}
	cleanStaleKongtrolTunnels(ifaceName)
	return false
}

func cleanStaleKongtrolTunnels(currentIface string) {
	m, err := openSCMReadOnly()
	if err != nil {
		return
	}
	services, err := m.ListServices()
	m.Disconnect()
	if err != nil {
		return
	}
	const prefix = wgServicePrefix + "kongtrol-wg-"
	for _, svcName := range services {
		if !strings.HasPrefix(svcName, prefix) {
			continue
		}
		if name := strings.TrimPrefix(svcName, wgServicePrefix); name != currentIface {
			_ = wgDown(name, "")
		}
	}
}

// tunnelServiceRunning returns true if the WireGuard tunnel service is
// running or starting. Used by Status() to avoid false error reports
// between service start and interface appearance.
func tunnelServiceRunning(ifaceName string) bool {
	st, ok := serviceState(ifaceName)
	return ok && (st == windows.SERVICE_RUNNING || st == windows.SERVICE_START_PENDING)
}

// wgShow returns the output of `wg show <iface>`. wg.exe talks to the
// tunnel's named pipe, which only administrators may open, so this fails
// for the unprivileged daemon and Status() falls back to the interface.
func wgShow(ifaceName string) (string, error) {
	return runCmd("wg", "show", ifaceName)
}

// wgTransfer returns byte counts for a tunnel (administrators only, see
// wgShow).
func wgTransfer(ifaceName string) (string, error) {
	return runCmd("wg", "show", ifaceName, "transfer")
}
