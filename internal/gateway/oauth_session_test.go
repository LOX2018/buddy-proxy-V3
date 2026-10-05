package gateway

import (
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wnddd839/codebuddy-proxy/internal/config"
)

func TestOAuthLaunchAuthorizedUsesLockedSnapshot(t *testing.T) {
	cfg := config.Config{OAuthSessionTTL: config.DefaultOAuthSessionTTL, Site: "domestic"}
	svc := New(cfg, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))

	svc.mu.Lock()
	svc.resetOAuthSessionLocked("domestic", "test", "http://127.0.0.1:32126")
	svc.oauth.Status = "waiting"
	svc.oauth.URL = "https://www.codebuddy.cn/login?platform=CLI&state=abc"
	svc.oauth.AuthState = "abc"
	id, token := svc.oauth.ID, svc.oauth.Token
	svc.mu.Unlock()

	if !svc.OAuthLaunchAuthorized(id, token) {
		t.Fatal("expected freshly generated launch credentials to authorize")
	}
	if svc.OAuthLaunchAuthorized(id, "wrong-token") {
		t.Fatal("expected mismatched token to be rejected")
	}

	snap := svc.CurrentOAuth()
	if snap.ID != id || snap.Token != token || snap.Status != "waiting" || snap.URL == "" {
		t.Fatalf("snapshot mismatch: %+v", snap)
	}
	live := svc.LiveOAuthSession()
	if live.ID != snap.ID || live.Token != snap.Token {
		t.Fatalf("LiveOAuthSession snapshot mismatch: %+v", live)
	}
}

func TestRaceOAuthResetAndLaunchAuthorize(t *testing.T) {
	cfg := config.Config{OAuthSessionTTL: config.DefaultOAuthSessionTTL, Site: "domestic"}
	svc := New(cfg, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))

	svc.mu.Lock()
	svc.resetOAuthSessionLocked("domestic", "test", "http://127.0.0.1:32126")
	svc.oauth.Status = "waiting"
	id, token := svc.oauth.ID, svc.oauth.Token
	svc.mu.Unlock()

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					svc.mu.Lock()
					svc.resetOAuthSessionLocked("domestic", "race", "http://127.0.0.1:32126")
					svc.oauth.Status = "waiting"
					svc.mu.Unlock()
				}
			}
		})
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					_ = svc.OAuthLaunchAuthorized(id, token)
					_ = svc.CurrentOAuth()
					_ = svc.LiveOAuthSession()
				}
			}
		})
	}

	time.Sleep(250 * time.Millisecond)
	close(stop)
	wg.Wait()
}

// 国际站授权改走无痕窗口后，管理台文案必须区分"已打开 / 打开失败 / 不适用"，
// 否则用户不知道去哪儿完成登录。
func TestOAuthWaitingMessageTracksPrivateWindow(t *testing.T) {
	cases := []struct {
		name    string
		session OAuthSession
		want    string
	}{
		{"opened", OAuthSession{Status: "waiting", OpenBrowser: "edge"}, "已在 edge 无痕窗口打开登录页"},
		{"failed", OAuthSession{Status: "waiting", OpenError: "no Edge or Chrome installation found"}, "无痕窗口打开失败"},
		{"legacy", OAuthSession{Status: "waiting"}, "请在打开的 CodeBuddy 页面完成登录"},
	}
	for _, tc := range cases {
		got := oauthMessage(tc.session)
		if !strings.Contains(got, tc.want) {
			t.Fatalf("%s: oauthMessage = %q, want it to contain %q", tc.name, got, tc.want)
		}
	}
}
