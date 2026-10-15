//go:build !windows

package singleinstance

import "errors"

const (
	MF_OK          = 0x00000000
	MB_ICONWARNING = 0x00000030
)

// Mutex 在 non-Windows 上是 no-op，主要为了让单元测试在 Linux 上跑通。
type Mutex struct{}

// ErrAlreadyRunning 表示同名实例已在运行。
var ErrAlreadyRunning = errors.New("another instance is already running")

func Acquire(name string) (*Mutex, error)          { return &Mutex{}, nil }
func (m *Mutex) Release()                          {}
func Popup(title, text string, icon uint32) uint32 { return 0 }
