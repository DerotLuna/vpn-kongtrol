package doctor

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
	"golang.org/x/sys/windows"
)

type tunnelBearDevice struct {
	Name                   string
	ConfigManagerErrorCode *uint32
}

func (d *run) checkTunnelBearDriver() {
	installed := false
	for _, probe := range vpndetect.Probes() {
		if probe.Label == "TunnelBear" {
			_, installed = vpndetect.Detect(probe)
			break
		}
	}
	devices, err := windowsTunnelBearDevices()
	d.reportTunnelBearDriver(installed, devices, err)
}

func (d *run) reportTunnelBearDriver(installed bool, devices []tunnelBearDevice, err error) {
	label := d.t("doctor.tunnelbear.label")
	hint := d.t("doctor.tunnelbear.hint")
	if err != nil {
		if installed {
			d.warn(label, d.t("doctor.tunnelbear.unavailable"), d.t("doctor.hint.retry"))
		}
		return
	}
	if installed && len(devices) == 0 {
		d.add(Check{Label: label, Status: StatusWarn, Detail: d.t("doctor.tunnelbear.missing"), Hint: hint, Help: "tunnelbear_driver"})
	}
	for _, device := range devices {
		if device.ConfigManagerErrorCode != nil && *device.ConfigManagerErrorCode != 0 {
			d.add(Check{Label: label, Status: StatusWarn, Detail: d.f("doctor.tunnelbear.device_error", device.Name, *device.ConfigManagerErrorCode), Hint: hint, Help: "tunnelbear_driver"})
		}
	}
}

func windowsTunnelBearDevices() ([]tunnelBearDevice, error) {
	dir, err := windows.GetSystemDirectory()
	if err != nil {
		return nil, err
	}
	const script = `$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [Text.Encoding]::UTF8
$devices = @(Get-CimInstance Win32_PnPEntity -Filter "PNPClass='Net' AND (Name LIKE 'TunnelBear Adapter%' OR Name LIKE 'TunnelBear Tunnel%')" |
  Select-Object Name, ConfigManagerErrorCode)
ConvertTo-Json -InputObject $devices -Compress`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(dir, "WindowsPowerShell", "v1.0", "powershell.exe"),
		"-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var devices []tunnelBearDevice
	err = json.Unmarshal(output, &devices)
	return devices, err
}
