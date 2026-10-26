package modelpolicy

import (
	"testing"
	"time"
)

// 分区窗口只作用于自身区域：国内时段全覆盖时，国际/全局时段与分区兜底配置仍能生效。
func TestPolicyCrossSiteWindowNotMasked(t *testing.T) {
	pol := Policy{
		Enabled:  true,
		Domestic: SiteRules{Allow: []string{"hy3"}},
		Global:   SiteRules{Allow: []string{"glm-5"}},
		Windows: []TimeWindow{
			{Start: "08:00", End: "20:00", Site: "domestic", Allow: []string{"hy3", "hy4-preview"}},
			{Start: "20:00", End: "08:00", Site: "domestic", Allow: []string{"hy4-preview"}},
			{Name: "glb-day", Start: "08:00", End: "20:00", Site: "global", Allow: []string{"deepseek-v4.1-flash"}},
			{Name: "all-night", Start: "22:00", End: "08:00", Site: "", Allow: []string{"hy3"}},
		},
	}
	pol.normalize()

	at := func(hour int) Policy {
		p := pol
		p.now = func() time.Time { return time.Date(2026, 1, 1, hour, 0, 0, 0, time.Local) }
		return p
	}

	// 12:00 国际：命中国际·日时段，不再被国内·日遮蔽。
	if at(12).Allowed("glm-5", "global") {
		t.Fatal("site-specific global window must not be masked by domestic windows")
	}
	if !at(12).Allowed("deepseek-v4.1-flash", "global") {
		t.Fatal("global day window should take effect")
	}
	// 12:00 国内：仍走国内·日时段。
	if !at(12).Allowed("hy3", "domestic") {
		t.Fatal("domestic day window should still take effect")
	}
	// 03:00 国际：国内·夜与国际·日均不覆盖，命中全局（空 site）时段。
	if at(3).Allowed("deepseek-v4.1-flash", "global") {
		t.Fatal("all-region window must exclude non-listed models")
	}
	if !at(3).Allowed("hy3", "global") {
		t.Fatal("all-region window should apply to global site")
	}
	// 空 site 只受空 site 窗口约束，不被分区窗口遮蔽。
	if at(3).Allowed("deepseek-v4.1-flash", "") {
		t.Fatal("all-region window should apply to unscoped requests")
	}
}
