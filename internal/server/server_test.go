package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/wnddd839/codebuddy-proxy/internal/accounts"
	"github.com/wnddd839/codebuddy-proxy/internal/admin"
	"github.com/wnddd839/codebuddy-proxy/internal/config"
	"github.com/wnddd839/codebuddy-proxy/internal/gateway"
	"github.com/wnddd839/codebuddy-proxy/internal/modelpolicy"
)

func testServer(t *testing.T, requireAPIKey bool, adminPassword, apiKey string) *Server {
	t.Helper()
	return testServerCfg(t, config.Config{
		Host:          "127.0.0.1",
		Port:          32126,
		RequireAPIKey: requireAPIKey,
		APIKey:        apiKey,
		AdminPassword: adminPassword,
		AccountsPath:  filepath.Join(t.TempDir(), "accounts.json"),
		Site:          "domestic",
		Transport:     config.DefaultTransport,
	})
}

func testServerCfg(t *testing.T, cfg config.Config) *Server {
	t.Helper()
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 32126
	}
	if cfg.AccountsPath == "" {
		cfg.AccountsPath = filepath.Join(t.TempDir(), "accounts.json")
	}
	if cfg.Transport == "" {
		cfg.Transport = config.DefaultTransport
	}
	svc := gateway.New(cfg, slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	svc.ModelPolicy = modelpolicy.NewWithPath(filepath.Join(t.TempDir(), "policy.json"))
	t.Setenv("CODEBUDDY_PROXY_ACTIVITY_PATH", filepath.Join(t.TempDir(), "activity.log"))
	svc.Provider.HTTP = &http.Client{Transport: stubProbeTransport{status: http.StatusUnauthorized, body: "Authorization Required"}}
	t.Cleanup(func() { _ = svc.Close() })
	srv := New(cfg, svc)
	t.Cleanup(func() { srv.closeActivity() })
	return srv
}

func TestResponsesEmptyBodyReturnsResponsesError(t *testing.T) {
	srv := testServer(t, true, "", "secret-key")
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/v1/responses", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer secret-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	// Responses API 已实现：空 body 返回 Responses 形态错误 {error:{code,message}}。
	if !strings.Contains(rec.Body.String(), `"error"`) || !strings.Contains(rec.Body.String(), `"code"`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestResponsesBackgroundRejected(t *testing.T) {
	srv := testServer(t, true, "", "secret-key")
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/v1/responses", strings.NewReader(`{"model":"auto","input":"hi","background":true}`))
	req.Header.Set("Authorization", "Bearer secret-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "unsupported_parameter") || !strings.Contains(rec.Body.String(), "background") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestResolveRequestSite(t *testing.T) {
	cases := []struct {
		model, header, key, want string
	}{
		// M2 修复后语义：显式模型前缀 cn:/global: > 绑定 Key 场所 > X-Site 头。
		{"global", "cn", "domestic", "global"},
		{"global", "cn", "global", "global"},
		{"", "cn", "global", "global"},
		{"", "", "cn", "domestic"},
		{"", "", "global", "global"},
		{"", "", "", ""},
		// 未绑定（keySite 为空）时保留原 model > header 的择站逻辑。
		{"global", "cn", "", "global"},
		{"", "cn", "", "domestic"},
	}
	for _, tc := range cases {
		got := resolveRequestSite(tc.model, tc.header, tc.key)
		if got != tc.want {
			t.Fatalf("resolveRequestSite(%q,%q,%q)=%q want %q", tc.model, tc.header, tc.key, got, tc.want)
		}
	}
}

func TestAuthorizeAPIKeyRequired(t *testing.T) {
	srv := testServer(t, true, "", "secret-key")
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/v1/models", nil)
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/v1/models", nil)
	req.Header.Set("Authorization", "Bearer secret-key")
	rec = httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("with key status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuthorizeMappedAPIKeys(t *testing.T) {
	srv := testServerCfg(t, config.Config{
		RequireAPIKey: true,
		APIKey:        "cbp_primary",
		APIKeys:       config.ParseAPIKeys("cbp_cn:domestic,cbp_global:global"),
		Site:          "domestic",
	})

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/v1/models", nil)
	req.Header.Set("Authorization", "Bearer cbp_cn")
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mapped key status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/v1/models", nil)
	req.Header.Set("Authorization", "Bearer cbp_unknown")
	rec = httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unknown key status=%d", rec.Code)
	}
}

func TestModelsCatalogFollowsXSite(t *testing.T) {
	srv := testServer(t, false, "", "")
	seedBothSites(t, srv)
	srv.Svc.Provider.HTTP = &http.Client{Transport: catalogByHostTransport{}}

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/v1/models", nil)
	req.Header.Set("X-Site", "global")
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	ids := modelIDsFromList(t, rec.Body.Bytes())
	if !containsID(ids, "gpt-5") || containsID(ids, "glm-5.3-flash") {
		t.Fatalf("X-Site=global catalog=%v", ids)
	}
	for _, id := range ids {
		if strings.Contains(id, ":") {
			t.Fatalf("aliased id %q in %v", id, ids)
		}
	}
}

func TestModelsCatalogFollowsAPIKeySite(t *testing.T) {
	srv := testServerCfg(t, config.Config{
		RequireAPIKey: true,
		APIKey:        "cbp_primary",
		APIKeys:       config.ParseAPIKeys("cbp_cn:domestic,cbp_global:global"),
		Site:          "global",
		Product:       "codebuddy",
	})
	seedBothSites(t, srv)
	srv.Svc.Provider.HTTP = &http.Client{Transport: catalogByHostTransport{}}

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/v1/models", nil)
	req.Header.Set("Authorization", "Bearer cbp_cn")
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	ids := modelIDsFromList(t, rec.Body.Bytes())
	if !containsID(ids, "glm-5.3-flash") || containsID(ids, "gpt-5") {
		t.Fatalf("domestic key catalog=%v", ids)
	}
}

func TestChatFollowsAPIKeySiteUnlessModelPrefix(t *testing.T) {
	srv := testServerCfg(t, config.Config{
		RequireAPIKey: true,
		APIKey:        "cbp_primary",
		APIKeys:       config.ParseAPIKeys("cbp_cn:domestic,cbp_global:global"),
		Site:          "domestic",
		Product:       "codebuddy",
	})
	seedBothSites(t, srv)
	transport := &recordingUpstreamTransport{}
	srv.Svc.Provider.HTTP = &http.Client{Transport: transport}

	post := func(key, model string) {
		t.Helper()
		body := `{"model":"` + model + `","messages":[{"role":"user","content":"hi"}]}`
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.HTTP.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("key=%s model=%s status=%d body=%s", key, model, rec.Code, rec.Body.String())
		}
	}

	post("cbp_global", "auto")
	post("cbp_global", "cn:auto")
	got := transport.chatTokens()
	if len(got) < 2 {
		t.Fatalf("chat tokens=%v", got)
	}
	if got[0] != "token-global" {
		t.Fatalf("unprefixed model with global key used %q want token-global", got[0])
	}
	if got[1] != "token-domestic" {
		t.Fatalf("cn: prefix must override key site, used %q want token-domestic", got[1])
	}
}

func TestModelInfoAliasRoute(t *testing.T) {
	srv := testServer(t, true, "", "secret-key")
	for _, path := range []string{"/v1/model/info", "/model/info"} {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126"+path, nil)
		req.Header.Set("Authorization", "Bearer secret-key")
		rec := httptest.NewRecorder()
		srv.HTTP.Handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("path %s returned 404", path)
		}
		if rec.Code != http.StatusOK && rec.Code != http.StatusBadGateway {
			t.Fatalf("path %s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestAdminOpenWithoutPassword(t *testing.T) {
	srv := testServer(t, true, "", "secret-key")
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/direct-admin/", nil)
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin status=%d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "CodeBuddy") {
		t.Fatalf("unexpected admin body")
	}
}

func TestAdminChatTestRoutes(t *testing.T) {
	srv := testServer(t, false, "", "")
	acc, _, err := srv.Svc.Pool.Upsert(accounts.CreateAccount(accounts.Account{
		Label: "probe", Site: "domestic", BearerToken: "token-probe", Enabled: true,
		AuthStatus: accounts.AuthStatus{UserID: "u-probe"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	srv.Svc.Provider.HTTP = &http.Client{Transport: roundTripOKChat{}}

	t.Run("batch empty-ish ok with model", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/direct-admin/api/codebuddy/test", strings.NewReader(`{"model":"auto"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:32126")
		req.Host = "127.0.0.1:32126"
		rec := httptest.NewRecorder()
		srv.HTTP.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			OK       bool   `json:"ok"`
			PoolSite string `json:"poolSite"`
			Model    string `json:"model"`
			Summary  struct {
				Total  int `json:"total"`
				Passed int `json:"passed"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if !payload.OK || payload.PoolSite != "domestic" || payload.Model != "auto" || payload.Summary.Total != 1 || payload.Summary.Passed != 1 {
			t.Fatalf("payload=%s", rec.Body.String())
		}
	})

	t.Run("single account", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/direct-admin/api/codebuddy/accounts/"+acc.ID+"/test", strings.NewReader(`{"model":"auto"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:32126")
		req.Host = "127.0.0.1:32126"
		rec := httptest.NewRecorder()
		srv.HTTP.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			OK        bool   `json:"ok"`
			AccountID string `json:"accountId"`
			Model     string `json:"model"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if !payload.OK || payload.AccountID != acc.ID || payload.Model != "auto" {
			t.Fatalf("payload=%s", rec.Body.String())
		}
	})

	t.Run("batch path not swallowed by accounts prefix", func(t *testing.T) {
		// POST /codebuddy/test must not be treated as accounts/{id}=test.
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/direct-admin/api/codebuddy/test", strings.NewReader(`{"model":"auto","site":"global"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:32126")
		req.Host = "127.0.0.1:32126"
		rec := httptest.NewRecorder()
		srv.HTTP.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			PoolSite string `json:"poolSite"`
			Summary  struct {
				Total int `json:"total"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.PoolSite != "global" || payload.Summary.Total != 0 {
			t.Fatalf("payload=%s", rec.Body.String())
		}
	})
}

type roundTripOKChat struct{}

func (roundTripOKChat) RoundTrip(req *http.Request) (*http.Response, error) {
	header := make(http.Header)
	header.Set("Content-Type", "text/event-stream")
	body := "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n"
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func TestAdminCheckinRouteUsesActivePoolSiteByDefault(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantSite string
	}{
		{"empty body falls back to active pool", `{}`, "domestic"},
		{"explicit global overrides", `{"site":"global"}`, "global"},
		{"alias normalizes to domestic", `{"site":"cn"}`, "domestic"},
		{"invalid json falls back to active pool", `not-json`, "domestic"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := testServer(t, false, "", "")
			req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/direct-admin/api/codebuddy/checkin", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "http://127.0.0.1:32126")
			req.Host = "127.0.0.1:32126"
			rec := httptest.NewRecorder()
			srv.HTTP.Handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			var payload struct {
				OK       bool   `json:"ok"`
				PoolSite string `json:"poolSite"`
				Summary  struct {
					Total int `json:"total"`
				} `json:"summary"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.PoolSite != tc.wantSite {
				t.Fatalf("poolSite=%q want=%q", payload.PoolSite, tc.wantSite)
			}
			// Empty pool: no upstream calls, but the batch must still report cleanly.
			if !payload.OK || payload.Summary.Total != 0 {
				t.Fatalf("payload=%s", rec.Body.String())
			}
		})
	}
}

func TestAdminCSRFBlocksCrossOriginMutation(t *testing.T) {
	srv := testServer(t, false, "", "")
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/direct-admin/api/pool-site", strings.NewReader(`{"site":"global"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	req.Host = "127.0.0.1:32126"
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("csrf status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["ok"] != false {
		t.Fatalf("payload=%v", payload)
	}
}

func TestAdminCSRFAllowsSameOriginMutation(t *testing.T) {
	srv := testServer(t, false, "", "")
	t.Setenv("CODEBUDDY_PROXY_ENV_FILE", filepath.Join(t.TempDir(), ".env"))
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/direct-admin/api/pool-site", strings.NewReader(`{"site":"domestic"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:32126")
	req.Host = "127.0.0.1:32126"
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("same-origin status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestManualPoolSwitchLogsAccountSwitch(t *testing.T) {
	srv := testServer(t, false, "", "")
	t.Setenv("CODEBUDDY_PROXY_ENV_FILE", filepath.Join(t.TempDir(), ".env"))
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/direct-admin/api/pool-site", strings.NewReader(`{"site":"domestic"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:32126")
	req.Host = "127.0.0.1:32126"
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("switch status=%d body=%s", rec.Code, rec.Body.String())
	}

	actReq := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/direct-admin/api/system/activity?limit=10", nil)
	actRec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(actRec, actReq)
	if actRec.Code != http.StatusOK {
		t.Fatalf("activity status=%d", actRec.Code)
	}
	var payload struct {
		Entries []struct {
			Kind   string         `json:"kind"`
			Fields map[string]any `json:"fields"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(actRec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, e := range payload.Entries {
		if e.Kind == "account-switch" {
			if e.Fields["mode"] != "manual" || e.Fields["to"] != "domestic" {
				t.Fatalf("account-switch fields=%v", e.Fields)
			}
			if e.Fields["account"] != "号池" {
				t.Fatalf("account-switch missing account name: %v", e.Fields)
			}
			return
		}
	}
	t.Fatalf("no account-switch event logged: %+v", payload.Entries)
}

func TestAdminProductSwitchSameOrigin(t *testing.T) {
	srv := testServer(t, false, "", "")
	t.Setenv("CODEBUDDY_PROXY_ENV_FILE", filepath.Join(t.TempDir(), ".env"))
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/direct-admin/api/pool-product", strings.NewReader(`{"product":"workbuddy"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:32126")
	req.Host = "127.0.0.1:32126"
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("product switch status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["ok"] != true {
		t.Fatalf("payload=%v", payload)
	}
	product, _ := payload["product"].(string)
	poolProduct, _ := payload["poolProduct"].(string)
	if product != "workbuddy" && poolProduct != "workbuddy" {
		t.Fatalf("payload product missing: %v", payload)
	}
}

func TestAdminPasswordRequiredWhenConfigured(t *testing.T) {
	srv := testServer(t, false, "admin-pass", "")
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/direct-admin/api/status", nil)
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/direct-admin/api/status", nil)
	req.SetBasicAuth("admin", "admin-pass")
	rec = httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authed status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Bearer admin password also works.
	req = httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/direct-admin/api/status", nil)
	req.Header.Set("Authorization", "Bearer admin-pass")
	rec = httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("bearer status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminRejectsPasswordQueryParam(t *testing.T) {
	srv := testServer(t, false, "admin-pass", "")
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/direct-admin/api/status?password=admin-pass", nil)
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("query password must be rejected, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminUsageAPI(t *testing.T) {
	srv := testServer(t, false, "", "")
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/direct-admin/api/usage?range=day&limit=5", nil)
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["ok"] != true {
		t.Fatalf("payload=%v", payload)
	}
	if _, ok := payload["summary"]; !ok {
		t.Fatalf("missing summary: %v", payload)
	}
}

func TestHealth(t *testing.T) {
	srv := testServer(t, false, "", "")
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/health", nil)
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health=%d", rec.Code)
	}
}

type stubProbeTransport struct {
	status int
	body   string
}

func (t stubProbeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	header := make(http.Header)
	header.Set("Content-Type", "text/plain")
	return &http.Response{
		StatusCode: t.status,
		Status:     http.StatusText(t.status),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Request:    req,
	}, nil
}

func TestReadyzAndDeepHealthFollowUpstream(t *testing.T) {
	srv := testServer(t, false, "", "")
	srv.Svc.Provider.HTTP = &http.Client{Transport: stubProbeTransport{status: http.StatusBadGateway, body: "<html>openresty"}}

	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/health?deep=1", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("deep health=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("liveness health should stay 200, got %d", rec.Code)
	}
}

func TestReadyzOKWhenAuthLayerAnswers(t *testing.T) {
	srv := testServer(t, false, "", "")
	srv.Svc.Provider.HTTP = &http.Client{Transport: stubProbeTransport{status: http.StatusUnauthorized, body: "Authorization Required"}}
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestResolveIncludeUsage(t *testing.T) {
	if !resolveIncludeUsage(nil) {
		t.Fatal("nil stream_options should default to include usage")
	}
	if !resolveIncludeUsage(&streamOptions{}) {
		t.Fatal("absent include_usage should default to true")
	}
	yes, no := true, false
	if !resolveIncludeUsage(&streamOptions{IncludeUsage: &yes}) {
		t.Fatal("explicit true")
	}
	if resolveIncludeUsage(&streamOptions{IncludeUsage: &no}) {
		t.Fatal("explicit false must skip usage trailer")
	}
}

func seedBothSites(t *testing.T, srv *Server) {
	t.Helper()
	if _, _, err := srv.Svc.Pool.Upsert(accounts.CreateAccount(accounts.Account{
		Label: "global", Site: "global", BearerToken: "token-global", Enabled: true,
	})); err != nil {
		t.Fatal(err)
	}
	if _, _, err := srv.Svc.Pool.Upsert(accounts.CreateAccount(accounts.Account{
		Label: "domestic", Site: "domestic", BearerToken: "token-domestic", Enabled: true,
	})); err != nil {
		t.Fatal(err)
	}
}

func modelIDsFromList(t *testing.T, raw []byte) []string {
	t.Helper()
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode models: %v body=%s", err, raw)
	}
	ids := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		ids = append(ids, item.ID)
	}
	return ids
}

func containsID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

type recordingUpstreamTransport struct {
	mu    sync.Mutex
	chats []string
}

func (t *recordingUpstreamTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	header := make(http.Header)
	token := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	if strings.Contains(req.URL.Path, "chat") {
		t.mu.Lock()
		t.chats = append(t.chats, token)
		t.mu.Unlock()
		header.Set("Content-Type", "text/event-stream")
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     http.StatusText(http.StatusOK),
			Header:     header,
			Body:       io.NopCloser(strings.NewReader("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n")),
			Request:    req,
		}, nil
	}
	header.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(`{"code":0,"data":{}}`)),
		Request:    req,
	}, nil
}

func (t *recordingUpstreamTransport) chatTokens() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string{}, t.chats...)
}

type catalogByHostTransport struct{}

func (catalogByHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	token := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	ids := []string{"deepseek-v4.1-flash", "gpt-5"}
	if token == "token-domestic" || strings.Contains(req.URL.Host, ".cn") {
		ids = []string{"deepseek-v4.1-flash", "glm-5.3-flash"}
	}
	rows := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, map[string]any{"id": id, "name": id})
	}
	body, _ := json.Marshal(map[string]any{"models": rows})
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Request:    req,
	}, nil
}

func TestClientConfigIncludesBoundAPIKeys(t *testing.T) {
	srv := testServerCfg(t, config.Config{
		APIKey:        "cbp_primary",
		RequireAPIKey: true,
		APIKeys:       config.ParseAPIKeys("cbp_aaa:global,cbp_bbb:domestic"),
	})
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/direct-admin/api/client-config", nil)
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		OK               bool   `json:"ok"`
		APIKey           string `json:"apiKey"`
		APIKeyPreview    string `json:"apiKeyPreview"`
		APIKeyConfigured bool   `json:"apiKeyConfigured"`
		APIKeys          []struct {
			Site       string `json:"site"`
			Preview    string `json:"preview"`
			Configured bool   `json:"configured"`
			Key        string `json:"key"`
		} `json:"apiKeys"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	// H2：列表接口不再返回完整主 Key 与绑定 Key。
	if payload.APIKey != "" || !payload.APIKeyConfigured || payload.APIKeyPreview == "" {
		t.Fatalf("primary apiKey must be masked: %+v", payload)
	}
	if len(payload.APIKeys) != 2 {
		t.Fatalf("apiKeys=%d want 2 body=%s", len(payload.APIKeys), rec.Body.String())
	}
	bySite := map[string]string{}
	for _, item := range payload.APIKeys {
		if !item.Configured || item.Preview == "" || item.Key != "" {
			t.Fatalf("bound key must be masked without key: %+v", item)
		}
		bySite[item.Site] = item.Preview
	}
	if bySite["global"] == "" || bySite["domestic"] == "" {
		t.Fatalf("apiKeys=%v", bySite)
	}

	// reveal 端点返回完整密钥（仍需鉴权）。
	reveal := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/direct-admin/api/client-config/reveal", nil)
	revealRec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(revealRec, reveal)
	if revealRec.Code != http.StatusOK {
		t.Fatalf("reveal status=%d body=%s", revealRec.Code, revealRec.Body.String())
	}
	var revealed struct {
		APIKey  string `json:"apiKey"`
		APIKeys []struct {
			Site string `json:"site"`
			Key  string `json:"key"`
		} `json:"apiKeys"`
	}
	if err := json.Unmarshal(revealRec.Body.Bytes(), &revealed); err != nil {
		t.Fatal(err)
	}
	if revealed.APIKey != "cbp_primary" {
		t.Fatalf("reveal apiKey=%q", revealed.APIKey)
	}
	full := map[string]string{}
	for _, item := range revealed.APIKeys {
		full[item.Site] = item.Key
	}
	if full["global"] != "cbp_aaa" || full["domestic"] != "cbp_bbb" {
		t.Fatalf("reveal apiKeys=%v", full)
	}
}

func TestGenerateAndDeleteBoundAPIKey(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), "proxy.env")
	t.Setenv("CODEBUDDY_PROXY_ENV_FILE", envPath)
	srv := testServerCfg(t, config.Config{
		APIKey:  "cbp_primary",
		APIKeys: config.ParseAPIKeys("cbp_old:global"),
	})

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/direct-admin/api/client-config/bound-keys", strings.NewReader(`{"site":"domestic"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("generate status=%d body=%s", rec.Code, rec.Body.String())
	}
	var generated struct {
		OK      bool   `json:"ok"`
		APIKey  string `json:"apiKey"`
		Created struct {
			Site string `json:"site"`
			Key  string `json:"key"`
		} `json:"created"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &generated); err != nil {
		t.Fatal(err)
	}
	// H2：主 Key 列表掩码；新 Key 走 created 反查返回值。
	if generated.APIKey != "" {
		t.Fatalf("primary apiKey must be masked in list payload: %q", generated.APIKey)
	}
	newKey := generated.Created.Key
	if generated.Created.Site != "domestic" || newKey == "" || !strings.HasPrefix(newKey, "cbp_") {
		t.Fatalf("created bound key shape=%+v", generated.Created)
	}

	envRaw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	envText := string(envRaw)
	if !strings.Contains(envText, "CODEBUDDY_PROXY_API_KEYS=") {
		t.Fatalf("env missing CODEBUDDY_PROXY_API_KEYS: %s", envText)
	}
	parsed := config.ParseAPIKeys(envAPIKeysValue(envText))
	gotKeys := map[string]string{}
	for _, b := range parsed {
		gotKeys[b.Key] = b.Site
	}
	if gotKeys["cbp_old"] != "global" || gotKeys[newKey] != "domestic" {
		t.Fatalf("env bindings=%v", gotKeys)
	}

	cfg := srv.Svc.Config()
	if ok, site := cfg.LookupAPIKey("cbp_old"); !ok || site != "global" {
		t.Fatalf("old key lookup=(%v,%q)", ok, site)
	}
	if ok, site := cfg.LookupAPIKey(newKey); !ok || site != "domestic" {
		t.Fatalf("new key lookup=(%v,%q)", ok, site)
	}
	if ok, _ := cfg.LookupAPIKey("cbp_primary"); !ok {
		t.Fatal("primary key should still authenticate")
	}

	// H2：按站点删除（不再回传密钥定位）。
	del := httptest.NewRequest(http.MethodDelete, "http://127.0.0.1:32126/direct-admin/api/client-config/bound-keys", strings.NewReader(`{"site":"domestic"}`))
	del.Header.Set("Content-Type", "application/json")
	delRec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(delRec, del)
	if delRec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", delRec.Code, delRec.Body.String())
	}
	cfg = srv.Svc.Config()
	if ok, _ := cfg.LookupAPIKey(newKey); ok {
		t.Fatal("deleted bound key still authenticates")
	}
	if ok, site := cfg.LookupAPIKey("cbp_old"); !ok || site != "global" {
		t.Fatalf("remaining bound key lookup=(%v,%q)", ok, site)
	}
	if ok, _ := cfg.LookupAPIKey("cbp_primary"); !ok {
		t.Fatal("primary key must not be deleted by bound-key list")
	}
	envRaw, err = os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(envRaw), newKey) {
		t.Fatalf("deleted key still in env: %s", envRaw)
	}
}

func TestAdminStatusDoesNotHitBillingByDefault(t *testing.T) {
	srv := testServer(t, false, "", "")
	seedBothSites(t, srv)
	transport := &adminCreditsTransport{}
	srv.Svc.Provider.HTTP = &http.Client{Transport: transport}

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/direct-admin/api/status", nil)
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if n := transport.billingCount(); n != 0 {
		t.Fatalf("plain status must not hit billing, got %d calls", n)
	}
	if strings.Contains(rec.Body.String(), `"creditsRefreshed":true`) {
		t.Fatalf("plain status must not claim credits refreshed: %s", rec.Body.String())
	}
}

func TestAdminStatusFreshRefreshesCreditsFromUpstream(t *testing.T) {
	srv := testServer(t, false, "", "")
	seedBothSites(t, srv)
	transport := &adminCreditsTransport{remaining: 88}
	srv.Svc.Provider.HTTP = &http.Client{Transport: transport}

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/direct-admin/api/status?fresh=1", nil)
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if n := transport.billingCount(); n == 0 {
		t.Fatal("fresh=1 must hit upstream billing")
	}
	var payload struct {
		CreditsRefreshed bool `json:"creditsRefreshed"`
		AccountUsages    []struct {
			OK        bool   `json:"ok"`
			AccountID string `json:"accountId"`
			Credits   *struct {
				Remaining *float64 `json:"remaining"`
			} `json:"credits"`
		} `json:"accountUsages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if !payload.CreditsRefreshed {
		t.Fatalf("creditsRefreshed missing: %s", rec.Body.String())
	}
	found := false
	for _, item := range payload.AccountUsages {
		if item.OK && item.Credits != nil && item.Credits.Remaining != nil && *item.Credits.Remaining == 88 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("fresh status missing live credits: %s", rec.Body.String())
	}
}

func TestAdminPageRefreshButtonRequestsFreshCredits(t *testing.T) {
	html := admin.PageHTML()
	if !strings.Contains(html, "refreshStatus(true)") {
		t.Fatal("refresh button must call refreshStatus(true)")
	}
	if !strings.Contains(html, "fresh ? '?fresh=1'") && !strings.Contains(html, `fresh ? "?fresh=1"`) {
		t.Fatal("refreshStatus(true) must request status?fresh=1")
	}
	if strings.Contains(html, "setInterval(function(){ refreshStatus(true)") {
		t.Fatal("15s poll must not pass fresh=true")
	}
}

func TestAdminPageStatusPillsDoNotWrapInternally(t *testing.T) {
	html := admin.PageHTML()
	if !strings.Contains(html, ".pill{") || !strings.Contains(html, "white-space:nowrap") {
		t.Fatal("status pills must keep white-space:nowrap so long health text cannot break each label")
	}
	if !strings.Contains(html, "flex-shrink:0") && !strings.Contains(html, "flex:0 0 auto") {
		t.Fatal("pillrow/pills must not shrink into per-character wrapping")
	}
	if !strings.Contains(html, `content="`+admin.UIRevision+`"`) {
		t.Fatalf("cbp-ui-revision meta must match UIRevision %q", admin.UIRevision)
	}
	if admin.UIRevision == "2026.09.22-chat-test" {
		t.Fatal("UIRevision must bump when status-bar CSS changes")
	}
}

type adminCreditsTransport struct {
	mu        sync.Mutex
	billing   int
	remaining float64
}

func (t *adminCreditsTransport) billingCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.billing
}

func (t *adminCreditsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	if strings.Contains(req.URL.Path, "get-user-resource") {
		t.mu.Lock()
		t.billing++
		remain := t.remaining
		t.mu.Unlock()
		body := fmt.Sprintf(`{"code":0,"data":{"Response":{"Data":{"Accounts":[{"CapacityRemain":%g,"CapacitySize":1000,"CapacityUsed":0,"CapacityType":1}]}}}}`, remain)
		return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	}
	if strings.Contains(req.URL.Path, "get-dosage-notify") {
		return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"dosageNotifyCode":0}}`)), Request: req}, nil
	}
	header.Set("Content-Type", "text/plain")
	return &http.Response{StatusCode: http.StatusUnauthorized, Header: header, Body: io.NopCloser(strings.NewReader("Authorization Required")), Request: req}, nil
}

func envAPIKeysValue(envText string) string {
	for _, line := range strings.Split(envText, "\n") {
		line = strings.TrimSpace(line)
		if key, val, ok := strings.Cut(line, "="); ok && strings.TrimSpace(key) == "CODEBUDDY_PROXY_API_KEYS" {
			return strings.Trim(strings.TrimSpace(val), `"'`)
		}
	}
	return ""
}

func TestModelPolicyAdminAPI(t *testing.T) {
	srv := testServer(t, false, "", "")
	t.Setenv("CODEBUDDY_PROXY_ENV_FILE", filepath.Join(t.TempDir(), ".env"))

	get := func() map[string]any {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/direct-admin/api/system/model-policy", nil)
		rec := httptest.NewRecorder()
		srv.HTTP.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET status=%d body=%s", rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}

	if p := get(); p["enabled"] != false {
		t.Fatalf("default policy should be disabled, got %v", p["enabled"])
	}

	putReq := httptest.NewRequest(http.MethodPut, "http://127.0.0.1:32126/direct-admin/api/system/model-policy",
		strings.NewReader(`{"enabled":true,"allow":["gpt-5"],"deny":[]}`))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Origin", "http://127.0.0.1:32126")
	putReq.Host = "127.0.0.1:32126"
	putRec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status=%d body=%s", putRec.Code, putRec.Body.String())
	}

	if p := get(); p["enabled"] != true {
		t.Fatalf("policy should be enabled, got %v", p["enabled"])
	}

	pol, err := srv.Svc.ModelPolicy.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(pol.Allow) != 1 || pol.Allow[0] != "gpt-5" {
		t.Fatalf("allow=%v", pol.Allow)
	}
}

func TestModelPolicyFiltersCatalog(t *testing.T) {
	srv := testServer(t, false, "", "")
	seedBothSites(t, srv)
	srv.Svc.Provider.HTTP = &http.Client{Transport: catalogByHostTransport{}}
	t.Setenv("CODEBUDDY_PROXY_ENV_FILE", filepath.Join(t.TempDir(), ".env"))

	if err := srv.Svc.ModelPolicy.Write(modelpolicy.Policy{Enabled: true, Allow: []string{"gpt-5"}}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:32126/v1/models", nil)
	req.Header.Set("X-Site", "global")
	rec := httptest.NewRecorder()
	srv.HTTP.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	ids := modelIDsFromList(t, rec.Body.Bytes())
	if !containsID(ids, "gpt-5") || containsID(ids, "deepseek-v4.1-flash") {
		t.Fatalf("filtered catalog=%v", ids)
	}
}

func TestModelPolicyRejectsChat(t *testing.T) {
	srv := testServer(t, true, "", "secret-key")
	t.Setenv("CODEBUDDY_PROXY_ENV_FILE", filepath.Join(t.TempDir(), ".env"))
	if err := srv.Svc.ModelPolicy.Write(modelpolicy.Policy{Enabled: true, Allow: []string{"gpt-5"}}); err != nil {
		t.Fatal(err)
	}

	mk := func(model string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:32126/v1/chat/completions",
			strings.NewReader(`{"model":"`+model+`","messages":[{"role":"user","content":"hi"}]}`))
		req.Header.Set("Authorization", "Bearer secret-key")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.HTTP.Handler.ServeHTTP(rec, req)
		return rec
	}

	if rec := mk("deepseek-v4.1-flash"); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "白名单") {
		t.Fatalf("blocked status=%d body=%s", rec.Code, rec.Body.String())
	}
}
