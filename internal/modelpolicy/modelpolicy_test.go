package modelpolicy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPathRespectsEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.json")
	t.Setenv("CODEBUDDY_PROXY_MODELPOLICY_PATH", path)
	if got := DefaultPath(); got != path {
		t.Fatalf("DefaultPath=%s want env override %s", got, path)
	}
}

func TestLoadOrCreateAndSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultFileName)
	p, err := Load(path)
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if p.Enabled {
		t.Fatal("empty policy should be disabled")
	}

	p.Enabled = true
	p.Deny = []string{"glm-5.3-flash"}
	if err := p.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, _ := os.ReadFile(path)
	if len(raw) == 0 {
		t.Fatal("file should not be empty")
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !loaded.Enabled || len(loaded.Deny) != 1 {
		t.Fatalf("unexpected loaded policy: %+v", loaded)
	}
	if loaded.Deny[0] != "glm-5.3-flash" {
		t.Fatalf("deny normalized wrong: %+v", loaded.Deny)
	}
}

func TestRejectReasonBehavior(t *testing.T) {
	cases := []struct {
		name   string
		pol    Policy
		id     string
		site   string
		expect bool // true=allowed
	}{
		{"disabled allows all", Policy{}, "deepseek-v4.1-flash", "", true},
		{"empty deny allows", Policy{Enabled: true}, "hy3", "", true},
		{"in deny list", Policy{Enabled: true, Deny: []string{"glm-5"}}, "glm-5", "", false},
		{"auto always allowed", Policy{Enabled: true, Deny: []string{"hy3"}}, "auto", "", true},
		{"denied", Policy{Enabled: true, Deny: []string{"glm"}}, "glm", "", false},
		{"domestic deny extends global", Policy{Enabled: true, Deny: []string{"global-model"}, Domestic: SiteRules{Deny: []string{"domestic-model"}}}, "domestic-model", "domestic", false},
		{"domestic deny isolated to domestic", Policy{Enabled: true, Deny: []string{"global-model"}, Domestic: SiteRules{Deny: []string{"domestic-model"}}}, "domestic-model", "global", true},
		{"global deny works", Policy{Enabled: true, Deny: []string{"hy3"}, Global: SiteRules{Deny: []string{"gpt-5"}}}, "gpt-5", "global", false},
		{"global deny isolated to global", Policy{Enabled: true, Deny: []string{"hy3"}, Global: SiteRules{Deny: []string{"gpt-5"}}}, "gpt-5", "domestic", true},
		{"cn alias works", Policy{Enabled: true, Deny: []string{"domestic-model"}, Domestic: SiteRules{Deny: []string{"domestic-model"}}}, "domestic-model", "cn", false},
		{"both sites denied same model", Policy{Enabled: true, Deny: []string{"common-bad"}, Domestic: SiteRules{Deny: []string{"domestic-bad"}}, Global: SiteRules{Deny: []string{"global-bad"}}}, "common-bad", "domestic", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.pol.Allowed(tc.id, tc.site); got != tc.expect {
				t.Fatalf("Allowed(%s, %s)=%v want %v (policy=%+v)", tc.id, tc.site, got, tc.expect, tc.pol)
			}
		})
	}
}

func TestFilterKeepsAuto(t *testing.T) {
	pol := Policy{
		Enabled: true,
		Deny:    []string{"glm"},
	}
	type fake struct{ ID string }
	fakes := []fake{{"auto"}, {"hy3"}, {"glm"}}

	allowed := 0
	for _, m := range fakes {
		if pol.Allowed(m.ID, "") {
			allowed++
		}
	}
	if allowed != 2 { // auto + hy3
		t.Fatalf("want 2 allowed, got %d", allowed)
	}
}

func TestManagerWriteUpdatesCached(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.json")
	m := NewWithPath(path)
	pol, _ := m.Read()
	if pol.Enabled {
		t.Fatal("fresh manager should have disabled policy")
	}
	pol.Enabled = true
	if err := m.Write(pol); err != nil {
		t.Fatalf("Write: %v", err)
	}
	again, err := m.Read()
	if err != nil || !again.Enabled {
		t.Fatalf("Read after Write: pol=%+v err=%v", again, err)
	}
}
