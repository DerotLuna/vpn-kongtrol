//go:build windows

package forticlient

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// guiConnect automates the FortiClient GUI to fill credentials and click Connect.
// Used as primary connect method on Windows when EMS policy blocks /vpnconnect CLI.
//
// Layout measured from FortiClient 6.4.10 at any window size (relative positions):
//   VPN dropdown center:    54.1% x, 58.6% y
//   Username field center:  54.1% x, 62.4% y
//   Password field center:  54.1% x, 66.6% y
//   Connect button center:  49.7% x, 75.4% y
func guiConnect(tunnelName, username, password string) error {
	script := buildGuiScript("connect", tunnelName, username, password)
	tmpDir, err := os.MkdirTemp("", "kongtrol-fcgui-*")
	if err != nil {
		return fmt.Errorf("forticlient gui: temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "fc_connect.ps1")
	if err := os.WriteFile(scriptPath, []byte(script), 0600); err != nil {
		return fmt.Errorf("forticlient gui: write script: %w", err)
	}

	out, err := runCmd("powershell", "-NoProfile", "-NonInteractive",
		"-ExecutionPolicy", "Bypass", "-File", scriptPath)
	if err != nil {
		return fmt.Errorf("forticlient gui: connect script: %w (output: %s)", err, out)
	}
	if strings.Contains(out, "ERROR:") {
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "ERROR:") {
				return fmt.Errorf("forticlient gui: %s", strings.TrimPrefix(strings.TrimSpace(line), "ERROR: "))
			}
		}
	}
	return nil
}

// guiDisconnect automates the FortiClient GUI to click Disconnect.
// When connected, FortiClient replaces the Connect button with Disconnect at the same position.
func guiDisconnect(tunnelName string) error {
	script := buildGuiScript("disconnect", tunnelName, "", "")
	tmpDir, err := os.MkdirTemp("", "kongtrol-fcgui-*")
	if err != nil {
		return fmt.Errorf("forticlient gui: temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "fc_disconnect.ps1")
	if err := os.WriteFile(scriptPath, []byte(script), 0600); err != nil {
		return fmt.Errorf("forticlient gui: write script: %w", err)
	}

	out, err := runCmd("powershell", "-NoProfile", "-NonInteractive",
		"-ExecutionPolicy", "Bypass", "-File", scriptPath)
	if err != nil {
		return fmt.Errorf("forticlient gui: disconnect script: %w (output: %s)", err, out)
	}
	if strings.Contains(out, "ERROR:") {
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "ERROR:") {
				return fmt.Errorf("forticlient gui: %s", strings.TrimPrefix(strings.TrimSpace(line), "ERROR: "))
			}
		}
	}
	return nil
}

// buildGuiScript generates the PowerShell GUI automation script.
// action is "connect" or "disconnect".
// Passwords are passed via temp clipboard to avoid SendKeys special-character escaping.
func buildGuiScript(action, tunnelName, username, password string) string {
	binary := binaryPath()

	// Escape single quotes in strings passed into PS script.
	escapedTunnel := strings.ReplaceAll(tunnelName, "'", "''")
	escapedUser := strings.ReplaceAll(username, "'", "''")
	escapedPass := strings.ReplaceAll(password, "'", "''")
	escapedBinary := strings.ReplaceAll(binary, "'", "''")

	return fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
using System.Threading;
public class FC {
    [StructLayout(LayoutKind.Sequential)]
    public struct RECT { public int left, top, right, bottom; }

    [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h, int n);
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
    [DllImport("user32.dll")] public static extern bool BringWindowToTop(IntPtr h);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, ref RECT r);
    [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
    [DllImport("user32.dll")] public static extern void mouse_event(uint flags, int x, int y, uint data, int extra);
    [DllImport("user32.dll")] public static extern IntPtr FindWindow(string cls, string title);
    [DllImport("user32.dll")] public static extern IntPtr FindWindowEx(IntPtr p, IntPtr a, string cls, string title);
    [DllImport("user32.dll")] public static extern IntPtr SendMessage(IntPtr h, uint msg, IntPtr w, IntPtr l);
    [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
    [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
    [DllImport("user32.dll")] public static extern bool AttachThreadInput(uint a, uint b, bool attach);
    [DllImport("kernel32.dll")] public static extern uint GetCurrentThreadId();

    public const uint ME_LDOWN = 0x0002;
    public const uint ME_LUP   = 0x0004;
    public const uint BM_CLICK = 0x00F5;

    // ForceForeground works around Windows 10/11 SetForegroundWindow restrictions
    // by attaching to the current foreground window's input thread first.
    public static void ForceForeground(IntPtr target) {
        ShowWindow(target, 9); // SW_RESTORE
        uint pid;
        uint targetThread = GetWindowThreadProcessId(target, out pid);
        uint currentThread = GetCurrentThreadId();
        IntPtr fgWnd = GetForegroundWindow();
        uint fgThread = GetWindowThreadProcessId(fgWnd, out pid);
        if (fgThread != currentThread) {
            AttachThreadInput(currentThread, fgThread, true);
        }
        SetForegroundWindow(target);
        BringWindowToTop(target);
        if (fgThread != currentThread) {
            AttachThreadInput(currentThread, fgThread, false);
        }
        Thread.Sleep(200);
    }

    public static void Click(int screenX, int screenY) {
        SetCursorPos(screenX, screenY);
        Thread.Sleep(100);
        mouse_event(ME_LDOWN, 0, 0, 0, 0);
        Thread.Sleep(80);
        mouse_event(ME_LUP, 0, 0, 0, 0);
        Thread.Sleep(100);
    }

    // DismissDialog finds a Win32 dialog by title, then clicks a button by label.
    public static bool DismissDialog(string dialogTitle, string buttonLabel) {
        IntPtr dlg = FindWindow(null, dialogTitle);
        if (dlg == IntPtr.Zero) return false;
        IntPtr btn = FindWindowEx(dlg, IntPtr.Zero, "Button", buttonLabel);
        if (btn == IntPtr.Zero) return false;
        ForceForeground(dlg);
        Thread.Sleep(100);
        SendMessage(btn, BM_CLICK, IntPtr.Zero, IntPtr.Zero);
        return true;
    }
}
'@

$binary    = '%s'
$tunnel    = '%s'
$username  = '%s'
$password  = '%s'
$action    = '%s'

# ── 1. Ensure FortiClient is running ─────────────────────────────────────────
$proc = Get-Process "FortiClient" -ErrorAction SilentlyContinue |
        Where-Object { $_.MainWindowHandle -ne 0 } | Select-Object -First 1

if (-not $proc) {
    if (-not (Test-Path $binary)) {
        Write-Output "ERROR: FortiClient not found at $binary"
        exit 1
    }
    # Launch without inheriting our console so Electron debug output
    # does not appear in the kongtrol terminal.
    Start-Process $binary -RedirectStandardOutput "$env:TEMP\fc_launch_out.txt" -RedirectStandardError "$env:TEMP\fc_launch_err.txt"
    $deadline = (Get-Date).AddSeconds(25)
    while (-not $proc -and (Get-Date) -lt $deadline) {
        Start-Sleep -Milliseconds 800
        $proc = Get-Process "FortiClient" -ErrorAction SilentlyContinue |
                Where-Object { $_.MainWindowHandle -ne 0 } | Select-Object -First 1
    }
    if (-not $proc) { Write-Output "ERROR: FortiClient window did not appear after 25s"; exit 1 }
    # Extra wait for CEF/Electron to finish rendering the VPN login form.
    Start-Sleep -Seconds 5
}

# ── 2. Restore and force foreground ──────────────────────────────────────────
# AttachThreadInput bypasses Windows 10/11 focus-steal restrictions so the
# window actually receives keyboard/mouse input, not just a taskbar flash.
$hwnd = [IntPtr]$proc.MainWindowHandle
[FC]::ForceForeground($hwnd)
# Wait for CEF content to be fully rendered and interactive.
# FortiClient Electron can take 2-3s after window appears before inputs accept events.
Start-Sleep -Milliseconds 2500

# ── 3. Get window rect ────────────────────────────────────────────────────────
$rect = New-Object FC+RECT
[FC]::GetWindowRect($hwnd, [ref]$rect) | Out-Null
$left = $rect.left; $top = $rect.top
$w    = $rect.right  - $rect.left
$h    = $rect.bottom - $rect.top

if ($w -lt 400 -or $h -lt 300) {
    Write-Output "ERROR: Window too small ($($w)x$($h)) — still minimized?"
    exit 1
}

Write-Output "Window $($w)x$($h) at ($left,$top)"

# Relative positions measured from FortiClient 6.4.10 screenshots.
# Connect/Disconnect button is at the same position in both states.
$btnX = $left + [int]($w * 0.497)
$btnY = $top  + [int]($h * 0.754)

if ($action -eq 'connect') {
    $vpnX  = $left + [int]($w * 0.541)
    $vpnY  = $top  + [int]($h * 0.586)
    $userX = $left + [int]($w * 0.541)
    $userY = $top  + [int]($h * 0.624)
    $passX = $left + [int]($w * 0.541)
    $passY = $top  + [int]($h * 0.666)

    if ($tunnel -ne '') {
        Write-Output "Selecting VPN tunnel at ($vpnX,$vpnY)..."
        [FC]::ForceForeground($hwnd)
        for ($i = 0; $i -lt 2; $i++) {
            [FC]::Click($vpnX, $vpnY)
            Start-Sleep -Milliseconds 120
        }
        [System.Windows.Forms.SendKeys]::SendWait("^a")
        Start-Sleep -Milliseconds 100
        [System.Windows.Forms.Clipboard]::SetText($tunnel)
        [System.Windows.Forms.SendKeys]::SendWait("^v")
        Start-Sleep -Milliseconds 120
        [System.Windows.Forms.SendKeys]::SendWait("{ENTER}")
        Start-Sleep -Milliseconds 350
    }

    Write-Output "Filling username at ($userX,$userY)..."
    # Re-assert foreground before clicking — CEF may have stolen focus during load.
    [FC]::ForceForeground($hwnd)
    # Retry click up to 3 times to ensure CEF field gets focus.
    for ($i = 0; $i -lt 3; $i++) {
        [FC]::Click($userX, $userY)
        Start-Sleep -Milliseconds 150
    }
    [System.Windows.Forms.SendKeys]::SendWait("^a")
    Start-Sleep -Milliseconds 120
    # Use clipboard to avoid SendKeys special-char escaping of +, ^, %%, ~, ()
    [System.Windows.Forms.Clipboard]::SetText($username)
    [System.Windows.Forms.SendKeys]::SendWait("^v")
    Start-Sleep -Milliseconds 250

    # Tab to password field — more reliable than coordinate click for the 2nd field.
    Write-Output "Tabbing to password field..."
    [System.Windows.Forms.SendKeys]::SendWait("{TAB}")
    Start-Sleep -Milliseconds 200
    [System.Windows.Forms.SendKeys]::SendWait("^a")
    Start-Sleep -Milliseconds 120
    [System.Windows.Forms.Clipboard]::SetText($password)
    [System.Windows.Forms.SendKeys]::SendWait("^v")
    Start-Sleep -Milliseconds 250

    # Clear clipboard so password is not left behind.
    [System.Windows.Forms.Clipboard]::Clear()

    Write-Output "Clicking Connect at ($btnX,$btnY)..."
    [FC]::Click($btnX, $btnY)

    # ── 4. Dismiss Security Alert (untrusted certificate) ─────────────────────
    # FortiClient shows a Win32 "Security Alert" dialog after clicking Connect
    # when the server has a self-signed / enterprise cert. Automatically accept.
    Write-Output "Waiting for Security Alert dialog..."
    $alertDeadline = (Get-Date).AddSeconds(15)
    $dismissed = $false
    while (-not $dismissed -and (Get-Date) -lt $alertDeadline) {
        # Try both common dialog titles and button labels.
        foreach ($title in @("Security Alert", "Alerta de seguridad", "Security Warning", "Certificate Warning")) {
            foreach ($btn in @("Yes", "&Yes", "Continue", "Sí", "Si", "&Sí", "&Si")) {
                if ([FC]::DismissDialog($title, $btn)) {
                    Write-Output "OK: '$title' dialog dismissed (clicked '$btn')"
                    $dismissed = $true
                    break
                }
            }
            if ($dismissed) { break }
        }
        if (-not $dismissed) { Start-Sleep -Milliseconds 300 }
    }
    if (-not $dismissed) {
        Write-Output "INFO: No Security Alert dialog appeared within 15s (may not be needed)"
    }

    # ── 5. Fail fast on common auth/access dialogs ────────────────────────────
    # Avoid waiting full tunnel timeout when FortiClient already reported failure.
    $failDeadline = (Get-Date).AddSeconds(20)
    while ((Get-Date) -lt $failDeadline) {
        $failed = $false
        foreach ($title in @(
            "Access denied", "Acceso denegado",
            "Authentication failed", "Error de autenticación",
            "Login failed", "Connection failed",
            "Error", "FortiClient Error"
        )) {
            foreach ($btn in @("OK", "&OK", "Aceptar", "Close", "Cerrar")) {
                if ([FC]::DismissDialog($title, $btn)) {
                    Write-Output "ERROR: FortiClient connection failed ($title)"
                    $failed = $true
                    break
                }
            }
            if ($failed) { break }
        }
        if ($failed) { exit 2 }
        Start-Sleep -Milliseconds 300
    }

    Write-Output "OK: connect sequence complete"

} elseif ($action -eq 'disconnect') {
    Write-Output "Clicking Disconnect at ($btnX,$btnY)..."
    [FC]::Click($btnX, $btnY)

    # Dismiss any confirmation dialog that may appear on disconnect.
    Start-Sleep -Milliseconds 500
    foreach ($title in @("Disconnect", "FortiClient", "Confirm")) {
        foreach ($btn in @("Yes", "&Yes", "OK", "Disconnect")) {
            [FC]::DismissDialog($title, $btn) | Out-Null
        }
    }

    Write-Output "OK: disconnect sequence complete"
}
`, escapedBinary, escapedTunnel, escapedUser, escapedPass, action)
}
