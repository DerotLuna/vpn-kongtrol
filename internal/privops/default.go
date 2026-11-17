package privops

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	localOnce  sync.Once
	localOps   *Local
	clientOnce sync.Once
	sharedCl   *Client
)

// Default returns the Ops implementation for this process:
//   - already elevated (admin terminal / root / the helper) → a shared Local;
//   - helper installed and answering → the process's shared Client (one
//     lease per process, renewed in the background once it owns state);
//   - otherwise ErrHelperNotInstalled (show GuidanceKey's message:
//     "Instala el ayudante una vez: kongtrol service install"), or
//     ErrHelperOutdated when the helper speaks another protocol version.
//
// It checks the helper on every call, so a helper installed while the
// daemon runs is picked up without a restart.
func Default(ctx context.Context) (Ops, error) {
	if IsElevated() {
		localOnce.Do(func() { localOps = NewLocal(LocalOptions{}) })
		return localOps, nil
	}
	clientOnce.Do(func() { sharedCl = NewClient() })
	c := sharedCl
	pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	res, err := c.Ping(pctx)
	if err != nil {
		if errors.Is(err, ErrHelperOutdated) || errors.Is(err, ErrDenied) {
			return nil, err
		}
		return nil, ErrHelperNotInstalled
	}
	if res.Protocol != ProtocolVersion {
		return nil, ErrHelperOutdated
	}
	return c, nil
}

// Available reports whether privileged operations can run in this process
// (elevated, or the helper answers), without returning the implementation.
func Available(ctx context.Context) error {
	_, err := Default(ctx)
	return err
}
