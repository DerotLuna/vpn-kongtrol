package privops

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// OpTimeout is the per-request time budget the helper allows each op.
func OpTimeout(op string) time.Duration {
	switch op {
	case OpOpenVPNStart, OpWGInstall, OpWGRemove, OpOpenVPNStop:
		return 45 * time.Second
	case OpKillSwitchOn, OpKillSwitchOff, OpDNSSet, OpDNSRestore, OpBlockSet:
		return 30 * time.Second
	}
	return 15 * time.Second
}

// KnownOp reports whether op is part of the contract.
func KnownOp(op string) bool {
	switch op {
	case OpPing, OpRouteAdd, OpRouteDelete, OpRouteFlush, OpDNSSet, OpDNSRestore,
		OpKillSwitchOn, OpKillSwitchOff, OpWGInstall, OpWGRemove, OpWGAllowedIPs,
		OpOpenVPNStart, OpOpenVPNStop, OpOpenVPNStatus, OpHostsWrite, OpLeaseRenew, OpBlockSet:
		return true
	}
	return false
}

// Dispatch decodes req.Args strictly, runs the op on ops and builds the
// response. It never panics on hostile input.
func Dispatch(ctx context.Context, ops Ops, req Request) Response {
	if req.V != ProtocolVersion {
		return Response{V: ProtocolVersion, Code: errCodeProtocol, Error: "unsupported protocol version"}
	}
	if !KnownOp(req.Op) {
		return Response{V: ProtocolVersion, Code: errCodeInvalid, Error: "unknown operation"}
	}
	if req.Lease != "" {
		if err := ValidateLeaseID(req.Lease); err != nil {
			return Response{V: ProtocolVersion, Code: errCodeInvalid, Error: err.Error()}
		}
		// Scope the lease to the authenticated caller so one user can
		// never renew or inherit another's.
		ctx = withLease(ctx, CallerFrom(ctx).ID+"/"+req.Lease)
	}
	result, err := dispatch(ctx, ops, req)
	if err != nil {
		resp := Response{V: ProtocolVersion, Code: ErrorCode(err), Error: err.Error()}
		var de *DeniedError
		if errors.As(err, &de) {
			resp.Reason = de.Reason
		}
		return resp
	}
	resp := Response{V: ProtocolVersion, OK: true}
	if result != nil {
		b, merr := json.Marshal(result)
		if merr != nil {
			return Response{V: ProtocolVersion, Code: errCodeFailed, Error: merr.Error()}
		}
		resp.Result = b
	}
	return resp
}

func decodeArgs(raw json.RawMessage, v any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if err := DecodeStrict(raw, v); err != nil {
		return invalid("arguments: %v", err)
	}
	return nil
}

func dispatch(ctx context.Context, ops Ops, req Request) (any, error) {
	switch req.Op {
	case OpPing:
		if err := decodeArgs(req.Args, &struct{}{}); err != nil {
			return nil, err
		}
		return ops.Ping(ctx)
	case OpRouteAdd, OpRouteDelete:
		var r Route
		if err := decodeArgs(req.Args, &r); err != nil {
			return nil, err
		}
		if req.Op == OpRouteAdd {
			return nil, ops.AddRoute(ctx, r)
		}
		return nil, ops.DeleteRoute(ctx, r)
	case OpRouteFlush, OpDNSRestore:
		var a IfaceArgs
		if err := decodeArgs(req.Args, &a); err != nil {
			return nil, err
		}
		if req.Op == OpRouteFlush {
			return nil, ops.FlushRoutes(ctx, a.Interface)
		}
		return nil, ops.RestoreDNS(ctx, a.Interface)
	case OpDNSSet:
		var a DNSRequest
		if err := decodeArgs(req.Args, &a); err != nil {
			return nil, err
		}
		return nil, ops.SetDNS(ctx, a)
	case OpKillSwitchOn:
		var a KillSwitchRequest
		if err := decodeArgs(req.Args, &a); err != nil {
			return nil, err
		}
		return nil, ops.KillSwitchEnable(ctx, a)
	case OpKillSwitchOff:
		if err := decodeArgs(req.Args, &struct{}{}); err != nil {
			return nil, err
		}
		return nil, ops.KillSwitchDisable(ctx)
	case OpWGInstall:
		var a WGInstallRequest
		if err := decodeArgs(req.Args, &a); err != nil {
			return nil, err
		}
		return ops.WGInstallTunnel(ctx, a)
	case OpWGRemove:
		var a WGRemoveRequest
		if err := decodeArgs(req.Args, &a); err != nil {
			return nil, err
		}
		return nil, ops.WGRemoveTunnel(ctx, a)
	case OpWGAllowedIPs:
		var a WGAllowedIPsRequest
		if err := decodeArgs(req.Args, &a); err != nil {
			return nil, err
		}
		return nil, ops.WGSetAllowedIPs(ctx, a)
	case OpOpenVPNStart:
		var a OpenVPNStartRequest
		if err := decodeArgs(req.Args, &a); err != nil {
			return nil, err
		}
		return ops.OpenVPNStart(ctx, a)
	case OpOpenVPNStop, OpOpenVPNStatus:
		var a SessionArgs
		if err := decodeArgs(req.Args, &a); err != nil {
			return nil, err
		}
		if req.Op == OpOpenVPNStop {
			return nil, ops.OpenVPNStop(ctx, a.ID)
		}
		return ops.OpenVPNStatus(ctx, a.ID)
	case OpHostsWrite:
		var a HostsArgs
		if err := decodeArgs(req.Args, &a); err != nil {
			return nil, err
		}
		return nil, ops.WriteHostsBlock(ctx, a.Entries)
	case OpLeaseRenew:
		if err := decodeArgs(req.Args, &struct{}{}); err != nil {
			return nil, err
		}
		return nil, ops.RenewLease(ctx)
	case OpBlockSet:
		var a BlockRequest
		if err := decodeArgs(req.Args, &a); err != nil {
			return nil, err
		}
		return nil, ops.SetBlockedDestinations(ctx, a)
	}
	return nil, invalid("unknown operation")
}
