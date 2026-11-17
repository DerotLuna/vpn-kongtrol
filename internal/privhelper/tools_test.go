package privhelper

import (
	"encoding/binary"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestPlanToolsPrefersSystemAndHonoursRequires(t *testing.T) {
	specs := []toolSpec{
		{Name: "openvpn", System: []string{"/usr/sbin/openvpn"}, Sources: []string{"/brew/openvpn"}},
		{Name: "wg", System: []string{"/usr/bin/wg"}, Sources: []string{"/brew/a/wg", "/brew/b/wg"}},
		{Name: "wg-quick", System: []string{"/usr/bin/wg-quick"}, Sources: []string{"/brew/wg-quick"}},
		{Name: "bash", System: []string{"/usr/local/bin/bash"}, Sources: []string{"/brew/bash"}, Requires: "wg-quick"},
		{Name: "wireguard-go", Sources: []string{"/brew/wireguard-go"}, Requires: "wg-quick"},
	}
	sys := map[string]bool{"/usr/sbin/openvpn": true}
	files := map[string]bool{"/brew/openvpn": true, "/brew/b/wg": true, "/brew/wg-quick": true, "/brew/bash": true, "/brew/wireguard-go": true}
	trusted := func(ps ...string) bool {
		for _, p := range ps {
			if sys[p] {
				return true
			}
		}
		return false
	}
	got := planTools(specs, trusted, func(p string) bool { return files[p] })
	want := []toolPick{
		{Name: "openvpn", System: true},
		{Name: "wg", Source: "/brew/b/wg"},
		{Name: "wg-quick", Source: "/brew/wg-quick"},
		{Name: "bash", Source: "/brew/bash", Requires: "wg-quick"},
		{Name: "wireguard-go", Source: "/brew/wireguard-go", Requires: "wg-quick"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("plan:\n got %+v\nwant %+v", got, want)
	}

	// Without wg-quick, its dependencies are not copied.
	delete(files, "/brew/wg-quick")
	for _, p := range planTools(specs, trusted, func(p string) bool { return files[p] }) {
		if p.Name == "bash" || p.Name == "wireguard-go" || p.Name == "wg-quick" {
			t.Errorf("unexpected pick %+v", p)
		}
	}
}

func TestChangedTools(t *testing.T) {
	m := ToolManifest{Format: toolsFormat, Tools: map[string]ToolRecord{
		"wg":       {Source: "/brew/wg", SHA256: "aa", Size: 1},
		"wg-quick": {Source: "/brew/wg-quick", SHA256: "bb", Size: 2},
		"openvpn":  {Source: "/brew/openvpn", SHA256: "cc", Size: 3},
		"bash":     {Source: "/brew/bash", SHA256: "dd", Size: 4},
	}}
	hash := func(p string) (string, int64, error) {
		switch p {
		case "/brew/wg":
			return "aa", 1, nil // unchanged
		case "/brew/wg-quick":
			return "xx", 2, nil // upgraded
		case "/brew/bash":
			return "dd", 5, nil // size differs
		}
		return "", 0, errors.New("gone")
	}
	got := changedTools(m, hash)
	want := []ToolChange{{"bash", "/brew/bash"}, {"openvpn", "/brew/openvpn"}, {"wg-quick", "/brew/wg-quick"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestPrepareScript(t *testing.T) {
	body := "\necho hi\n"
	for in, want := range map[string]string{
		"#!/usr/bin/env bash":      "#!/usr/bin/env bash" + body,
		"#!/opt/homebrew/bin/bash": "#!/usr/bin/env bash" + body,
		"#!/bin/bash":              "#!/usr/bin/env bash" + body,
		"#!/bin/sh":                "#!/bin/sh" + body,
	} {
		got, err := prepareToolContent([]byte(in + body))
		if err != nil || string(got) != want {
			t.Errorf("%q: %q %v", in, got, err)
		}
	}
	for _, in := range []string{"#!/opt/homebrew/bin/python3", "#!/usr/bin/env -S bash -x", "#!", "#!/bin/bash -p"} {
		if _, err := prepareToolContent([]byte(in + body)); err == nil {
			t.Errorf("%q accepted", in)
		}
	}
	if _, err := prepareToolContent([]byte("MZ\x90\x00")); err == nil {
		t.Error("unknown format accepted")
	}
}

// machoWithDylib builds a minimal 64-bit little-endian Mach-O that loads lib.
func machoWithDylib(cmd uint32, lib string) []byte {
	name := append([]byte(lib), 0)
	size := 24 + len(name)
	size += (8 - size%8) % 8
	lc := make([]byte, size)
	binary.LittleEndian.PutUint32(lc[0:], cmd)
	binary.LittleEndian.PutUint32(lc[4:], uint32(size))
	binary.LittleEndian.PutUint32(lc[8:], 24) // name offset
	copy(lc[24:], name)
	hdr := make([]byte, 32)
	binary.LittleEndian.PutUint32(hdr[0:], 0xfeedfacf)
	binary.LittleEndian.PutUint32(hdr[4:], 0x0100000c) // arm64
	binary.LittleEndian.PutUint32(hdr[12:], 2)         // MH_EXECUTE
	binary.LittleEndian.PutUint32(hdr[16:], 1)         // ncmds
	binary.LittleEndian.PutUint32(hdr[20:], uint32(size))
	return append(hdr, lc...)
}

func TestCheckMachO(t *testing.T) {
	ok := [][]byte{
		machoWithDylib(lcLoadDylib, "/usr/lib/libSystem.B.dylib"),
		machoWithDylib(lcLoadDylib, "/System/Library/Frameworks/Security.framework/Versions/A/Security"),
	}
	for _, b := range ok {
		if _, err := prepareToolContent(b); err != nil {
			t.Errorf("system library refused: %v", err)
		}
	}
	bad := map[string][]byte{
		"homebrew openssl": machoWithDylib(lcLoadDylib, "/opt/homebrew/opt/openssl@3/lib/libssl.3.dylib"),
		"weak":             machoWithDylib(lcLoadWeakDylib, "/opt/homebrew/lib/liblzo2.2.dylib"),
		"rpath":            machoWithDylib(lcLoadDylib, "@rpath/libfoo.dylib"),
		"dotdot":           machoWithDylib(lcLoadDylib, "/usr/lib/../../opt/homebrew/lib/x.dylib"),
		"lc_rpath":         machoWithDylib(lcRpath, "/opt/homebrew/lib"),
	}
	for name, b := range bad {
		if _, err := prepareToolContent(b); err == nil || !strings.Contains(err.Error(), "outside the system") {
			t.Errorf("%s accepted: %v", name, err)
		}
	}
}

func TestSystemLib(t *testing.T) {
	if !systemLib("/lib64/ld-linux-x86-64.so.2", elfLibDirs...) {
		t.Error("system loader refused")
	}
	for _, p := range []string{"/home/linuxbrew/.linuxbrew/lib/ld.so", "/lib/../home/x", "lib/x", ""} {
		if systemLib(p, elfLibDirs...) {
			t.Errorf("%q accepted", p)
		}
	}
}
