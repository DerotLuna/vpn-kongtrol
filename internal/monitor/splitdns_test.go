package monitor

import (
	"strings"
	"testing"
)

func TestSplitDNS_RemoveAndRenderBlock(t *testing.T) {
	original := "127.0.0.1 localhost\n" +
		splitDNSStartMarker + "\n" +
		"1.1.1.1 api.example.com\n" +
		splitDNSEndMarker + "\n" +
		"::1 localhost\n"

	clean := removeSplitDNSBlock(original)
	if clean == "" {
		t.Fatal("expected non-empty hosts content after removing split-dns block")
	}
	if clean == original {
		t.Fatal("expected split-dns block to be removed")
	}

	block := renderSplitDNSBlock(map[string]string{"api.example.com": "1.1.1.1"})
	if block == "" {
		t.Fatal("expected rendered split-dns block")
	}
	if !strings.Contains(block, splitDNSStartMarker) || !strings.Contains(block, splitDNSEndMarker) {
		t.Fatal("rendered block missing markers")
	}
}
