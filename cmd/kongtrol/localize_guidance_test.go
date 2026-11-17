package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

func TestErrorGuidance(t *testing.T) {
	t.Setenv("KONGTROL_LANG", "en")
	local := fmt.Errorf("wireguard: disconnect: %w", &privops.DeniedError{Reason: privops.DenyWGNotManaged, Detail: "x"})
	remote := &daemonctl.RemoteError{Status: "500", Message: "raw", Key: "privops.denied.wg_not_managed"}
	for _, err := range []error{local, remote} {
		line, ok := errorGuidance(err)
		if !ok || line != ct("privops.denied.wg_not_managed") {
			t.Fatalf("%v → %q %v", err, line, ok)
		}
	}
	if _, ok := errorGuidance(errors.New("other")); ok {
		t.Fatal("plain error got guidance")
	}
}
