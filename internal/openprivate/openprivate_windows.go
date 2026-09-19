//go:build windows

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

// OpenDirectory 在资源管理器中打开指定目录。
func OpenDirectory(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	file, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if file.IsDir() {
		return exec.Command("explorer", abs).Start()
	}
	return exec.Command("explorer", "/select,", abs).Start()
}

// OpenConfigDir 打开私有配置目录。
func OpenConfigDir() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	return OpenDirectory(dir)
}
