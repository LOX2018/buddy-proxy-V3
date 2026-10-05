package updatecheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		current, latest string
		greater         bool
		comparable      bool
	}{
		{"v0.5.5-0.20260924093436-04ae4552ec77", "v0.5.4", false, true},
		{"v0.5.4", "v0.5.4", false, true},
		{"v0.5.4", "v0.5.5", true, true},
		{"v4.9", "v0.5.4", false, true},
		{"v0.5.4", "v4.9", true, true},
		{"0.5.4", "0.5.4", false, true},
		{"v4.10", "v4.9", false, true},
		{"dev", "v0.5.4", false, false},
		{"", "", false, false},
		{"abc", "v0.5.4", false, false},
	}
	for _, c := range cases {
		greater, comparable := Compare(c.current, c.latest)
		if greater != c.greater || comparable != c.comparable {
			t.Errorf("Compare(%q,%q) = (%v,%v), want (%v,%v)", c.current, c.latest, greater, comparable, c.greater, c.comparable)
		}
	}
}

func TestNormalizeRepo(t *testing.T) {
	cases := map[string]string{
		"":                       DefaultRepo,
		"https://github.com/a/b": "a/b",
		"http://github.com/a/b/": "a/b",
		"  a/b  ":                "a/b",
		"A/B":                    "A/B",
	}
	for in, want := range cases {
		if got := normalizeRepo(in); got != want {
			t.Errorf("normalizeRepo(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUpdaterCachesAndRechecks(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://github.com/a/b/releases/tag/v9.9.9","published_at":"2026-01-01T00:00:00Z","body":"hello"}`))
	}))
	defer srv.Close()

	orig := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = orig })

	u := New(srv.Client())
	r1 := u.Check(context.Background(), "a/b", "v1.0.0")
	if !r1.UpdateAvailable {
		t.Fatalf("expected update available, got %+v", r1)
	}
	r2 := u.Check(context.Background(), "a/b", "v1.0.0")
	if !r2.UpdateAvailable {
		t.Fatalf("expected cached update, got %+v", r2)
	}
	if hits != 1 {
		t.Fatalf("expected 1 network hit (cached), got %d", hits)
	}

	// 强制过期后应重新请求。
	u.mu.Lock()
	u.checked = time.Now().Add(-cacheTTL - time.Minute)
	u.mu.Unlock()
	r3 := u.Check(context.Background(), "a/b", "v1.0.0")
	if !r3.UpdateAvailable {
		t.Fatalf("expected refetched update, got %+v", r3)
	}
	if hits != 2 {
		t.Fatalf("expected 2 network hits after expiry, got %d", hits)
	}
}

func TestUpdaterHandlesUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", 500)
	}))
	defer srv.Close()

	orig := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = orig })

	u := New(srv.Client())
	r := u.Check(context.Background(), "a/b", "v1.0.0")
	if r.UpdateAvailable {
		t.Fatalf("no update expected on error, got %+v", r)
	}
	if !strings.Contains(r.Error, "status 500") {
		t.Fatalf("expected error detail, got %+v", r.Error)
	}
}

func TestUpdaterUsesInjectedClientWithToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://github.com/a/b/releases/tag/v9.9.9"}`))
	}))
	defer srv.Close()

	orig := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = orig })
	t.Setenv("GITHUB_TOKEN", "  test-token  ")

	u := New(srv.Client())
	r := u.Check(context.Background(), "a/b", "v1.0.0")
	if r.Error != "" {
		t.Fatalf("unexpected error: %v", r.Error)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("expected bearer token on injected client, got %q", gotAuth)
	}
	if !r.UpdateAvailable {
		t.Fatalf("expected update available, got %+v", r)
	}
}

func TestFailedCheckRetriesAfterShortTTL(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			http.Error(w, "boom", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://github.com/a/b/releases/tag/v9.9.9"}`))
	}))
	defer srv.Close()

	orig := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = orig })

	u := New(srv.Client())
	first := u.Check(context.Background(), "a/b", "v1.0.0")
	if !strings.Contains(first.Error, "status 403") {
		t.Fatalf("expected 403 detail, got %+v", first)
	}
	// 失败结论在 errorCacheTTL 内不重复打 GitHub。
	if cached := u.Check(context.Background(), "a/b", "v1.0.0"); cached.Error == "" {
		t.Fatalf("expected cached error, got %+v", cached)
	}
	if hits != 1 {
		t.Fatalf("expected 1 network hit while error is fresh, got %d", hits)
	}

	u.mu.Lock()
	u.checked = time.Now().Add(-errorCacheTTL - time.Minute)
	u.mu.Unlock()
	third := u.Check(context.Background(), "a/b", "v1.0.0")
	if third.Error != "" {
		t.Fatalf("expected retry to succeed, got %+v", third)
	}
	if hits != 2 {
		t.Fatalf("expected 2 network hits after error TTL, got %d", hits)
	}
}

func TestCompareRealTags(t *testing.T) {
	// 用真实可见的历史 tag 验证不受前缀干扰。
	if got, _ := Compare("v0.5.5-0.20260924093436-04ae4552ec77", "v0.5.5"); got {
		t.Fatalf("latest equal to current should not be greater")
	}
	greater, comparable := Compare("v0.5.5", "v0.5.5")
	if greater || !comparable {
		t.Fatalf("got (%v,%v)", greater, comparable)
	}
}
