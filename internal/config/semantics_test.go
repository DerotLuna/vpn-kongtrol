package config

import (
	"strings"
	"testing"
)

func TestLoad_PolicyViaUnknownProfile_Fails(t *testing.T) {
	path := writeConfig(t, `
vpns:
  office:
    type: openvpn
    config: /tmp/office.ovpn
    auth:
      method: certificate
policies:
  - name: "office-policy"
    match:
      domains: ["example.com"]
    via: missing
`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "references unknown profile") {
		t.Fatalf("expected unknown profile validation error, got: %v", err)
	}
}

func TestLoad_PolicyWithoutMatchers_Fails(t *testing.T) {
	path := writeConfig(t, `
vpns:
  office:
    type: openvpn
    config: /tmp/office.ovpn
    auth:
      method: certificate
policies:
  - name: "empty-policy"
    match: {}
    via: office
`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "must define at least one matcher") {
		t.Fatalf("expected empty matcher validation error, got: %v", err)
	}
}

func TestLoad_DuplicatePolicyNames_Fails(t *testing.T) {
	path := writeConfig(t, `
vpns:
  office:
    type: openvpn
    config: /tmp/office.ovpn
    auth:
      method: certificate
policies:
  - name: "dup"
    match:
      domains: ["a.example.com"]
    via: office
  - name: "dup"
    match:
      domains: ["b.example.com"]
    via: office
`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate policy name") {
		t.Fatalf("expected duplicate policy validation error, got: %v", err)
	}
}

func TestLoad_GroupUnknownProfile_Fails(t *testing.T) {
	path := writeConfig(t, `
vpns:
  office:
    type: openvpn
    config: /tmp/office.ovpn
    auth:
      method: certificate
groups:
  team:
    profiles: ["office", "missing"]
`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "group \"team\" references unknown profile") {
		t.Fatalf("expected invalid group profile validation error, got: %v", err)
	}
}

func TestLoad_SchedulerUnknownProfile_Fails(t *testing.T) {
	path := writeConfig(t, `
vpns:
  office:
    type: openvpn
    config: /tmp/office.ovpn
    auth:
      method: certificate
monitor:
  scheduler:
    enabled: true
    rules:
      - name: "work-hours"
        profiles: ["missing"]
        weekdays: ["mon","tue"]
        start: "09:00"
        end: "18:00"
`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "scheduler rule") {
		t.Fatalf("expected scheduler validation error, got: %v", err)
	}
}

func TestLoad_SchedulerInvalidTime_Fails(t *testing.T) {
	path := writeConfig(t, `
vpns:
  office:
    type: openvpn
    config: /tmp/office.ovpn
    auth:
      method: certificate
monitor:
  scheduler:
    enabled: true
    rules:
      - name: "work-hours"
        profiles: ["office"]
        weekdays: ["mon","tue"]
        start: "9:00"
        end: "18:00"
`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "invalid start time") {
		t.Fatalf("expected scheduler invalid start time error, got: %v", err)
	}
}
