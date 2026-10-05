//go:build !windows

package openprivate

// OpenInPrivateWindow 在非 Windows 上没有跨浏览器统一的无痕窗口开关，直接回
// ErrPrivateWindowUnsupported，由调用方把链接展示出来让用户自己开无痕窗口。
func OpenInPrivateWindow(raw string) (string, error) {
	if _, err := normalizeAuthURL(raw); err != nil {
		return "", err
	}
	return "", ErrPrivateWindowUnsupported
}
