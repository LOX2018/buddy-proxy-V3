//go:build windows

package openprivate

import (
	"os"
	"path/filepath"
	"testing"
)

// Edge 必须排在候选首位：它带 --inprivate，是 Windows 上的默认浏览器。
func TestPrivateBrowserOrderPrefersEdge(t *testing.T) {
	if len(privateBrowsers) < 2 || privateBrowsers[0].name != "edge" {
		t.Fatalf("privateBrowsers = %+v, want edge first", privateBrowsers)
	}
	if privateBrowsers[1].name != "chrome" {
		t.Fatalf("second candidate = %q, want chrome", privateBrowsers[1].name)
	}
}

// Edge/Chrome 都不在 PATH 上（msedge.exe 只注册在 App Paths 注册表里，Go 的
// LookPath 不读它），所以安装目录扫描是唯一可行的定位方式。
func TestLocatePrivateBrowserScansInstallRoots(t *testing.T) {
	root := t.TempDir()
	edge := privateBrowsers[0]
	exe := filepath.Join(root, edge.relPaths[0])
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 把 PATH 也指到空目录，确保走的是安装目录扫描而不是 PATH 查找。
	t.Setenv("PATH", t.TempDir())
	for _, key := range []string{"LOCALAPPDATA", "ProgramFiles", "ProgramFiles(x86)"} {
		t.Setenv(key, root)
	}

	got, ok := locatePrivateBrowser(edge)
	if !ok {
		t.Fatalf("locatePrivateBrowser 没找到已存在的 %s", exe)
	}
	if got != exe {
		t.Fatalf("locatePrivateBrowser = %q, want %q", got, exe)
	}
}

func TestBrowserRootsSkipEmptyEnv(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("ProgramFiles", "")
	t.Setenv("ProgramFiles(x86)", "  ")
	if got := browserRoots(); len(got) != 0 {
		t.Fatalf("browserRoots = %v, want empty", got)
	}
}

// 本机真实安装探测：解析不到就只记录，避免在没有浏览器的 CI 上误报。
func TestLocatePrivateBrowserResolvesHostBrowser(t *testing.T) {
	for _, browser := range privateBrowsers {
		if path, ok := locatePrivateBrowser(browser); ok {
			t.Logf("resolved %s -> %s", browser.name, path)
			return
		}
	}
	t.Skip("本机没有 Edge 或 Chrome，跳过真实安装探测")
}
