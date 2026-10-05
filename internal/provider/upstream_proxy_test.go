package provider

import (
	"net/http"
	"net/url"
	"sort"
	"testing"

	"github.com/wnddd839/codebuddy-proxy/internal/config"
)

func productRequest(t *testing.T, host string) *http.Request {
	t.Helper()
	u, err := url.Parse("https://" + host + "/v2/plugin/auth/state")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return &http.Request{URL: u}
}

func TestUpstreamHostsCoversProductEndpoints(t *testing.T) {
	got := append([]string(nil), config.UpstreamHosts()...)
	sort.Strings(got)
	want := []string{
		"copilot.tencent.com",
		"www.codebuddy.ai",
		"www.codebuddy.cn",
		"www.workbuddy.ai",
		"www.workbuddy.cn",
	}
	if len(got) != len(want) {
		t.Fatalf("hosts=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("hosts=%v want=%v", got, want)
		}
	}
}

// 产品自身主机必须绕开环境代理：国际站经环境代理会 TLS 握手超时。
func TestUpstreamProxyGoesDirectForProductHosts(t *testing.T) {
	proxy := upstreamProxyFunc(config.Config{})
	for _, host := range config.UpstreamHosts() {
		got, err := proxy(productRequest(t, host))
		if err != nil {
			t.Fatalf("%s: %v", host, err)
		}
		if got != nil {
			t.Fatalf("%s: expected direct, got %s", host, got)
		}
	}
}

// 显式代理优先于直连，保留「国际站需要代理才能出网」这类用户的出路。
func TestUpstreamProxyExplicitSettingWins(t *testing.T) {
	proxy := upstreamProxyFunc(config.Config{UpstreamProxy: "http://127.0.0.1:10999"})
	for _, host := range config.UpstreamHosts() {
		got, err := proxy(productRequest(t, host))
		if err != nil {
			t.Fatalf("%s: %v", host, err)
		}
		if got == nil || got.String() != "http://127.0.0.1:10999" {
			t.Fatalf("%s: expected explicit proxy, got %v", host, got)
		}
	}
}

// 非产品主机（如 GitHub）仍然跟随环境代理，第三方出站不受这次改动影响。
func TestUpstreamProxyDelegatesThirdPartyHosts(t *testing.T) {
	proxy := upstreamProxyFunc(config.Config{})
	req := productRequest(t, "api.github.com")
	got, err := proxy(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want, wantErr := http.ProxyFromEnvironment(req)
	if wantErr != nil {
		t.Fatalf("env proxy err: %v", wantErr)
	}
	if (got == nil) != (want == nil) || got != nil && got.String() != want.String() {
		t.Fatalf("third-party host should follow env proxy: got %v want %v", got, want)
	}
}
