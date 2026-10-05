package openprivate

import "testing"

func TestNormalizeAuthURL(t *testing.T) {
	accepted := map[string]string{
		"https://www.codebuddy.ai/login?platform=CLI&state=abc": "https://www.codebuddy.ai/login?platform=CLI&state=abc",
		"  http://127.0.0.1:32126/x  ":                          "http://127.0.0.1:32126/x",
	}
	for raw, want := range accepted {
		got, err := normalizeAuthURL(raw)
		if err != nil {
			t.Fatalf("normalizeAuthURL(%q) error: %v", raw, err)
		}
		if got != want {
			t.Fatalf("normalizeAuthURL(%q) = %q, want %q", raw, got, want)
		}
	}

	rejected := []string{
		"",
		"not a url",
		"www.codebuddy.ai/login",
		"javascript:alert(1)",
		"file:///C:/Windows/win.ini",
	}
	for _, raw := range rejected {
		if got, err := normalizeAuthURL(raw); err == nil {
			t.Fatalf("normalizeAuthURL(%q) = %q, want error", raw, got)
		}
	}
}

// OpenInPrivateWindow 只有非法链接才会在任何平台上确定性失败；合法链接会真的拉起
// 浏览器，所以这里不覆盖那条路径。
func TestOpenInPrivateWindowRejectsNonHTTP(t *testing.T) {
	if browser, err := OpenInPrivateWindow("javascript:alert(1)"); err == nil || browser != "" {
		t.Fatalf("OpenInPrivateWindow javascript: = (%q, %v), want empty browser and error", browser, err)
	}
}
