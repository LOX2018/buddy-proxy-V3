package activitylog

import (
	"encoding/json"
	"fmt"
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
	// DefaultMaxDays 默认保留天数：早于该窗口的记录在启动或跨天时被裁掉。
	DefaultMaxDays = 7
)

// DefaultPath 返回默认活动日志路径 ~/.codebuddy/activity/activity.log。
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultDirName, "activity", "activity.log"), nil
}

// Logger 是追加式 JSONL 活动日志（一行一条事件）。跨天或启动时按保留天数裁剪
// 历史，只删早于保留窗口的记录——不会每日整文件清空，历史可回看。
type Logger struct {
	mu      sync.Mutex
	path    string
	max     int64
	maxDays int
	file    *os.File
	day     string
}

// Event 是写入日志的标准化事件结构。
type Event struct {
	TS     time.Time      `json:"ts"`
	Kind   string         `json:"kind"`
	Fields map[string]any `json:"fields,omitempty"`
}

// New 创建/打开日志文件（不存在则创建），按 Append 复用。
// 打开后先裁剪一次：保留最近 maxDays 天的记录，早于窗口的行被丢弃。
func New(path string) (*Logger, error) {
	if path == "" {
		return nil, fmt.Errorf("activitylog: empty path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	l := &Logger{path: path, max: MaxFileSizeBytes, maxDays: DefaultMaxDays, file: f, day: todayKey(time.Now())}
	l.mu.Lock()
	l.trimLocked()
	l.mu.Unlock()
	return l, nil
}

// WithMaxSize 设置触发轮转的字节上限。
func (l *Logger) WithMaxSize(n int64) *Logger {
	if n > 0 {
		l.max = n
	}
	return l
}

// WithMaxDays 设置保留天数；<=0 表示不限天数（只靠字节轮转约束）。
func (l *Logger) WithMaxDays(n int) *Logger {
	l.maxDays = n
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

// Recordf 以键值对形式写入事件（等价于 map[string]any 构造）。
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
	l.maybeTrimLocked()
	l.rotateLocked()
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

// maybeTrimLocked 在跨天后按保留天数裁剪历史。同日写入是零开销判断。
func (l *Logger) maybeTrimLocked() {
	if l.day != todayKey(time.Now()) {
		l.day = todayKey(time.Now())
		l.trimLocked()
	}
}

// trimLocked 合并轮转备份 *.1 与主文件（*.1 恒早于主文件），丢弃早于保留窗口
// 的事件，重写主文件并清掉备份。maxDays<=0 时不裁剪。
// 读到的坏行一并丢弃，避免无法判定时间的数据长期留在盘上。
func (l *Logger) trimLocked() {
	if l.maxDays <= 0 {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour * time.Duration(l.maxDays))
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
	kept := make([]string, 0, 64)
	for _, p := range []string{l.path + ".1", l.path} {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var ev Event
			if json.Unmarshal([]byte(line), &ev) != nil {
				continue
			}
			if ev.TS.Before(cutoff) {
				continue
			}
			kept = append(kept, line)
		}
	}
	_ = os.Remove(l.path + ".1")
	out := []byte(strings.Join(kept, "\n"))
	if len(kept) > 0 {
		out = append(out, '\n')
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	l.file = f
	if len(out) > 0 {
		_, _ = f.Write(out)
	}
}

// rotateLocked 在超过字节上限时把主文件轮转为 *.1（先删旧备份再改名）。
func (l *Logger) rotateLocked() {
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
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	l.file = f
}

// Tail 返回末尾最近 n 条事件（从新到旧）与合并后的字节数。
// 轮转备份 *.1 与主文件都会被读到，保留窗口内的历史不会因轮转而消失。
// 文件都不存在时返回空切片与 size=0，不视为错误。
// Tail 只读：不裁剪、不删除轮转备份，避免 GET 视口产生副作用。
func Tail(path string, n int) ([]map[string]any, int64, error) {
	if n <= 0 {
		n = 100
	}
	var (
		size  int64
		lines []string
	)
	for _, p := range []string{path + ".1", path} {
		fi, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, size, err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil, size, err
		}
		size += fi.Size()
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.TrimSpace(line) != "" {
				lines = append(lines, line)
			}
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
	return entries, size, nil
}
