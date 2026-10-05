//go:build windows

package tray

import (
	"encoding/binary"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

type TrayApp struct {
	hwnd          uintptr
	hinstance     uintptr
	OpenURL       string
	StatusURL     string
	Tip           string
	Open          func()
	OpenConfigDir func()
	Quit          func()
}

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW       = user32.NewProc("RegisterClassExW")
	procCreateWindowExW        = user32.NewProc("CreateWindowExW")
	procDefWindowProcW         = user32.NewProc("DefWindowProcW")
	procGetMessageW            = user32.NewProc("GetMessageW")
	procTranslateMessage       = user32.NewProc("TranslateMessage")
	procDispatchMessageW       = user32.NewProc("DispatchMessageW")
	procDestroyWindow          = user32.NewProc("DestroyWindow")
	procPostQuitMessage        = user32.NewProc("PostQuitMessage")
	procCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	procPostMessageW           = user32.NewProc("PostMessageW")
	procAppendMenuW            = user32.NewProc("AppendMenuW")
	procTrackPopupMenu         = user32.NewProc("TrackPopupMenu")
	procDestroyMenu            = user32.NewProc("DestroyMenu")
	procLoadIconW              = user32.NewProc("LoadIconW")
	procLoadImageW             = user32.NewProc("LoadImageW")
	procCreateIconFromResource = user32.NewProc("CreateIconFromResource")
	procLoadCursorW            = user32.NewProc("LoadCursorW")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procGetCursorPos           = user32.NewProc("GetCursorPos")

	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
)

const (
	IMAGE_ICON      = 1
	LR_DEFAULTCOLOR = 0x0000
	SM_CXSMICON     = 49
	SM_CYSMICON     = 50

	WM_USER        = 0x0400
	WM_DESTROY     = 0x0002
	WM_COMMAND     = 0x0111
	WM_LBUTTONDOWN = 0x0201
	WM_RBUTTONUP   = 0x0205

	WM_TRAYICON = WM_USER + 20

	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_INFO    = 0x00000010

	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800

	TPM_RETURNCMD   = 0x0100
	TPM_RIGHTBUTTON = 0x0002

	IDM_OPEN       = 40001
	IDM_STATUS     = 40002
	IDM_ABOUT      = 40003
	IDM_EXIT       = 40004
	IDM_OPENCONFIG = 40005

	IDI_APPLICATION = 32512
	IDC_ARROW       = 32512
	COLOR_WINDOW    = 5

	WS_POPUP         = 0x80000000
	WS_CAPTION       = 0x00C00000
	WS_SYSMENU       = 0x00080000
	WS_THICKFRAME    = 0x00040000
	SW_HIDE          = 0
	WS_EX_TOOLWINDOW = 0x00000080
	SW_SHOW          = 5
)

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type POINT struct {
	X, Y int32
}

type NOTIFYICONDATA struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

var current *TrayApp

//go:uintptrescapes
func callbackProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if current != nil {
		return current.wndProc(hwnd, msg, wParam, lParam)
	}
	return callDefWindowProc(hwnd, msg, wParam, lParam)
}

func callDefWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func (a *TrayApp) wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		switch lParam {
		case WM_LBUTTONDOWN:
			if a.Open != nil {
				a.Open()
			}
		case WM_RBUTTONUP:
			a.showMenu()
		}
	case WM_COMMAND:
		id := wParam & 0xFFFF
		switch id {
		case IDM_OPEN:
			if a.Open != nil {
				a.Open()
			}
		case IDM_STATUS:
			if a.StatusURL != "" {
				openURL(a.StatusURL)
			}
		case IDM_ABOUT:
			if a.Open != nil {
				a.Open()
			}
		case IDM_OPENCONFIG:
			if a.OpenConfigDir != nil {
				a.OpenConfigDir()
			}
		case IDM_EXIT:
			a.removeIcon()
			if a.Quit != nil {
				a.Quit()
			}
			procDestroyWindow.Call(hwnd)
		}
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
	}
	return callDefWindowProc(hwnd, msg, wParam, lParam)
}

func utf16z(s string) *uint16 {
	u, _ := syscall.UTF16PtrFromString(s)
	return u
}

func (a *TrayApp) showMenu() {
	hMenu, _, _ := procCreatePopupMenu.Call()
	procAppendMenuW.Call(hMenu, MF_STRING, IDM_OPEN, uintptr(unsafe.Pointer(utf16z("打开管理台 (Open Console)"))))
	procAppendMenuW.Call(hMenu, MF_STRING, IDM_STATUS, uintptr(unsafe.Pointer(utf16z("打开 /v1 接口 (API URL)"))))
	procAppendMenuW.Call(hMenu, MF_STRING, IDM_OPENCONFIG, uintptr(unsafe.Pointer(utf16z("打开配置目录 (Open Config Dir)"))))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, IDM_ABOUT, uintptr(unsafe.Pointer(utf16z("关于 CodeBuddy Proxy"))))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, IDM_EXIT, uintptr(unsafe.Pointer(utf16z("退出 (Exit)"))))

	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(a.hwnd)
	cmd, _, _ := procTrackPopupMenu.Call(hMenu, TPM_RETURNCMD|TPM_RIGHTBUTTON, uintptr(pt.X), uintptr(pt.Y), 0, a.hwnd, 0)
	procDestroyMenu.Call(hMenu)
	if cmd != 0 {
		a.wndProc(a.hwnd, WM_COMMAND, uintptr(cmd)&0xFFFF, 0)
	}
}

func New(tip string) *TrayApp {
	runtime.LockOSThread()
	return &TrayApp{Tip: tip}
}

func (a *TrayApp) Start() error {
	current = a
	className := utf16z("CodeBuddyProxyTrayWnd")
	hInstance, _, _ := procGetModuleHandleW.Call(0)

	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		Style:         0,
		LpfnWndProc:   syscall.NewCallback(callbackProc),
		CbClsExtra:    0,
		CbWndExtra:    0,
		HInstance:     hInstance,
		HIcon:         loadIcon(hInstance),
		HCursor:       loadCursor(),
		HbrBackground: uintptr(COLOR_WINDOW + 1),
		LpszMenuName:  nil,
		LpszClassName: className,
		HIconSm:       0,
	}
	if r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmtError("RegisterClassExW failed")
	}

	hwnd, _, _ := procCreateWindowExW.Call(
		WS_EX_TOOLWINDOW,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16z("CodeBuddy Proxy"))),
		WS_POPUP,
		0, 0, 0, 0,
		0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		return fmtError("CreateWindowExW failed")
	}
	a.hwnd = hwnd
	a.hinstance = hInstance

	nid := NOTIFYICONDATA{}
	nid.CbSize = uint32(unsafe.Sizeof(NOTIFYICONDATA{}))
	nid.HWnd = hwnd
	nid.UID = 1
	nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	nid.UCallbackMessage = WM_TRAYICON
	nid.HIcon = loadIcon(a.hinstance)
	copyStringToUTF16(nid.SzTip[:], a.Tip)
	r, _, _ := procShellNotifyIconW.Call(uintptr(NIM_ADD), uintptr(unsafe.Pointer(&nid)))
	if r == 0 {
		return fmtError("Shell_NotifyIconW failed to add tray icon")
	}
	return nil
}

func (a *TrayApp) removeIcon() {
	nid := NOTIFYICONDATA{}
	nid.CbSize = uint32(unsafe.Sizeof(NOTIFYICONDATA{}))
	nid.HWnd = a.hwnd
	nid.UID = 1
	procShellNotifyIconW.Call(uintptr(NIM_DELETE), uintptr(unsafe.Pointer(&nid)))
}

func (a *TrayApp) Notify(title, text string) {
	nid := NOTIFYICONDATA{}
	nid.CbSize = uint32(unsafe.Sizeof(NOTIFYICONDATA{}))
	nid.HWnd = a.hwnd
	nid.UID = 1
	nid.UFlags = NIF_INFO
	copyStringToUTF16(nid.SzInfoTitle[:], title)
	copyStringToUTF16(nid.SzInfo[:], text)
	procShellNotifyIconW.Call(uintptr(NIM_MODIFY), uintptr(unsafe.Pointer(&nid)))
}

func (a *TrayApp) Loop() {
	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 {
			break
		}
		if r == 0xFFFFFFFF {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// RequestQuit posts a WM_COMMAND/IDM_EXIT to the tray window thread from any
// goroutine (e.g. a signal handler), so the message pump exits cleanly.
func (a *TrayApp) RequestQuit() {
	if a.hwnd == 0 {
		return
	}
	procPostMessageW.Call(a.hwnd, WM_COMMAND, IDM_EXIT, 0)
}

// Cleanup removes the tray icon and destroys the hidden window.
func (a *TrayApp) Cleanup() {
	if a.hwnd == 0 {
		return
	}
	a.removeIcon()
	procDestroyWindow.Call(a.hwnd)
	a.hwnd = 0
}

func loadIcon(hInstance uintptr) uintptr {
	h := createGradientIcon()
	if h != 0 {
		return h
	}
	r, _, _ := procLoadIconW.Call(0, IDI_APPLICATION)
	return r
}

func createGradientIcon() uintptr {
	const (
		w, h = 32, 32
		bpp  = 32
	)
	header := [40]byte{}
	binary.LittleEndian.PutUint32(header[0:], 40)
	binary.LittleEndian.PutUint32(header[4:], w)
	binary.LittleEndian.PutUint32(header[8:], h*2)
	binary.LittleEndian.PutUint16(header[12:], 1)
	binary.LittleEndian.PutUint16(header[14:], bpp)

	pixels := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			t := float64(x+y) / float64(w+h-2)
			pixels[off+0] = uint8(168*(1-t) + 74*t)
			pixels[off+1] = uint8(196*(1-t) + 138*t)
			pixels[off+2] = uint8(94*(1-t) + 232*t)
			pixels[off+3] = 255
		}
	}
	flipped := make([]byte, w*h*4)
	rowBytes := w * 4
	for y := 0; y < h; y++ {
		src := y * rowBytes
		dst := (h - 1 - y) * rowBytes
		copy(flipped[dst:dst+rowBytes], pixels[src:src+rowBytes])
	}
	andMask := make([]byte, ((w+31)/32)*4*h)

	data := make([]byte, 40+len(flipped)+len(andMask))
	copy(data, header[:])
	copy(data[40:], flipped)
	copy(data[40+len(flipped):], andMask)

	hIcon, _, _ := procCreateIconFromResource.Call(
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		1,
		0x00030000,
	)
	return hIcon
}

func loadCursor() uintptr {
	r, _, _ := procLoadCursorW.Call(0, IDC_ARROW)
	return r
}

func copyStringToUTF16(dst []uint16, s string) {
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		u = []uint16{0}
	}
	for i := 0; i < len(dst)-1 && i < len(u); i++ {
		dst[i] = u[i]
	}
	dst[len(dst)-1] = 0
}

func openURL(target string) {
	launchURL(target)
}

type errorString string

func (e errorString) Error() string { return string(e) }

func fmtError(s string) error { return errorString(s) }

func OpenURL(target string) {
	launchURL(target)
}

// launchURL 用 rundll32 调默认浏览器打开 http(s) 链接。
// 相比 `cmd /c start`，不经过 cmd 解析，避免 URL 中的元字符被当作命令执行，
// 同时只放行 http/https 协议。
func launchURL(target string) {
	target = strings.TrimSpace(target)
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return
	}
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
}
