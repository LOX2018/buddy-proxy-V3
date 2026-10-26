package modelpolicy

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
		{"site empty allow falls back to global allow", Policy{Enabled: true, Allow: []string{"hy3"}, Domestic: SiteRules{}}, "hy3", "domestic", true},
		{"domestic empty does NOT inherit global allow", Policy{Enabled: true, Allow: []string{"hy3"}, Domestic: SiteRules{}, Global: SiteRules{Allow: []string{"gpt-5"}}}, "gpt-5", "domestic", false},
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

func TestTimeWindowContains(t *testing.T) {
	cases := []struct {
		name     string
		start    string
		end      string
		now      time.Time
		expected bool
	}{
		{"inside day window", "09:00", "18:00", time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local), true},
		{"before start", "09:00", "18:00", time.Date(2026, 1, 1, 8, 59, 0, 0, time.Local), false},
		{"at end excluded", "09:00", "18:00", time.Date(2026, 1, 1, 18, 0, 0, 0, time.Local), false},
		{"at start included", "09:00", "18:00", time.Date(2026, 1, 1, 9, 0, 0, 0, time.Local), true},
		{"overnight inside first half", "22:00", "06:00", time.Date(2026, 1, 1, 23, 30, 0, 0, time.Local), true},
		{"overnight inside second half", "22:00", "06:00", time.Date(2026, 1, 1, 3, 0, 0, 0, time.Local), true},
		{"overnight excluded midday", "22:00", "06:00", time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local), false},
		{"invalid start", "9:60", "18:00", time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local), false},
		{"equal start end", "10:00", "10:00", time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := TimeWindow{Start: tc.start, End: tc.end}
			if got := w.Contains(tc.now); got != tc.expected {
				t.Fatalf("Contains(%s,%s,%s)=%v want %v", tc.start, tc.end, tc.now.Format("15:04"), got, tc.expected)
			}
		})
	}
}

func TestTimeWindowMatchSite(t *testing.T) {
	cases := []struct {
		site     string
		request  string
		expected bool
	}{
		{"", "domestic", true},
		{"", "global", true},
		{"domestic", "domestic", true},
		{"domestic", "cn", false},
		{"global", "global", true},
		{"global", "domestic", false},
		{"Global", "GLOBAL", true},
	}
	for _, tc := range cases {
		w := TimeWindow{Site: tc.site}
		if got := w.MatchSite(tc.request); got != tc.expected {
			t.Fatalf("MatchSite(%q, %q)=%v want %v", tc.site, tc.request, got, tc.expected)
		}
	}
}

func TestParseHM(t *testing.T) {
	cases := []struct {
		in       string
		h, m     int
		expected bool
	}{
		{"09:30", 9, 30, true},
		{"23:59", 23, 59, true},
		{"00:00", 0, 0, true},
		{"9:30", 9, 30, true},
		{"24:00", 24, 0, false},
		{"12:60", 12, 60, false},
		{"12", 0, 0, false},
		{"12:ab", 0, 0, false},
		{"", 0, 0, false},
	}
	for _, tc := range cases {
		h, m, ok := parseHM(tc.in)
		if ok != tc.expected {
			t.Fatalf("parseHM(%q) ok=%v want %v", tc.in, ok, tc.expected)
		}
		if ok && (h != tc.h || m != tc.m) {
			t.Fatalf("parseHM(%q)=%d:%d want %d:%d", tc.in, h, m, tc.h, tc.m)
		}
	}
}

func TestPolicyScheduleOverridesSiteAllow(t *testing.T) {
	now := time.Date(2026, 1, 1, 23, 0, 0, 0, time.Local)
	pol := Policy{
		Enabled:  true,
		Allow:    []string{"gpt-5"},
		Domestic: SiteRules{Allow: []string{"hy3"}},
		Windows: []TimeWindow{{
			Name:  "night",
			Start: "22:00",
			End:   "08:00",
			Allow: []string{"hy4-preview"},
		}},
	}
	pol.now = func() time.Time { return now }

	if pol.Allowed("hy4-preview", "domestic") != true {
		t.Fatal("scheduled window should allow its own list even when site allow differs")
	}
	if pol.Allowed("hy3", "domestic") {
		t.Fatal("scheduled window should override site allow")
	}
	if pol.Allowed("gpt-5", "global") {
		t.Fatal("scheduled window (all-site) should override global allow")
	}
	if pol.RejectReason("hy3", "domestic") == "" {
		t.Fatal("expected reject reason for overridden model")
	}
}

func TestPolicyScheduleSiteScoped(t *testing.T) {
	now := time.Date(2026, 1, 1, 23, 0, 0, 0, time.Local)
	pol := Policy{
		Enabled: true,
		Allow:   []string{"gpt-5"},
		Windows: []TimeWindow{{
			Name:  "cn-night",
			Start: "22:00",
			End:   "08:00",
			Site:  "domestic",
			Allow: []string{"hy4-preview"},
		}},
	}
	pol.now = func() time.Time { return now }

	if !pol.Allowed("hy4-preview", "domestic") {
		t.Fatal("domestic-scoped window should allow its list")
	}
	if !pol.Allowed("gpt-5", "global") {
		t.Fatal("non-scoped site should still use global allow")
	}
}

func TestPolicyScheduleMissesFallsBack(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local)
	pol := Policy{
		Enabled: true,
		Allow:   []string{"gpt-5"},
		Windows: []TimeWindow{{
			Start: "22:00",
			End:   "08:00",
			Allow: []string{"hy4-preview"},
		}},
	}
	pol.now = func() time.Time { return now }
	if !pol.Allowed("gpt-5", "") {
		t.Fatal("outside schedule window should fall back to base allow")
	}
	if pol.Allowed("hy4-preview", "") {
		t.Fatal("outside schedule window should not grant scheduled list")
	}
}

func TestPolicyScheduleAutoStillAllowed(t *testing.T) {
	now := time.Date(2026, 1, 1, 23, 0, 0, 0, time.Local)
	pol := Policy{
		Enabled: true,
		Windows: []TimeWindow{{
			Start: "22:00",
			End:   "08:00",
			Allow: []string{"hy4-preview"},
		}},
	}
	pol.now = func() time.Time { return now }
	if !pol.Allowed("auto", "domestic") {
		t.Fatal("auto should always be allowed")
	}
}

func TestNormalizeDropsInvalidWindows(t *testing.T) {
	pol := Policy{
		Windows: []TimeWindow{
			{Start: "09:00", End: "10:00", Allow: []string{"hy3"}},
			{Start: "bad", End: "10:00", Allow: []string{"hy3"}},
			{Start: "10:00", End: "10:00", Allow: []string{"hy3"}},
			{Start: "11:00", End: "12:00"},
			{Start: "12:00", End: "13:00", Allow: []string{"hy4-preview"}},
		},
	}
	pol.normalize()
	if len(pol.Windows) != 2 {
		t.Fatalf("expected 2 windows after normalize, got %d: %+v", len(pol.Windows), pol.Windows)
	}
}

func TestPolicyActiveWindowAndTitle(t *testing.T) {
	pol := Policy{
		Windows: []TimeWindow{
			{Name: "night", Start: "22:00", End: "08:00", Allow: []string{"hy3"}},
		},
	}
	pol.normalize()
	if w := pol.ActiveWindow(time.Date(2026, 1, 1, 23, 0, 0, 0, time.Local)); w == nil || w.Name != "night" {
		t.Fatalf("expected active window night, got %+v", w)
	}
	if title := pol.ActiveWindowTitle(time.Date(2026, 1, 1, 23, 0, 0, 0, time.Local)); title != "night" {
		t.Fatalf("expected title 'night', got %q", title)
	}
	if title := pol.ActiveWindowTitle(time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local)); title != "" {
		t.Fatalf("expected empty title, got %q", title)
	}
}

func TestPolicyScheduleSurvivesSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultFileName)
	pol := Policy{
		Enabled: true,
		Allow:   []string{"gpt-5"},
		Windows: []TimeWindow{{Name: "night", Start: "22:00", End: "08:00", Site: "domestic", Allow: []string{"hy4-preview"}}},
	}
	if err := pol.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Windows) != 1 || loaded.Windows[0].Name != "night" || loaded.Windows[0].Site != "domestic" {
		t.Fatalf("windows lost through save/load: %+v", loaded.Windows)
	}
}

// 日夜双时段：同一 site 下 day/night 两条窗口，谁先定义谁优先命中。
func TestPolicyDayNightTwoWindows(t *testing.T) {
	base := Policy{
		Enabled:  true,
		Domestic: SiteRules{Allow: []string{"hy3"}},
		Global:   SiteRules{Allow: []string{"glm-5"}},
		Windows: []TimeWindow{
			{Start: "08:00", End: "20:00", Site: "domestic", Allow: []string{"hy3", "hy4-preview"}},
			{Start: "20:00", End: "08:00", Site: "domestic", Allow: []string{"hy4-preview"}},
		},
	}
	day := base
	day.now = func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local) }
	if !day.Allowed("hy3", "domestic") || !day.Allowed("hy4-preview", "domestic") {
		t.Fatal("day window should allow both hy3 and hy4-preview")
	}

	night := base
	night.now = func() time.Time { return time.Date(2026, 1, 1, 23, 0, 0, 0, time.Local) }
	if !night.Allowed("hy4-preview", "domestic") {
		t.Fatal("night window should allow hy4-preview")
	}
	if night.Allowed("hy3", "domestic") {
		t.Fatal("night window should exclude hy3 (site allow overridden)")
	}
	// 国际分区不受国内时段影响，仍走自己的 allow。
	if !night.Allowed("glm-5", "global") {
		t.Fatal("night window scoped to domestic must not touch global site")
	}
	if night.Allowed("hy3", "global") {
		t.Fatal("global site should still use its own allow")
	}
}

// 日夜双时段互不重叠时全天覆盖；不重叠的 day/night 之间无空档。
func TestPolicyDayNightNoGap(t *testing.T) {
	pol := Policy{
		Enabled: true,
		Windows: []TimeWindow{
			{Start: "08:00", End: "20:00", Allow: []string{"hy3"}},
			{Start: "20:00", End: "08:00", Allow: []string{"gpt-5"}},
		},
	}
	for _, tc := range []struct {
		hour    int
		want    string
		allowed bool
	}{
		{0, "gpt-5", true}, {7, "hy3", false}, {8, "hy3", true},
		{12, "hy3", true}, {19, "hy3", true}, {20, "hy3", false},
		{21, "gpt-5", true}, {23, "gpt-5", true},
	} {
		pol.now = func() time.Time {
			return time.Date(2026, 1, 1, tc.hour, 0, 0, 0, time.Local)
		}
		if got := pol.Allowed(tc.want, ""); got != tc.allowed {
			t.Fatalf("hour=%d allowed(%s)=%v want %v", tc.hour, tc.want, got, tc.allowed)
		}
	}
}

// 同 site 窗口重叠时取 allow 并集：两个时段各自在自己的范围内都生效。
func TestPolicyOverlappingWindowsUnion(t *testing.T) {
	pol := Policy{
		Enabled: true,
		Windows: []TimeWindow{
			{Name: "first", Start: "00:00", End: "23:59", Allow: []string{"hy3"}},
			{Name: "second", Start: "08:00", End: "18:00", Allow: []string{"gpt-5"}},
		},
	}
	pol.now = func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local) }
	if !pol.Allowed("hy3", "") {
		t.Fatal("first window's models should still be allowed")
	}
	if !pol.Allowed("gpt-5", "") {
		t.Fatal("overlapping window's models should also be allowed (union, not first-match)")
	}
	if pol.Allowed("glm-5", "") {
		t.Fatal("models absent from every matching window must be rejected")
	}
	if w := pol.ActiveWindow(pol.now()); w == nil || w.Name != "first" {
		t.Fatalf("ActiveWindow should report first, got %+v", w)
	}
}

// 同 site 完全重叠的两条时段（例如日/夜都填同一区间）取并集，
// Write 后再次 Read 仍走 mtime 判等缓存（相等不重载）；再写入一个更旧 mtime
// 的副本时，Equal 应触发重新加载——验证 "按 mtime 单调递增才跳过" 这一假设是否成立。
// 旧实现用 After(m.mod)，任何 mtime 回退都被当"未变更"，备份/还原会长期停在旧值。
func TestManagerReloadOnBackwardMtimeChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.json")
	// 初始写入 T0 版本。
	pol1 := Policy{Enabled: true}
	if err := pol1.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	m := NewWithPath(path)
	if cur, _ := m.Read(); !cur.Enabled {
		t.Fatal("first Read must see Enabled=true")
	}
	// 用 Chtimes 把磁盘 mtime 调到未来，再写入同内容——mtime 回退（相对缓存）触发重载，
	// 内容应被读回而不是卡在被 Write 时更新的旧缓存上。
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	future := fi.ModTime().Add(2 * time.Hour)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	if err := pol1.Save(path); err != nil {
		t.Fatalf("Save v1 again: %v", err)
	}
	cur, err := m.Read()
	if err != nil || !cur.Enabled {
		t.Fatalf("after backward mtime: got %+v err=%v", cur, err)
	}
	// 再写一个不同内容验证 reload 真的生效。
	pol2 := Policy{Enabled: false, Allow: []string{"x"}}
	if err := pol2.Save(path); err != nil {
		t.Fatalf("Save v2: %v", err)
	}
	cur2, err := m.Read()
	if err != nil || cur2.Enabled || len(cur2.Allow) != 1 || cur2.Allow[0] != "x" {
		t.Fatalf("after v2: got %+v err=%v", cur2, err)
	}
}
func TestPolicyDuplicateRangeUnion(t *testing.T) {
	pol := Policy{
		Enabled:  true,
		Domestic: SiteRules{Allow: []string{"hy3", "step-5-preview", "hy4-preview"}},
		Windows: []TimeWindow{
			{Start: "23:00", End: "07:59", Site: "domestic", Allow: []string{"hy3", "step-5-preview"}},
			{Start: "23:00", End: "07:59", Site: "domestic", Allow: []string{"hy4-preview", "hy3"}},
		},
	}
	pol.normalize()

	night := pol
	night.now = func() time.Time { return time.Date(2026, 10, 14, 0, 52, 0, 0, time.Local) }
	if !night.Allowed("hy4-preview", "domestic") {
		t.Fatal("hy4-preview must be effective inside the overlapping night range")
	}
	if !night.Allowed("step-5-preview", "domestic") {
		t.Fatal("other night-window models should also be allowed")
	}
	if night.Allowed("deepseek-v4.1-flash", "domestic") {
		t.Fatal("models outside every matching window must be rejected at night")
	}

	day := pol
	day.now = func() time.Time { return time.Date(2026, 10, 14, 12, 0, 0, 0, time.Local) }
	if !day.Allowed("hy4-preview", "domestic") {
		t.Fatal("outside all windows, site allow should fall back")
	}
}
