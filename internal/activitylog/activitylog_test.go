package activitylog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordAppendsJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")

	l, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l.Close()

	l.Record("start", map[string]any{"addr": "127.0.0.1:32126"})
	l.Recordf("checkin", "accounts", 3, "ok", 2)
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d: %q", len(lines), lines)
	}
	var ev Event
	if err := json.Unmarshal([]byte(lines[0]), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Kind != "start" {
		t.Fatalf("kind=%q want start", ev.Kind)
	}
	if ev.Fields["addr"] != "127.0.0.1:32126" {
		t.Fatalf("fields=%v", ev.Fields)
	}
	if ev.TS.IsZero() {
		t.Fatalf("ts missing")
	}
}

func TestRotateWhenOverLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")

	l, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.WithMaxSize(64)
	defer l.Close()

	l.Recordf("event", "pad", strings.Repeat("x", 4096))
	l.Recordf("event", "second", "entry")
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile current: %v", err)
	}
	if !strings.Contains(string(raw), "second") {
		t.Fatalf("current log should hold the newest entry, got %q", raw)
	}
}

func TestRecordOddKVTrimsLast(t *testing.T) {
	dir := t.TempDir()
	l, err := New(filepath.Join(dir, "a.log"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l.Close()
	l.Recordf("odd", "k") // 奇数个参数：末尾丢弃
	l.Close()

	raw, _ := os.ReadFile(l.path)
	if !strings.Contains(string(raw), `"kind":"odd"`) {
		t.Fatalf("unexpected log: %q", raw)
	}
}

func TestTailReadsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")
	l, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for i := 0; i < 5; i++ {
		l.Recordf("event", "index", i)
	}
	l.Close()

	entries, size, err := Tail(path, 3)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d", len(entries))
	}
	for i, e := range entries {
		if e["kind"] != "event" {
			t.Fatalf("entry[%d] kind=%v", i, e["kind"])
		}
		if e["fields"].(map[string]any)["index"] != float64(4-i) {
			t.Fatalf("entry[%d] fields=%v want index=%d", i, e["fields"], 4-i)
		}
	}
	if size <= 0 {
		t.Fatalf("size=%d want >0", size)
	}

	entries, _, err = Tail(filepath.Join(dir, "missing.log"), 10)
	if err != nil {
		t.Fatalf("Tail missing: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("missing file should yield empty entries, got %d", len(entries))
	}
}

func TestNewDayTrimKeepsWithinRetention(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")
	l, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l.Close()
	l.Recordf("event", "old", 1)
	l.Recordf("event", "old", 2)

	// 模拟跨天：把内部日期键改到昨天，再写入应触发裁剪，但保留窗口内的历史记录不丢。
	l.mu.Lock()
	l.day = todayKey(time.Now().Add(-24 * time.Hour))
	l.mu.Unlock()
	l.Recordf("event", "fresh", "x")

	entries, size, err := Tail(path, 10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3 entries (2 kept history + 1 fresh), got %d: %v", len(entries), entries)
	}
	if entries[0]["fields"].(map[string]any)["fresh"] != "x" {
		t.Fatalf("newest entry should be the fresh one: %v", entries)
	}
	if size <= 0 {
		t.Fatalf("size=%d want >0", size)
	}
}

func TestNewDayTrimDropsPastCutoff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")

	line := func(daysAgo int, tag string) string {
		b, _ := json.Marshal(Event{TS: time.Now().Add(-24 * time.Hour * time.Duration(daysAgo)), Kind: "event", Fields: map[string]any{"tag": tag}})
		return string(b) + "\n"
	}
	seed := []byte(line(3, "drop") + line(0, "keep"))
	if err := os.WriteFile(path, seed, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	l, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l.Close()
	// 模拟跨天：下一次写入按 1 天窗口裁剪（New 时按默认 7 天，两条都还在）。
	l.WithMaxDays(1)
	l.mu.Lock()
	l.day = todayKey(time.Now().Add(-24 * time.Hour))
	l.mu.Unlock()
	l.Recordf("event", "fresh", "y")

	entries, _, err := Tail(path, 10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries (today kept + fresh), got %d: %v", len(entries), entries)
	}
	for _, e := range entries {
		if f, ok := e["fields"].(map[string]any); ok {
			if tag, _ := f["tag"].(string); tag == "drop" {
				t.Fatalf("entry past the 1-day cutoff must be dropped: %v", e)
			}
		}
	}
}

func TestMaxDaysZeroDisablesTrim(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")
	b, _ := json.Marshal(Event{TS: time.Now().Add(-72 * time.Hour), Kind: "event", Fields: map[string]any{"tag": "ancient"}})
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	l, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.WithMaxDays(0)
	defer l.Close()
	l.mu.Lock()
	l.trimLocked()
	l.mu.Unlock()

	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "ancient") {
		t.Fatalf("maxDays=0 must disable trimming, got %q", raw)
	}
}

func TestTailReadsRotatedBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")

	l, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.WithMaxSize(64)
	l.Recordf("event", "old", "rotated")
	l.Recordf("event", "new", "current")
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, size, err := Tail(path, 10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Tail must span the rotated backup, want 2 entries got %d: %v", len(entries), entries)
	}
	if entries[0]["fields"].(map[string]any)["new"] != "current" {
		t.Fatalf("newest should be the current entry: %v", entries)
	}
	if size <= 0 {
		t.Fatalf("size=%d want >0", size)
	}
}

func TestTailDoesNotMutateStalePreviousDayFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")
	l, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Recordf("event", "old", "yesterday")
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	stale := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(path, stale, stale); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	entries, size, err := Tail(path, 10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(entries) == 0 || size <= 0 {
		t.Fatalf("stale file should still be readable: entries=%d size=%d", len(entries), size)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatalf("Tail must be read-only, file mutated: before=%q after=%q", before, after)
	}
}

func TestNewKeepsRecentHistoryOnStartup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")
	// 昨天与两天前都在默认 7 天保留窗口内：重启不得清空。
	b, _ := json.Marshal(Event{TS: time.Now().Add(-48 * time.Hour), Kind: "event", Fields: map[string]any{"old": "yesterday"}})
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	l2, err := New(path)
	if err != nil {
		t.Fatalf("New stale: %v", err)
	}
	defer l2.Close()
	l2.Recordf("event", "today", "now")

	entries, _, err := Tail(path, 10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("yesterday's history must survive a restart, want 2 got %d: %v", len(entries), entries)
	}
	if entries[0]["fields"].(map[string]any)["today"] != "now" {
		t.Fatalf("newest entry should be the fresh one: %v", entries)
	}
}

func TestNewTrimsBeyondRetentionOnStartup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")
	line := func(daysAgo int, tag string) string {
		b, _ := json.Marshal(Event{TS: time.Now().Add(-24 * time.Hour * time.Duration(daysAgo)), Kind: "event", Fields: map[string]any{"tag": tag}})
		return string(b) + "\n"
	}
	// 30 天前的记录超出默认 7 天窗口，应被裁掉；轮转备份里的同样要清。
	if err := os.WriteFile(path, []byte(line(0, "keep")), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(path+".1", []byte(line(30, "stale-backup")+line(2, "backup-keep")), 0o644); err != nil {
		t.Fatalf("WriteFile backup: %v", err)
	}

	l2, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l2.Close()

	entries, _, err := Tail(path, 10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("want backup-keep + keep, got %d: %v", len(entries), entries)
	}
	for _, e := range entries {
		if f, ok := e["fields"].(map[string]any); ok {
			if tag, _ := f["tag"].(string); tag == "stale-backup" {
				t.Fatalf("rotated backup past the retention window must be pruned: %v", e)
			}
		}
	}
	if _, err := os.Stat(path + ".1"); !os.IsNotExist(err) {
		t.Fatalf("backup should be consumed by the trim, stat err=%v", err)
	}
}

func TestTrimDropsGarbageLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")
	b, _ := json.Marshal(Event{TS: time.Now(), Kind: "event", Fields: map[string]any{"tag": "good"}})
	seed := []byte("not json\n\n" + string(b) + "\n")
	if err := os.WriteFile(path, seed, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	l2, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l2.Close()

	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "not json") {
		t.Fatalf("unparseable lines should be dropped by the trim: %q", raw)
	}
	if !strings.Contains(string(raw), "good") {
		t.Fatalf("parseable entries must be kept: %q", raw)
	}
}

func TestNewKeepsTodayDataOnSameDayRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")
	l, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Recordf("event", "today", "keep")
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	l2, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l2.Close()
	l2.Recordf("event", "today", "append")

	entries, _, err := Tail(path, 10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("same-day restart should keep today's data, got %d: %v", len(entries), entries)
	}
}
