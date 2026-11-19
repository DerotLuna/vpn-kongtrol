package policy

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// classifyTable is testdata/classify_targets.json, shared with the
// dashboard's "＋ Agregar" sheet (web/ui/src/components/add/model.test.ts):
// both must read the same entries the same way (ADR-0055).
type classifyTable struct {
	Cases []struct {
		Name       string   `json:"name"`
		In         []string `json:"in"`
		Subdomains bool     `json:"subdomains"`
		Domains    []string `json:"domains"`
		IPRanges   []string `json:"ip_ranges"`
		Apps       []string `json:"apps"`
		First      string   `json:"first"`
		Invalid    []string `json:"invalid"`
	} `json:"cases"`
	AutoName []struct {
		First string `json:"first"`
		Via   string `json:"via"`
		Want  string `json:"want"`
	} `json:"auto_name"`
}

func loadClassifyTable(t *testing.T) classifyTable {
	t.Helper()
	raw, err := os.ReadFile("testdata/classify_targets.json")
	if err != nil {
		t.Fatal(err)
	}
	var tab classifyTable
	if err := json.Unmarshal(raw, &tab); err != nil {
		t.Fatal(err)
	}
	if len(tab.Cases) == 0 || len(tab.AutoName) == 0 {
		t.Fatal("empty table")
	}
	return tab
}

// orNil makes an empty JSON list compare equal to a nil Go slice.
func orNil(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

func TestClassifyTargets(t *testing.T) {
	for _, c := range loadClassifyTable(t).Cases {
		t.Run(c.Name, func(t *testing.T) {
			got, invalid := ClassifyTargets(c.In, c.Subdomains)
			want := Targets{Domains: orNil(c.Domains), IPRanges: orNil(c.IPRanges), Apps: orNil(c.Apps), First: c.First}
			if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(invalid, orNil(c.Invalid)) {
				t.Fatalf("got %+v invalid %q, want %+v invalid %q", got, invalid, want, c.Invalid)
			}
		})
	}
}

func TestAutoName(t *testing.T) {
	for _, c := range loadClassifyTable(t).AutoName {
		if got := AutoName("%[1]s por %[2]s", c.First, c.Via, nil); got != c.Want {
			t.Errorf("AutoName(%q) = %q, want %q", c.First, got, c.Want)
		}
	}
	long := AutoName("%[1]s por %[2]s", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.com", "v", nil)
	if n := len([]rune(long)); n > 60 {
		t.Fatalf("long name has %d runes", n)
	}
	if got := AutoName("%[1]s via %[2]s", "a.com", "v", []string{"a.com via v"}); got != "a.com via v 2" {
		t.Fatal(got)
	}
}

func TestSuggestName(t *testing.T) {
	existing := []string{"github.com por oficina", "GITHUB.COM POR OFICINA 2"}
	if got := SuggestName("x por y", existing); got != "x por y" {
		t.Fatal(got)
	}
	if got := SuggestName("github.com por oficina", existing); got != "github.com por oficina 3" {
		t.Fatal(got)
	}
}
