//go:build windows

package privops

import (
	"context"
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

// PipeName is the helper's named pipe.
const PipeName = `\\.\pipe\kongtrol-helper`

// dialHelper connects to the pipe and verifies that it is owned by SYSTEM
// or Administrators, so a standard user who created a pipe with the same
// name first ("pipe squatting") is never trusted.
func dialHelper(ctx context.Context) (net.Conn, error) {
	conn, err := winio.DialPipeContext(ctx, PipeName)
	if err != nil {
		return nil, err
	}
	if err := verifyPipeServer(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func verifyPipeServer(conn net.Conn) error {
	fd, ok := conn.(interface{ Fd() uintptr })
	if !ok {
		return fmt.Errorf("%w: cannot inspect helper pipe", ErrDenied)
	}
	sd, err := windows.GetSecurityInfo(windows.Handle(fd.Fd()), windows.SE_KERNEL_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("%w: read helper pipe owner: %v", ErrDenied, err)
	}
	owner, _, err := sd.Owner()
	if err != nil || !isAdminOrSystem(owner) {
		return fmt.Errorf("%w: helper pipe is not owned by the system", ErrDenied)
	}
	return nil
}
