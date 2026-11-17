package privhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// leaseReapEvery is how often expired client leases are checked.
const leaseReapEvery = time.Second

// serve loads helper.json, prepares the state directory and serves
// requests until ctx is cancelled. OpenVPN sessions die with the helper.
func serve(ctx context.Context) error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("privhelper: load config: %w", err)
	}
	if err := privops.EnsurePrivateDir(privops.DefaultStateDir()); err != nil {
		return fmt.Errorf("privhelper: state dir: %w", err)
	}
	local := privops.NewLocal(privops.LocalOptions{TrustedOwners: []string{cfg.UserID}})
	defer local.StopAll()
	audit := OpenAudit(AuditPath())
	failOpenAtStart(ctx, local, audit)
	l, err := listen(cfg)
	if err != nil {
		return fmt.Errorf("privhelper: listen: %w", err)
	}
	go local.RunLeaseReaper(ctx, leaseReapEvery, func(rel []privops.ExpiredRelease) {
		args, _ := json.Marshal(rel)
		audit.Record(AuditEntry{Time: time.Now(), Op: "lease.expired", Caller: "helper", Args: args, OK: true})
	})
	srv := &Server{Ops: local, Authenticate: authenticator(cfg), Audit: audit}
	audit.Record(AuditEntry{Op: "start", OK: true, Caller: "helper " + privops.BuildVersion})
	return srv.Serve(ctx, l)
}

// failOpenAtStart removes state a previous helper run may have left (it
// crashed or the machine rebooted while a daemon owned it): no lease can
// own anything yet, so the kill switch and the hosts block go, the same
// fail-open rule leases use; so do the blocked destinations. A daemon that is still alive re-applies its
// kill switch when its tunnels reconnect.
func failOpenAtStart(ctx context.Context, local *privops.Local, audit *Audit) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for op, fn := range map[string]func() error{
		"startup.killswitch_off": func() error { return local.KillSwitchDisable(cctx) },
		"startup.hosts_clear":    func() error { return local.WriteHostsBlock(cctx, nil) },
		"startup.block_clear":    func() error { return local.SetBlockedDestinations(cctx, privops.BlockRequest{}) },
	} {
		e := AuditEntry{Time: time.Now(), Op: op, Caller: "helper", OK: true}
		if err := fn(); err != nil {
			e.OK, e.Error = false, err.Error()
		}
		audit.Record(e)
	}
}
