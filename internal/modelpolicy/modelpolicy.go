// Package modelpolicy 实现模型白名单（proxy-modelpolicy.json）。
//
// 策略文件位于 ~/.codebuddy/proxy-modelpolicy.json，结构：
//
//	{
//	  "enabled": true,
//	  "deny": ["glm-5.3-flash"],
//	  "domestic": {"deny": ["glm-5.2"]},
//	  "global":   {"deny": ["deepseek-v4.1-flash"]}
//	}
//
// 规则：enabled 关 → 不限制；deny 命中 → 拒绝（优先）。
// auto/default 始终放行。分区 deny 累加全局 deny。
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

// SiteRules 是单区域（domestic / global）的独立禁用规则。
type SiteRules struct {
	Deny []string `json:"deny,omitempty"`
}

// Policy 是模型白名单配置。
type Policy struct {
	Enabled  bool      `json:"enabled"`
	Deny     []string  `json:"deny,omitempty"`
	Domestic SiteRules `json:"domestic,omitempty"`
	Global   SiteRules `json:"global,omitempty"`
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
	if !fi.ModTime().After(m.mod) {
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
// site 为空时仅使用全局 deny；"domestic" 或 "global" 时叠加分区 deny。
func (p Policy) RejectReason(publicID, site string) string {
	id := models.PublicModelID(publicID)
	if !p.Enabled {
		return ""
	}
	lower := strings.ToLower(strings.TrimSpace(id))
	if id == "" || lower == "auto" || lower == "default" {
		return ""
	}
	denied := p.set(append(append([]string{}, p.Deny...), p.siteDeny(site)...))
	if _, isDenied := denied[lower]; isDenied {
		return "模型 " + id + " 已被禁用"
	}
	return ""
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
	denied := p.set(append(append([]string{}, p.Deny...), p.siteDeny(site)...))
	for _, m := range list {
		id := strings.ToLower(models.PublicModelID(m.ID))
		if _, bad := denied[id]; bad {
			dropped++
			continue
		}
		if id == "auto" || id == "default" {
			kept = append(kept, m)
			continue
		}
		kept = append(kept, m)
	}
	return kept, dropped
}

func (p *Policy) normalize() {
	p.Deny = cleanIDs(p.Deny)
	p.Domestic.Deny = cleanIDs(p.Domestic.Deny)
	p.Global.Deny = cleanIDs(p.Global.Deny)
}

// siteDeny 返回指定区域的额外 deny 列表。
func (p *Policy) siteDeny(site string) []string {
	switch strings.ToLower(strings.TrimSpace(site)) {
	case "domestic", "cn":
		return p.Domestic.Deny
	case "global", "intl":
		return p.Global.Deny
	default:
		return nil
	}
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
