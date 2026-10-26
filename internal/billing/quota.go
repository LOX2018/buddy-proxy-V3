package billing

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// QuotaState 是从上游 billing 接口提炼的配额快照，用于号池冷却与选号。
type QuotaState struct {
	Remaining  *float64 `json:"remaining"`
	Unlimited  bool     `json:"unlimited,omitzero"`
	ResetAt    int64    `json:"resetAt,omitzero"`
	CheckedAt  int64    `json:"checkedAt,omitzero"`
	NotifyCode int      `json:"notifyCode,omitzero"`
}

func (q QuotaState) Exhausted(now time.Time) bool {
	if q.Unlimited {
		return false
	}
	if q.Remaining != nil && *q.Remaining > 0 {
		return false
	}
	nowMs := now.UnixMilli()
	if q.ResetAt > 0 && q.ResetAt <= nowMs {
		return false
	}
	if q.Remaining != nil && *q.Remaining <= 0 {
		return true
	}
	return q.NotifyCode == 3
}

func (q QuotaState) CooldownDuration(now time.Time) time.Duration {
	if !q.Exhausted(now) || q.ResetAt <= now.UnixMilli() {
		return 0
	}
	return time.Duration(q.ResetAt-now.UnixMilli()) * time.Millisecond
}

func QuotaStateFromUsage(usage UsageResult, now time.Time) QuotaState {
	state := QuotaState{
		Remaining: cloneFloatPtr(usage.Credits.Remaining),
		Unlimited: usage.Credits.Unlimited,
		ResetAt:   creditsNextResetAt(usage.Credits, now),
		CheckedAt: now.UnixMilli(),
	}
	if usage.Notify != nil {
		state.NotifyCode = usage.Notify.DosageNotifyCode
	}
	if state.ResetAt == 0 && state.Exhausted(now) {
		// 上游未返回结束时间但已耗尽：保守等待 5 小时（海外滚动窗口常见值）。
		state.ResetAt = now.Add(5 * time.Hour).UnixMilli()
	}
	return state
}

func IsQuotaExhaustedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"quota", "credit", "额度", "用量", "exhaust", "insufficient",
		"不足", "耗尽", "dosage", "resource exhausted",
		"no remaining", "out of credits", "out of quota",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func creditsNextResetAt(credits Credits, now time.Time) int64 {
	nowMs := now.UnixMilli()
	best := int64(0)
	consider := func(raw string) {
		ms := ParseBillingTimeMillis(raw)
		if ms <= nowMs {
			return
		}
		if best == 0 || ms < best {
			best = ms
		}
	}
	consider(credits.CycleEndTime)
	for _, pkg := range credits.Packages {
		consider(pkg.CycleEndTime)
		consider(pkg.SlicePeriodEndTime)
	}
	return best
}

func parseBillingTimeMillis(raw string) int64 {
	return ParseBillingTimeMillis(raw)
}

const maxModelRateLimitCooldown = 48 * time.Hour

// resetAnchors 是各语言文案里"重置时刻"的引导词：英文 will reset at，国内站中文 将在。
var resetAnchors = []string{"will reset at", "将在"}

// resetAnchor 返回引导词在文案中的结束下标，找不到返回 -1。
// 国内站 6004 文案是"将在 2026-10-16 23:00:00 UTC+8 重置"，不含 "will reset at"，
// 旧逻辑直接回落固定 2 分钟冷却——限流窗口内每个账号被反复打，白消耗请求。
func resetAnchor(msg string) int {
	for _, a := range resetAnchors {
		if i := strings.Index(msg, a); i >= 0 {
			return i + len(a)
		}
	}
	return -1
}

// ModelRateLimitCooldown 从 6004 / rate-model 的"将在…重置 / will reset at"文案解析冷却时长。
// 解析失败或已过期时返回 ok=false，由调用方回落固定表。
func ModelRateLimitCooldown(err error, now time.Time) (time.Duration, bool) {
	if err == nil {
		return 0, false
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	has6004 := strings.Contains(lower, "6004")
	hasRateModel := strings.Contains(lower, "rate-model") || strings.Contains(lower, "rate model")
	if !has6004 && !hasRateModel {
		return 0, false
	}
	anchor := resetAnchor(msg)
	if anchor < 0 {
		return 0, false
	}
	tail := msg[anchor:]
	ms := ParseBillingTimeMillis(extractWillResetAt(tail))
	if ms <= 0 {
		// 旧逻辑解析失败：生产 6004 文案的时间戳后紧跟 ", alternatively…"或"，您也可以…"，
		// 逗号不在旧截断集里，整段尾巴进解析器必然失败 → 回落固定 2 分钟。改按时间戳
		// 形状提取，时间戳之后的文案不参与解析。
		ms = ParseBillingTimeMillis(extractWillResetAtByShape(tail))
	}
	if ms <= 0 {
		return 0, false
	}
	d := time.Duration(ms-now.UnixMilli()) * time.Millisecond
	if d <= 0 {
		return 0, false
	}
	if d > maxModelRateLimitCooldown {
		d = maxModelRateLimitCooldown
	}
	return d, true
}

// extractWillResetAt 截取重置引导词之后的时间戳（v0.4.9.5 原逻辑，截断集与后缀处理不变）。
func extractWillResetAt(tail string) string {
	raw := strings.TrimSpace(tail)
	if cut := strings.IndexAny(raw, "\n;)]["); cut >= 0 {
		raw = strings.TrimSpace(raw[:cut])
	}
	lowerRaw := strings.ToLower(raw)
	for _, suffix := range []string{" utc+8", " utc+08", " utc+08:00", " gmt+8", " gmt+08"} {
		if strings.HasSuffix(lowerRaw, suffix) {
			return strings.TrimSpace(raw[:len(raw)-len(suffix)])
		}
	}
	return raw
}

// reWillResetAt 提取时间戳本体：日期 + 时间 + 可选小数秒 + 可选时区
// （`UTC+8` / `GMT+8` 风格，或 RFC3339 的 `Z` / `±HH:MM`）。
var reWillResetAt = regexp.MustCompile(`(?i)\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:\s*(?:UTC|GMT)\s*[+-]\s*\d{1,2}(?::?\d{2})?|Z|[+-]\d{2}:?\d{2})?`)

// reChinaZoneSuffix 匹配 ` UTC+8` / ` GMT +08:00` 这类中国时区后缀（容忍空格，但必须是 `+`）。
// ParseBillingTimeMillis 不认这种写法，剥掉后按 Asia/Shanghai 解析；非 +8 偏移（含 `UTC-8`）
// 不剥——宁可解析失败回落固定冷却，也不按错误的时区换算。
var reChinaZoneSuffix = regexp.MustCompile(`(?i)\s*(?:GMT|UTC)\s*\+\s*0?8(?::?0{2})?$`)

// extractWillResetAtByShape 按时间戳形状提取，供旧逻辑解析失败时兜底：生产 6004 文案的
// 时间戳后紧跟 ", alternatively, you can switch to…"（逗号不在旧截断集里），旧逻辑会把
// 整段文案带进 ParseBillingTimeMillis → 失败 → 回落固定 2 分钟冷却。
// 英文与国内中文文案共用这一条：reWillResetAt 只吃到时间戳+时区就停，尾部中文
// （"重置，您也可以切换其他模型继续使用。"）不参与匹配。
func extractWillResetAtByShape(tail string) string {
	raw := strings.TrimSpace(reWillResetAt.FindString(tail))
	return strings.TrimSpace(reChinaZoneSuffix.ReplaceAllString(raw, ""))
}

// ParseBillingTimeMillis 解析上游时间戳。无时区按 Asia/Shanghai。
func ParseBillingTimeMillis(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "<nil>" {
		return 0
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UnixMilli()
		}
	}
	loc := billingLocalLocation()
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return t.UnixMilli()
		}
	}
	if n, ok := asNumber(raw); ok && n > 1_000_000_000_000 {
		return int64(n)
	}
	if n, ok := asNumber(raw); ok && n > 1_000_000_000 {
		return int64(n * 1000)
	}
	return 0
}

var billingLocalLocation = sync.OnceValue(func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Local
	}
	return loc
})

func cloneFloatPtr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	return new(*v)
}
