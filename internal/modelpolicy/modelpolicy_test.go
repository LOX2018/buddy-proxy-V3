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
	p.Allow = []string{"hy3", "codebuddy/hy4-preview", "", "auto"}
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
	if !loaded.Enabled || len(loaded.Allow) != 3 {
		t.Fatalf("unexpected loaded policy: %+v", loaded)
	}
	if loaded.Allow[0] != "hy3" || loaded.Allow[1] != "hy4-preview" || loaded.Allow[2] != "auto" {
		t.Fatalf("allow normalized wrong: %+v", loaded.Allow)
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
		{"empty allow allows all", Policy{Enabled: true}, "hy3", "", true},
		{"in allow list", Policy{Enabled: true, Allow: []string{"hy3", "hy4-preview"}}, "hy4-preview", "", true},
		{"not in allow list", Policy{Enabled: true, Allow: []string{"hy3"}}, "glm-5", "", false},
		{"auto always allowed", Policy{Enabled: true, Allow: []string{"hy3"}}, "auto", "", true},
		{"default always allowed", Policy{Enabled: true, Allow: []string{"hy3"}}, "default", "", true},
		{"domestic allow overrides global", Policy{Enabled: true, Allow: []string{"gpt-5"}, Domestic: SiteRules{Allow: []string{"hy3"}}}, "hy3", "domestic", true},
		{"domestic allow excludes global-only", Policy{Enabled: true, Allow: []string{"gpt-5"}, Domestic: SiteRules{Allow: []string{"hy3"}}}, "gpt-5", "domestic", false},
		{"site-specific allow does not restrict other site", Policy{Enabled: true, Domestic: SiteRules{Allow: []string{"hy3"}}}, "hy3", "global", true},
		{"global allow overrides global fallback", Policy{Enabled: true, Allow: []string{"hy3"}, Global: SiteRules{Allow: []string{"gpt-5"}}}, "gpt-5", "global", true},
		{"global allow excludes domestic-only", Policy{Enabled: true, Allow: []string{"hy3"}, Global: SiteRules{Allow: []string{"gpt-5"}}}, "hy3", "global", false},
		{"site empty allow falls back to global", Policy{Enabled: true, Allow: []string{"hy3"}, Domestic: SiteRules{}}, "hy3", "domestic", true},
		{"cn alias uses domestic", Policy{Enabled: true, Domestic: SiteRules{Allow: []string{"hy3"}}}, "hy3", "cn", true},
		{"intl alias uses global", Policy{Enabled: true, Global: SiteRules{Allow: []string{"gpt-5"}}}, "gpt-5", "intl", true},
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
		Allow:   []string{"hy3"},
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
