//go:build windows

package openprivate

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type privateBrowser struct {
	name     string
	exe      string
	flags    []string
	relPaths []string
}

// privateBrowsers 是无痕窗口候选，按优先级排列。Edge 是 Windows 的默认浏览器，
// Chrome 作为退路；两者都只认各自的隐私模式开关，不用普通窗口（普通窗口会复用
// 浏览器里已登录的 CodeBuddy 会话，让"换一个账号"变成"同一个账号再授权一次"）。
var privateBrowsers = []privateBrowser{
	{
		name:  "edge",
		exe:   "msedge.exe",
		flags: []string{"--inprivate"},
		relPaths: []string{
			filepath.Join("Microsoft", "Edge", "Application", "msedge.exe"),
		},
	},
	{
		name:  "chrome",
		exe:   "chrome.exe",
		flags: []string{"--incognito"},
		relPaths: []string{
			filepath.Join("Google", "Chrome", "Application", "chrome.exe"),
		},
	},
}

// OpenInPrivateWindow 用无痕窗口打开授权链接，返回实际拉起浏览器的名字。
// 浏览器已在运行时会把参数移交给现有进程并立刻退出，所以这里只 Start 不 Wait。
func OpenInPrivateWindow(raw string) (string, error) {
	target, err := normalizeAuthURL(raw)
	if err != nil {
		return "", err
	}
	for _, browser := range privateBrowsers {
		path, ok := locatePrivateBrowser(browser)
		if !ok {
			continue
		}
		cmd := exec.Command(path, append(append([]string{}, browser.flags...), target)...)
		if err := cmd.Start(); err != nil {
			continue
		}
		_ = cmd.Process.Release()
		return browser.name, nil
	}
	return "", errors.New("no Edge or Chrome installation found")
}

// locatePrivateBrowser 先按 PATH / App Paths 查找，再扫描常见的安装根目录。
func locatePrivateBrowser(browser privateBrowser) (string, bool) {
	if path, err := exec.LookPath(browser.exe); err == nil {
		return path, true
	}
	for _, root := range browserRoots() {
		for _, rel := range browser.relPaths {
			candidate := filepath.Join(root, rel)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, true
			}
		}
	}
	return "", false
}

func browserRoots() []string {
	roots := make([]string, 0, 3)
	for _, key := range []string{"LOCALAPPDATA", "ProgramFiles", "ProgramFiles(x86)"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			roots = append(roots, value)
		}
	}
	return roots
}
