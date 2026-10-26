//go:build windows

package singleinstance

import (
	"errors"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")

	procCreateMutexW = kernel32.NewProc("CreateMutexW")
	procCloseHandle  = kernel32.NewProc("CloseHandle")
	procMessageBoxW  = user32.NewProc("MessageBoxW")
)

const (
	ERROR_ALREADY_EXISTS = 183
	MF_OK                = 0x00000000
	MB_ICONWARNING       = 0x00000030
	MB_TOPMOST           = 0x00000080
	MB_SETFOREGROUND     = 0x00000100
)

// Mutex 表示一个进程级独占锁，持有时禁止其他实例以同一名字获取。
type Mutex struct {
	handle syscall.Handle
}

// ErrAlreadyRunning 表示同名实例已在运行。
var ErrAlreadyRunning = errors.New("another instance is already running")

// Acquire 尝试获取指定名字的命名互斥量。已存在（另一实例运行中）时返回
// ErrAlreadyRunning，句柄已立即释放，调用方可继续做提示/退出。
func Acquire(name string) (*Mutex, error) {
	utf16Name, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	h, _, callErr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(utf16Name)))
	if h == 0 {
		return nil, callErr
	}
	if callErr != nil && errors.Is(callErr, syscall.Errno(ERROR_ALREADY_EXISTS)) {
		procCloseHandle.Call(h)
		return nil, ErrAlreadyRunning
	}
	return &Mutex{handle: syscall.Handle(h)}, nil
}

// Release 释放互斥量句柄。进程退出时内核会自动回收，此处显式调用只是为了
// 尽早让其他实例能抢占，方便立即重启。
func (m *Mutex) Release() {
	if m == nil || m.handle == 0 {
		return
	}
	procCloseHandle.Call(uintptr(m.handle))
	m.handle = 0
}

// Popup 用系统级 MessageBox 弹出一个模态对话框。用于错误提示（尤其无托盘
// 残留时），标题与正文按 UTF-16 传给 Win32；返回用户点击的按钮码。
//
// 不传窗口句柄作为所有者：启动失败弹窗常发生在登录阶段，此时前台窗口属于
// explorer.exe，而 explorer 的线程正忙于加载 shell 扩展。把外部件句柄当作
// 所有者会让模态对话框等待对方线程的泵消息锁，结果是"桌面卡死"——对话框
// 永不响应。无所有者 + TOPMOST/SETFOREGROUND 保证对话框自身始终可点击。
func Popup(title, text string, icon uint32) uint32 {
	t16, _ := syscall.UTF16PtrFromString(title)
	m16, _ := syscall.UTF16PtrFromString(text)
	r, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(m16)),
		uintptr(unsafe.Pointer(t16)),
		uintptr(MF_OK|icon|MB_TOPMOST|MB_SETFOREGROUND),
	)
	return uint32(r)
}
