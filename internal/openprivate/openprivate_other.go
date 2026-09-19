//go:build !windows

package openprivate

import (
	"os"
	"os/exec"
	"path/filepath"
)

// ConfigDir 返回私有配置目录 ~/.codebuddy，不存在则创建。
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ConfigDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// OpenDirectory 用系统默认文件管理器打开指定目录。
func OpenDirectory(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	opener := "xdg-open"
	args := []string{abs}
	if _, err := exec.LookPath(opener); err != nil {
		opener = "open"
	}
	cmd := exec.Command(opener, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	return nil
}

// OpenConfigDir 打开私有配置目录。
func OpenConfigDir() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	return OpenDirectory(dir)
}
