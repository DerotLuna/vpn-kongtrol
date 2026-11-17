//go:build windows

package openvpn

import (
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// lifetimeHandle is the Job Object openvpn.exe is assigned to.
type lifetimeHandle = windows.Handle

// setProcAttr configures the command so the child process is in its own
// process group (CREATE_NEW_PROCESS_GROUP), so Ctrl+C events don't propagate
// to the child unintentionally, and with no console window of its own
// (HideWindow) — openvpn.exe is a console-subsystem binary, and a parent
// with no console (the tray, built with -H=windowsgui) would otherwise get
// a visible console window for the tunnel's entire lifetime.
func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
}

// attachLifetime assigns openvpn.exe to a Job Object flagged
// JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE. Kongtrol holds the only handle to the
// job, so when Kongtrol exits for any reason (including a crash or being
// killed) the kernel closes the handle and terminates openvpn with it.
// Best effort: on failure the job is discarded and nil-equivalent state kept.
func attachLifetime(p *process) error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return err
	}
	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false, uint32(p.cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return err
	}
	defer windows.CloseHandle(proc)
	if err := windows.AssignProcessToJobObject(job, proc); err != nil {
		_ = windows.CloseHandle(job)
		return err
	}
	p.lifetime = job
	return nil
}

// releaseLifetime closes the job handle. Called only after openvpn has
// exited, so KILL_ON_JOB_CLOSE has nothing left to terminate.
func releaseLifetime(p *process) {
	if p.lifetime != 0 {
		_ = windows.CloseHandle(p.lifetime)
		p.lifetime = 0
	}
}
