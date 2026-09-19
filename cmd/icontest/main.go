//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	kernel      = syscall.NewLazyDLL("kernel32.dll")
	pGetModuleH = kernel.NewProc("GetModuleHandleW")
	pFindRes    = kernel.NewProc("FindResourceW")
	pLoadRes    = kernel.NewProc("LoadResource")
	pSizeRes    = kernel.NewProc("SizeofResource")
	pLockRes    = kernel.NewProc("LockResource")
	pLoadImageW = user32.NewProc("LoadImageW")
)

const RT_ICON = 3
const RT_GROUP_ICON = 14
const RT_MANIFEST = 24

func find(t uint32, name uintptr) (uintptr, uintptr) {
	hInst, _, _ := pGetModuleH.Call(0)
	hRes, _, _ := pFindRes.Call(hInst, name, uintptr(t))
	return hInst, hRes
}

func main() {
	hInst, _, _ := pGetModuleH.Call(0)
	fmt.Printf("hInstance=0x%x\n", hInst)
	for _, t := range []uint32{RT_ICON, RT_GROUP_ICON, RT_MANIFEST, 1, 14, 3, 24} {
		_, hRes := find(t, 1)
		fmt.Printf("FindResourceW(type=%d,id=1): 0x%x\n", t, hRes)
	}
	// Also try LoadIconW group icon
	pLoadIcon := user32.NewProc("LoadIconW")
	hIcon, _, _ := pLoadIcon.Call(hInst, 1)
	fmt.Printf("LoadIconW(hInst,1)=0x%x\n", hIcon)
	_ = unsafe.Pointer(&pLockRes)
	_ = pLoadRes
	_ = pSizeRes
}
