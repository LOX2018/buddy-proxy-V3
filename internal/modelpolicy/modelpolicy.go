// Package modelpolicy 实现模型白名单（proxy-modelpolicy.json）。
//
// 策略文件位于 ~/.codebuddy/proxy-modelpolicy.json，结构：
//
//	{
//	  "enabled": true,
//	  "allow": ["hy3", "hy4-preview"],
//	  "domestic": {"allow": ["hy3"]},
//	  "global":   {"allow": ["gpt-5"]}
//	}
//
// 规则：enabled 关 → 不限制；allow 非空且未命中 → 拒绝；
// allow 为空 → 全放行。auto/default 始终放行。
// 分区 allow 非空时覆盖全局 allow。
package modelpolicy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wnddd839/codebuddy-proxy/internal/atomicwrite"
	"github.com/wnddd839/codebuddy-proxy/internal/models"
)

// DefaultFileName 是策略文件名（默认目录为 ~/.codebuddy）。
const DefaultFileName = "proxy-modelpolicy.json"

// SiteRules 是单区域（domestic / global）的独立白名单规则。
type SiteRules struct {
	Allow []string `json:"allow,omitempty"`
}

// TimeWindow 是每日时段分区（本地时区 HH:MM）。当前时间落在 [Start, End)
// 内且 Allow 非空时，该 Allow 参与生效——同一区域所有命中的窗口取 allow 并集，
// 无窗口命中时回落到分区/全局 allow。Site 为空时对所有区域生效，
// "domestic" / "global" 时只作用于该区域。
type TimeWindow struct {
	Name  string   `json:"name,omitempty"`
	Start string   `json:"start"`
	End   string   `json:"end"`
	Site  string   `json:"site,omitempty"`
	Allow []string `json:"allow,omitempty"`
}

// Policy 是模型白名单配置。
type Policy struct {
	Enabled  bool         `json:"enabled"`
	Allow    []string     `json:"allow,omitempty"`
	Domestic SiteRules    `json:"domestic,omitempty"`
	Global   SiteRules    `json:"global,omitempty"`
	Windows  []TimeWindow `json:"windows,omitempty"`

	now func() time.Time // 测试注入；空则用 time.Now。
}

// DefaultPath 解析策略文件路径：优先环境变量 CODEBUDDY_PROXY_MODELPOLICY_PATH，
// 否则 ~/.codebuddy/proxy-modelpolicy.json。
func DefaultPath() string {
	if p := strings.TrimSpace(os.Getenv("CODEBUDDY_PROXY_MODELPOLICY_PATH")); p != "" {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".codebuddy", DefaultFileName)
	}
	return DefaultFileName
}

// Load 从 path 读取并解析策略；文件不存在返回零值策略而不报错。
func Load(path string) (Policy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Policy{}, nil
		}
		return Policy{}, err
	}
	var p Policy
	if err := json.Unmarshal(raw, &p); err != nil {
		return Policy{}, fmt.Errorf("modelpolicy: parse %s: %w", path, err)
	}
	p.normalize()
	return p, nil
}

// Save 原子写入策略到 path（临时文件 + 改名），目录不存在时自动创建。
func (p Policy) Save(path string) error {
	p.normalize()
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return atomicwrite.Write(path, raw, 0o600)
}

// Manager 提供带缓存（mtime 变更感知）的策略读写。
type Manager struct {
	mu   sync.Mutex
	path string

	pol Policy
	mod time.Time
	err error
}

// New 创建 Manager，懒加载（首次 Read 才落盘）。
func New() *Manager {
	return &Manager{path: DefaultPath()}
}

// NewWithPath 使用自定义路径创建 Manager（供测试）。
func NewWithPath(path string) *Manager {
	return &Manager{path: path}
}

// Path 返回策略文件路径。
func (m *Manager) Path() string {
	return m.path
}

// Read 返回当前策略（文件变更自动重载；解析失败返回上次成功值与错误）。
func (m *Manager) Read() (Policy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.reloadLocked(); err != nil && m.err == nil {
		// 新出现的读取错误保留，但业务侧以旧策略继续运行。
		m.err = err
	}
	return m.pol, m.err
}

// Enabled 返回策略是否开启且已加载成功。
func (m *Manager) Enabled() bool {
	pol, err := m.Read()
	return err == nil && pol.Enabled
}

// Write 保存策略并更新缓存。
func (m *Manager) Write(p Policy) error {
	if err := p.Save(m.path); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pol = p
	m.err = nil
	if fi, err := os.Stat(m.path); err == nil {
		m.mod = fi.ModTime()
	}
	return nil
}

func (m *Manager) reloadLocked() error {
	fi, statErr := os.Stat(m.path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			m.pol = Policy{}
			m.mod = time.Time{}
			return nil
		}
		return statErr
	}
	// 用 Equal 而非 After：外部把更旧 mtime 的备份覆盖回策略文件时，After 恒为 false，
	// 缓存会永远停在旧值。任何 mtime 变化（含回退）都视为需要重新加载。
	if fi.ModTime().Equal(m.mod) {
		return nil
	}
	pol, err := Load(m.path)
	if err != nil {
		return err
	}
	m.pol = pol
	m.mod = fi.ModTime()
	m.err = nil
	return nil
}

// RejectReason 返回拒绝原因；允许时返回空串。
// site 为空时仅使用全局 allow；"domestic" 或 "global" 时分区 allow 覆盖全局。
func (p Policy) RejectReason(publicID, site string) string {
	return p.RejectReasonAt(publicID, site, p.clock())
}

// RejectReasonAt 与 RejectReason 相同，但使用调用方给定的当前时间（用于
// 定时窗口判断和测试）。
func (p Policy) RejectReasonAt(publicID, site string, now time.Time) string {
	id := models.PublicModelID(publicID)
	if !p.Enabled {
		return ""
	}
	lower := strings.ToLower(strings.TrimSpace(id))
	if id == "" || lower == "auto" || lower == "default" {
		return ""
	}
	allow := p.effectiveAllowAt(site, now)
	if len(allow) == 0 {
		return ""
	}
	if _, ok := p.set(allow)[lower]; ok {
		return ""
	}
	if len(allow) == 1 {
		return "模型 " + id + " 不在白名单，仅允许 " + allow[0]
	}
	return "模型 " + id + " 不在白名单，仅允许 " + strings.Join(allow, "、")
}

// Allowed 报告公开模型 id 在给定区域是否允许。
func (p Policy) Allowed(publicID, site string) bool {
	return p.RejectReason(publicID, site) == ""
}

// Filter 按站点过滤模型目录，返回保留列表与丢弃数量。
func (p Policy) Filter(list []models.Model, site string) (kept []models.Model, dropped int) {
	if !p.Enabled {
		return list, 0
	}
	allow := p.set(p.effectiveAllowAt(site, p.clock()))
	for _, m := range list {
		id := strings.ToLower(models.PublicModelID(m.ID))
		if id == "auto" || id == "default" || len(allow) == 0 {
			kept = append(kept, m)
			continue
		}
		if _, ok := allow[id]; ok {
			kept = append(kept, m)
			continue
		}
		dropped++
	}
	return kept, dropped
}

// ActiveWindow 返回当前时刻命中的第一个定时窗口；未命中返回 nil。
// 仅用于展示：实际生效的是所有命中窗口的 allow 并集（见 effectiveAllowAt）。
func (p Policy) ActiveWindow(now time.Time) *TimeWindow {
	idx := p.activeWindowAt(now)
	if idx < 0 {
		return nil
	}
	w := p.Windows[idx]
	return &w
}

// ActiveWindowTitle 返回当前命中窗口的显示名（用于 UI 提示），未命中返回空串。
func (p Policy) ActiveWindowTitle(now time.Time) string {
	w := p.ActiveWindow(now)
	if w == nil {
		return ""
	}
	return w.DisplayName()
}

func (p *Policy) normalize() {
	p.Allow = cleanIDs(p.Allow)
	p.Domestic.Allow = cleanIDs(p.Domestic.Allow)
	p.Global.Allow = cleanIDs(p.Global.Allow)
	if p.Windows == nil {
		return
	}
	out := make([]TimeWindow, 0, len(p.Windows))
	for i := range p.Windows {
		w := p.Windows[i]
		w.Start = strings.TrimSpace(w.Start)
		w.End = strings.TrimSpace(w.End)
		w.Site = strings.TrimSpace(w.Site)
		w.Name = strings.TrimSpace(w.Name)
		w.Allow = cleanIDs(w.Allow)
		// 至少一个时间合法且 allow 非空才保留。
		if _, _, ok := parseHM(w.Start); !ok {
			continue
		}
		if _, _, ok := parseHM(w.End); !ok {
			continue
		}
		if w.Start == w.End {
			continue
		}
		if len(w.Allow) == 0 {
			continue
		}
		out = append(out, w)
	}
	p.Windows = out
}

// effectiveAllow 返回指定区域的生效 allow（当前时间）。
func (p *Policy) effectiveAllow(site string) []string {
	return p.effectiveAllowAt(site, p.clock())
}

// effectiveAllowAt 返回指定区域在时间 now 的生效 allow：
// 严格按时间匹配——所有命中 now 且作用于该区域的窗口 allow 取并集，
// 不按定义顺序优先取第一条。无窗口命中时回落到分区 allow，分区 allow 非空则使用，
// 否则回落到 global allow，再回落到全局 allow。
func (p *Policy) effectiveAllowAt(site string, now time.Time) []string {
	siteKey := strings.ToLower(strings.TrimSpace(site))
	merged := make([]string, 0, 8)
	for i := range p.Windows {
		w := p.Windows[i]
		if !w.Contains(now) {
			continue
		}
		ws := strings.ToLower(strings.TrimSpace(w.Site))
		if ws != "" && ws != siteKey {
			continue
		}
		merged = append(merged, w.Allow...)
	}
	if len(merged) > 0 {
		return cleanIDs(merged)
	}
	switch strings.ToLower(strings.TrimSpace(site)) {
	case "domestic", "cn":
		if len(p.Domestic.Allow) > 0 {
			return p.Domestic.Allow
		}
		// 国内分区 allow 为空时回落到全局 allow（而非 global 分区 allow，避免国际白名单覆盖国内）
		return p.Allow
	case "global", "intl":
		if len(p.Global.Allow) > 0 {
			return p.Global.Allow
		}
		// 国际分区 allow 为空时回落到全局 allow
		return p.Allow
	}
	// 其他站点（空站点）直接回落到全局 allow
	return p.Allow
}

func (p *Policy) activeWindowAt(now time.Time) int {
	for i := range p.Windows {
		w := p.Windows[i]
		if w.Contains(now) {
			return i
		}
	}
	return -1
}

func (p Policy) clock() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

// DisplayName 返回窗口的可读名称（优先 name，否则 start-end 或 site 后缀）。
func (w TimeWindow) DisplayName() string {
	if w.Name != "" {
		return w.Name
	}
	label := w.Start + "-" + w.End
	if site := w.MatchSiteName(); site != "" {
		label += " · " + site
	}
	return label
}

// Contains 报告 now（本地时区）是否落在 [Start, End) 时段内；跨午夜自动处理。
func (w TimeWindow) Contains(now time.Time) bool {
	sh, sm, ok := parseHM(w.Start)
	if !ok {
		return false
	}
	eh, em, ok := parseHM(w.End)
	if !ok {
		return false
	}
	start := sh*60 + sm
	end := eh*60 + em
	cur := now.Hour()*60 + now.Minute()
	if start == end {
		return false
	}
	if start < end {
		return cur >= start && cur < end
	}
	return cur >= start || cur < end
}

// MatchSite 报告窗口是否作用于给定区域。空 Site 表示对所有区域生效。
func (w TimeWindow) MatchSite(site string) bool {
	if strings.TrimSpace(w.Site) == "" {
		return true
	}
	return strings.ToLower(strings.TrimSpace(w.Site)) == strings.ToLower(strings.TrimSpace(site))
}

// MatchSiteName 返回 site 的展示名（domestic/global/空）。
func (w TimeWindow) MatchSiteName() string {
	return strings.TrimSpace(w.Site)
}

// parseHM 解析 "HH:MM" 为小时与分钟；格式错误或越界返回 ok=false。
func parseHM(s string) (h, m int, ok bool) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	hStr, mStr := parts[0], parts[1]
	if len(hStr) < 1 || len(mStr) < 1 {
		return 0, 0, false
	}
	for _, c := range hStr + mStr {
		if c < '0' || c > '9' {
			return 0, 0, false
		}
	}
	if _, err := fmt.Sscanf(hStr, "%d", &h); err != nil {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(mStr, "%d", &m); err != nil {
		return 0, 0, false
	}
	if h < 0 || h > 23 {
		return 0, 0, false
	}
	if m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

func (p Policy) set(list []string) map[string]struct{} {
	out := make(map[string]struct{}, len(list))
	for _, s := range list {
		out[strings.ToLower(s)] = struct{}{}
	}
	return out
}

func cleanIDs(list []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(list))
	for _, raw := range list {
		id := models.PublicModelID(strings.TrimSpace(raw))
		if id == "" {
			continue
		}
		lower := strings.ToLower(id)
		if _, dup := seen[lower]; dup {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, id)
	}
	return out
}
