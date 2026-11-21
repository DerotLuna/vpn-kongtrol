//go:build !linux

package privops

import "context"

func CheckDefaultEgress(_, _, _ string) DefaultEgressCheck {
	return DefaultEgressCheck{Reason: "unsupported_platform"}
}

func checkNativeDefaultRules(context.Context, string) error {
	return ErrUnsupported
}
