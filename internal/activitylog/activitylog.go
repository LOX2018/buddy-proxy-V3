package activitylog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// MaxFileSizeBytes 默认大小上限：超过则轮转为 *.1。
	MaxFileSizeBytes = 4 << 20
	// DefaultDirName 默认日志目录名（相对主目录扩展后为 ~/.codebuddy/activity）。
	DefaultDirName = ".codebuddy"
)

// DefaultPath 返回默认活动日志路径 ~/.codebuddy/activity/activity.log。
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultDirName, "activity", "activity.log"), nil
}

// Logger 是追加式 JSONL 活动日志（一行一条事件），每日零点自动清空重建，只保留当天记录。
type Logger struct {
	mu   sync.Mutex
	path string
	max  int64
	file *os.File
	day  string
}

// Event 是写入日志的标准化事件结构。
type Event struct {
	TS     time.Time      `json:"ts"`
	Kind   string         `json:"kind"`
	Fields map[string]any `json:"fields,omitempty"`
}

// New 创建/打开日志文件（不存在则创建），按 Append 复用。
// 若日志文件含非当天事件（前一日遗留或多日混存），则清空重建：只保留当天日志。
func New(path string) (*Logger, error) {
	if path == "" {
		return nil, fmt.Errorf("activitylog: empty path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	l := &Logger{path: path, max: MaxFileSizeBytes, file: f, day: todayKey(time.Now())}
	if rf, err := os.Open(path); err == nil {
		stale := containsStaleDay(rf, l.day)
		_ = rf.Close()
		if stale {
			l.resetForTodayLocked()
		}
	}
	return l, nil
}

// containsStaleDay 报告文件中是否存在 ts 日期不等于 given 的事件（参数控管便于单测）。
func containsStaleDay(r io.Reader, given string) bool {
	raw, err := io.ReadAll(r)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if todayKey(ev.TS) != given {
			return true
		}
	}
	return false
}

// WithMaxSize 设置触发轮转的字节上限。
func (l *Logger) WithMaxSize(n int64) *Logger {
	if n > 0 {
		l.max = n
	}
	return l
}

// Close 关闭日志文件（幂等）。
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

// Record 写入一条活动事件。
func (l *Logger) Record(kind string, fields map[string]any) {
	line, err := json.Marshal(Event{TS: time.Now(), Kind: kind, Fields: fields})
	if err != nil {
		return
	}
	l.write(line)
}

// Recordf 以键值对方式写入事件（等价于 map[string]any 构造）。
func (l *Logger) Recordf(kind string, kv ...any) {
	if len(kv)%2 != 0 {
		kv = kv[:len(kv)-1]
	}
	fields := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		fields[key] = kv[i+1]
	}
	l.Record(kind, fields)
}

func (l *Logger) write(line []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maybeResetNewDay()
	l.maybeRotateLocked()
	if l.file == nil {
		return
	}
	line = append(line, '\n')
	_, _ = l.file.Write(line)
}

// todayKey 返回本地日期键（YYYY-MM-DD），用于判断是否跨天。
func todayKey(t time.Time) string {
	return t.Format("2006-01-02")
}

// maybeResetNewDay 在跨天后清空重建日志文件，仅保留当天记录。
func (l *Logger) maybeResetNewDay() {
	if l.day != todayKey(time.Now()) {
		l.resetForTodayLocked()
	}
}

// resetForTodayLocked 清空当前文件并移除轮转备份，确保只保留当天日志。
func (l *Logger) resetForTodayLocked() {
	l.day = todayKey(time.Now())
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
	_ = os.Remove(l.path + ".1")
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	l.file = f
}

func (l *Logger) maybeRotateLocked() {
	if l.max <= 0 || l.file == nil {
		return
	}
	fi, err := l.file.Stat()
	if err != nil {
		return
	}
	if fi.Size() < l.max {
		return
	}
	_ = l.file.Close()
	l.file = nil
	backup := l.path + ".1"
	_ = os.Remove(backup)
	_ = os.Rename(l.path, backup)
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	l.file = f
}

// Tail 返回文件末尾最近 n 条事件（从新到旧）与文件当前字节数。
// 文件不存在时返回空切片与 size=0，不视为错误。
// 若文件最后写入时间不是当天，则视为已过期：清空文件（并删除轮转备份），当天尚未产生记录。
func Tail(path string, n int) ([]map[string]any, int64, error) {
	if n <= 0 {
		n = 100
	}
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	if todayKey(fi.ModTime()) != todayKey(time.Now()) {
		_ = os.Remove(path + ".1")
		if err := os.Truncate(path, 0); err != nil {
			return nil, 0, err
		}
		return nil, 0, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fi.Size(), err
	}
	all := strings.Split(string(raw), "\n")
	lines := make([]string, 0, len(all))
	for _, line := range all {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	start := len(lines) - n
	if start < 0 {
		start = 0
	}
	entries := make([]map[string]any, 0, n)
	for i := start; i < len(lines); i++ {
		var obj map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &obj); err != nil {
			continue
		}
		entries = append(entries, obj)
	}
	// 从新到旧返回。
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, fi.Size(), nil
}
