// Package updatecheck 监控上游仓库的最新发布版本，供管理台在出现新版本时弹窗提示。
package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wnddd839/codebuddy-proxy/internal/httputil"
)

// DefaultRepo 是本代理跟踪的上游仓库（环境变量 CODEBUDDY_PROXY_UPSTREAM_REPO 可覆盖）。
const DefaultRepo = "wnddd839/buddy-proxy"

var apiBase = "https://api.github.com"

// networkTimeout 限制单次 GitHub 请求耗时，避免离线/墙内环境卡住管理台。
const networkTimeout = 8 * time.Second

// cacheTTL 是两次 GitHub 请求的最小间隔。带上令牌后限额为 5000 次/时，
// 匿名则为 60 次/时且按出口 IP 计——共享代理下很容易被陌生人占满。
const cacheTTL = 6 * time.Hour

// errorCacheTTL 是失败结论的有效期。网络抖动或匿名限额 403 不该把「检查失败」
// 钉住整个 cacheTTL，否则管理台要等 6 小时才会再试一次。
const errorCacheTTL = 15 * time.Minute

// Result 是一次版本检查的结论。
type Result struct {
	Repo            string `json:"repo"`
	CurrentVersion  string `json:"currentVersion"`
	LatestTag       string `json:"latestTag"`
	UpdateAvailable bool   `json:"updateAvailable"`
	Comparable      bool   `json:"comparable"`
	URL             string `json:"url"`
	PublishedAt     string `json:"publishedAt"`
	Notes           string `json:"notes"`
	CheckedAt       string `json:"checkedAt"`
	Error           string `json:"error,omitempty"`
}

// githubRelease 是 GitHub API 返回对象里我们关心的字段。
type githubRelease struct {
	TagName     string `json:"tag_name"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
}

// Updater 带进程内缓存的版本检查器，线程安全。
type Updater struct {
	mu       sync.Mutex
	client   *http.Client
	last     *Result
	checked  time.Time
	inflight bool
}

// New 创建版本检查器。client 为 nil 时使用默认超时客户端。
func New(client *http.Client) *Updater {
	if client == nil {
		client = &http.Client{Timeout: networkTimeout, CheckRedirect: httputil.SameOriginRedirectPolicy}
	}
	return &Updater{client: client}
}

// Check 返回最新版本对比结果。TTL 内直接命中缓存，不重复请求 GitHub；
// 缓存过期（或没有缓存）时发起一次网络请求。repo 为空使用 DefaultRepo。
func (u *Updater) Check(ctx context.Context, repo, current string) *Result {
	repo = normalizeRepo(repo)

	u.mu.Lock()
	ttl := cacheTTL
	if u.last != nil && u.last.Error != "" {
		ttl = errorCacheTTL
	}
	if u.last != nil && u.last.Repo == repo && time.Since(u.checked) < ttl {
		res := finish(*u.last, current)
		u.mu.Unlock()
		return &res
	}
	if u.inflight {
		if u.last != nil && u.last.Repo == repo {
			res := finish(*u.last, current)
			u.mu.Unlock()
			return &res
		}
		u.mu.Unlock()
		return &Result{Repo: repo, CurrentVersion: current, CheckedAt: nowRFC3339(), Error: "update check in progress"}
	}
	u.inflight = true
	client := u.client
	u.mu.Unlock()

	base := fetch(ctx, client, repo)
	base.Repo = repo

	u.mu.Lock()
	u.inflight = false
	u.last = base
	u.checked = time.Now()
	res := finish(*base, current)
	u.mu.Unlock()
	return &res
}

// Warm 在后台发起一次检查（幂等，供服务器启动时预热缓存）。
func (u *Updater) Warm(ctx context.Context, repo string) {
	go u.Check(ctx, repo, "")
}

// finish 用当前运行版本补全对比结论并返回副本。
func finish(base Result, current string) Result {
	res := base
	res.CurrentVersion = current
	if res.Error != "" || res.LatestTag == "" {
		res.UpdateAvailable = false
		res.Comparable = false
		return res
	}
	greater, comparable := Compare(current, res.LatestTag)
	res.UpdateAvailable = comparable && greater
	res.Comparable = comparable
	return res
}

// fetch 抓取 latest release 的基本信息（不含当前版本对比）。
func fetch(ctx context.Context, client *http.Client, repo string) *Result {
	res := &Result{CheckedAt: nowRFC3339()}
	rel, err := getLatestRelease(ctx, client, repo)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.LatestTag = rel.TagName
	res.URL = rel.HTMLURL
	res.PublishedAt = rel.PublishedAt
	res.Notes = truncateNotes(rel.Body)
	return res
}

func getLatestRelease(ctx context.Context, client *http.Client, repo string) (githubRelease, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/releases/latest", apiBase, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "codebuddy-proxy/updatecheck")
	// 令牌只走 https，且不落日志；缺失时退回匿名请求。
	if token := githubToken(); token != "" && strings.HasPrefix(endpoint, "https://") {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			return githubRelease{}, fmt.Errorf("github api status %d (rate limited; set GITHUB_TOKEN to raise the cap)", resp.StatusCode)
		}
		return githubRelease{}, fmt.Errorf("github api status %d", resp.StatusCode)
	}
	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return githubRelease{}, err
	}
	if strings.TrimSpace(rel.TagName) == "" {
		return githubRelease{}, fmt.Errorf("release tag empty")
	}
	return rel, nil
}

// githubToken 返回可选的 GitHub 令牌，供版本检查提高 API 限额。
func githubToken() string {
	for _, name := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}

// Compare 比较当前版本与最新 tag：最新 > 当前返回 true。
// 双方都能解析出数字版本号才比较；解析失败时 comparable=false。
func Compare(current, latest string) (greater, comparable bool) {
	cc, cok := parseInts(current)
	ll, lok := parseInts(latest)
	if !cok || !lok || len(cc) == 0 || len(ll) == 0 {
		return false, false
	}
	n := len(cc)
	if len(ll) < n {
		n = len(ll)
	}
	for i := 0; i < n; i++ {
		if ll[i] != cc[i] {
			return ll[i] > cc[i], true
		}
	}
	return len(ll) > len(cc), true
}

var versionRe = regexp.MustCompile(`(?i)(?:^|\D)v?(\d+)(?:\.(\d+))?(?:\.(\d+))?`)

// parseInts 提取版本字符串里的数字版本段（主/次/修订）。
func parseInts(s string) ([]int, bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "dev") {
		return nil, false
	}
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return nil, false
	}
	out := make([]int, 0, 3)
	for _, g := range m[1:] {
		if g == "" {
			break
		}
		if n, err := strconv.Atoi(g); err == nil {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func truncateNotes(body string) string {
	body = strings.TrimSpace(body)
	if len(body) > 800 {
		body = body[:800] + "…"
	}
	return body
}

func normalizeRepo(repo string) string {
	repo = strings.TrimSpace(repo)
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "http://github.com/")
	repo = strings.Trim(repo, "/")
	if repo == "" {
		return DefaultRepo
	}
	return repo
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
