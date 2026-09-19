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

func TestNewDayResetKeepsOnlyToday(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")
	l, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer l.Close()
	l.Recordf("event", "old", 1)
	l.Recordf("event", "old", 2)

	// 模拟跨天：把内部日期键改到昨天，再写入应触发清空并只保留新记录。
	l.mu.Lock()
	l.day = todayKey(time.Now().Add(-24 * time.Hour))
	l.mu.Unlock()
	l.Recordf("event", "fresh", "x")

	entries, size, err := Tail(path, 10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want only today's 1 entry, got %d entries: %v", len(entries), entries)
	}
	if entries[0]["fields"].(map[string]any)["fresh"] != "x" {
		t.Fatalf("unexpected entries: %v", entries)
	}
	if size <= 0 {
		t.Fatalf("size=%d want >0", size)
	}
}

func TestTailClearsStalePreviousDayFile(t *testing.T) {
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

	entries, size, err := Tail(path, 10)
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if len(entries) != 0 || size != 0 {
		t.Fatalf("stale file should be cleared: entries=%d size=%d", len(entries), size)
	}
	raw, _ := os.ReadFile(path)
	if strings.TrimSpace(string(raw)) != "" {
		t.Fatalf("file should be truncated, got %q", raw)
	}
}

func TestNewClearsStaleFileOnStartup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "activity.log")
	old, _ := json.Marshal(Event{TS: time.Now().Add(-48 * time.Hour), Kind: "event", Fields: map[string]any{"old": "yesterday"}})
	if err := os.WriteFile(path, append(old, '\n'), 0o644); err != nil {
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
	if len(entries) != 1 {
		t.Fatalf("want 1 fresh entry, got %d: %v", len(entries), entries)
	}
	if entries[0]["fields"].(map[string]any)["today"] != "now" {
		t.Fatalf("unexpected entries: %v", entries)
	}
}

func TestContainsStaleDay(t *testing.T) {
	now := time.Now()
	before := now.Add(-48 * time.Hour)
	line := func(ts time.Time) string {
		b, _ := json.Marshal(Event{TS: ts, Kind: "k"})
		return string(b) + "\n"
	}
	today := todayKey(now)
	if containsStaleDay(strings.NewReader(line(now)), today) {
		t.Fatal("today entry should not be stale")
	}
	if !containsStaleDay(strings.NewReader(line(before)), today) {
		t.Fatal("yesterday entry should be stale")
	}
	if !containsStaleDay(strings.NewReader(line(before)+line(now)), today) {
		t.Fatal("mixed-day file should be stale")
	}
	if containsStaleDay(strings.NewReader("not json\n\n"), today) {
		t.Fatal("garbage lines should be ignored")
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
