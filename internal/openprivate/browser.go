package openprivate

import (
	"errors"
	"net/url"
	"strings"
)

// ErrPrivateWindowUnsupported 表示当前平台没有跨浏览器统一的无痕窗口启动方式。
var ErrPrivateWindowUnsupported = errors.New("private window launch is not supported on this platform")

// normalizeAuthURL 校验准备交给浏览器的授权链接，只接受带主机的 http(s) 绝对地址。
// 链接会作为命令行参数传给浏览器，先挡掉 javascript: 之类的非 HTTP scheme。
func normalizeAuthURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("oauth url cannot be parsed")
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", errors.New("oauth url is not an absolute http(s) url")
	}
	return parsed.String(), nil
}
