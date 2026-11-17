package security

import (
	"context"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// privTimeout bounds one privileged call. Enable/disable and DNS apply/
// restore run on the daemon's connect, disconnect and shutdown paths, so a
// wedged helper or system tool must never block them indefinitely.
const privTimeout = 30 * time.Second

// opsFunc returns the privileged implementation: in-process when elevated,
// else the Kongtrol helper service (ErrHelperNotInstalled when missing,
// rendered with privops.GuidanceKey). Tests replace it.
var opsFunc = privops.Default

func withOps(fn func(ctx context.Context, ops privops.Ops) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), privTimeout)
	defer cancel()
	ops, err := opsFunc(ctx)
	if err != nil {
		return err
	}
	return fn(ctx, ops)
}
