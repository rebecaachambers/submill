//go:build windows

package worker

import (
	"sync"
	"syscall"
	"unsafe"
)

// ===== Win32 DLLs =====
var (
	shell32 = syscall.NewLazyDLL("shell32.dll")

	procShellNotifyIcon  = shell32.NewProc("Shell_NotifyIconW")
	procExtractIconEx    = shell32.NewProc("ExtractIconExW")
	procDestroyIcon      = user32.NewProc("DestroyIcon")
	procCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	procAppendMenu       = user32.NewProc("AppendMenuW")
	procTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	procDestroyMenu      = user32.NewProc("DestroyMenu")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procLoadIcon         = user32.NewProc("LoadIconW")
)

const (
	// Shell_NotifyIcon
	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	// NOTIFYICONDATA flags
	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_STATE   = 0x00000008

	// Custom message
	WM_TRAYICON = 0x8000 + 1

	// Window messages
	WM_DESTROY   = 0x0002
	WM_CLOSE     = 0x0010
	WM_RBUTTONUP = 0x0205
	WM_LBUTTONDBLCLK = 0x0203

	// Menu
	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800

	// TrackPopupMenu
	TPM_RIGHTBUTTON = 0x00000002
	TPM_NONOTIFY    = 0x00000080
	TPM_RETURNCMD   = 0x00000100

	// Window style
	WS_POPUP = 0x80000000

	// Menu item IDs
	IDM_EXIT    = 1001
	IDM_STATUS  = 1002

	// Icon loading
	IDI_APPLICATION = 32512
)

type point struct {
	X int32
	Y int32
}

type notifyIconData struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uTimeout         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
}

// Tray manages a system-tray icon with right-click menu.
type Tray struct {
	hwnd     uintptr
	hIcon    uintptr
	iconID   uint32
	mu       sync.Mutex
	quit     chan struct{}
	tooltip  string
}

var (
	trayInst   *Tray
	trayOnce   sync.Once
)

// StartTray creates the system-tray icon and starts the message loop.
// Returns a channel that closes when the user requests exit (right-click → 关闭).
func StartTray() *Tray {
	t := &Tray{
		iconID: 1,
		quit:   make(chan struct{}),
	}
	trayInst = t

	// Load icon: try netshell.dll first (network icon), fallback to app icon
	t.hIcon = loadTrayIcon()

	go t.messageLoop()

	return t
}

// loadTrayIcon tries to get a network icon, falling back to a simple icon.
func loadTrayIcon() uintptr {
	// Try netshell.dll resource 0 (network icon on most Windows versions)
	var large, small uintptr
	ret, _, _ := procExtractIconEx.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("netshell.dll"))),
		0, // resource index
		uintptr(unsafe.Pointer(&large)),
		uintptr(unsafe.Pointer(&small)),
		1,
	)
	if ret != ^uintptr(0) && ret > 0 && small != 0 {
		if large != 0 {
			procDestroyIcon.Call(large)
		}
		return small
	}

	// Fallback: standard application icon
	icon, _, _ := procLoadIcon.Call(0, uintptr(IDI_APPLICATION))
	return icon
}

// messageLoop creates a hidden window, adds tray icon, and runs the message pump.
func (t *Tray) messageLoop() {
	instance, _, _ := procGetModuleHandle.Call(0)

	// Register window class for message-only window
	className, _ := syscall.UTF16PtrFromString("SubMillTray")

	wndProc := syscall.NewCallback(t.wndProc)

	wc := wndClassEx{
		Size:     uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:  wndProc,
		Instance: instance,
		ClassName: className,
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	// Create a message-only hidden window (HWND_MESSAGE = -3)
	hwnd, _, _ := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(""))),
		WS_POPUP,
		0, 0, 0, 0,
		^uintptr(2), // HWND_MESSAGE
		0, instance, 0,
	)
	t.hwnd = hwnd

	// Add tray icon
	t.addTrayIcon()

	// Message loop
	var m msg
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if ret == 0 {
			break
		}
		procTranslateMsg.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMsg.Call(uintptr(unsafe.Pointer(&m)))
	}

	// Cleanup
	t.removeTrayIcon()
	if t.hIcon != 0 {
		procDestroyIcon.Call(t.hIcon)
	}
	procDestroyWindow.Call(hwnd)
	close(t.quit)
}

// wndProc handles window messages for the tray's hidden window.
func (t *Tray) wndProc(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_TRAYICON:
		switch lparam {
		case WM_RBUTTONUP:
			procPostQuitMessage.Call(0) // right-click: exit directly
		case WM_LBUTTONDBLCLK:
			// Double-click: show status balloon
			t.showBalloon("SubMill", t.tooltip)
		}
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wparam, lparam)
	return ret
}

// addTrayIcon adds the icon to the system tray.
func (t *Tray) addTrayIcon() {
	tip, _ := syscall.UTF16FromString("SubMill - 初始化中...")
	nid := notifyIconData{
		cbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:             t.hwnd,
		uID:              t.iconID,
		uFlags:           NIF_MESSAGE | NIF_ICON | NIF_TIP,
		uCallbackMessage: WM_TRAYICON,
		hIcon:            t.hIcon,
	}
	copy(nid.szTip[:], tip)
	procShellNotifyIcon.Call(uintptr(NIM_ADD), uintptr(unsafe.Pointer(&nid)))
}

// removeTrayIcon removes the icon from the system tray.
func (t *Tray) removeTrayIcon() {
	nid := notifyIconData{
		cbSize: uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:   t.hwnd,
		uID:    t.iconID,
	}
	procShellNotifyIcon.Call(uintptr(NIM_DELETE), uintptr(unsafe.Pointer(&nid)))
}

// SetTooltip updates the tray icon tooltip text. Safe to call from any goroutine.
func (t *Tray) SetTooltip(text string) {
	t.mu.Lock()
	t.tooltip = text
	t.mu.Unlock()

	tip, _ := syscall.UTF16FromString(text)
	nid := notifyIconData{
		cbSize: uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:   t.hwnd,
		uID:    t.iconID,
		uFlags: NIF_TIP,
	}
	copy(nid.szTip[:], tip)
	procShellNotifyIcon.Call(uintptr(NIM_MODIFY), uintptr(unsafe.Pointer(&nid)))
}

// showContextMenu displays the right-click popup menu.
func (t *Tray) showContextMenu() {
	menu, _, _ := procCreatePopupMenu.Call()

	// Status item
	statusText := "SubMill - 运行中"
	t.mu.Lock()
	if t.tooltip != "" {
		statusText = t.tooltip
	}
	t.mu.Unlock()

	procAppendMenu.Call(menu, MF_STRING, IDM_STATUS, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(statusText))))
	procAppendMenu.Call(menu, MF_SEPARATOR, 0, 0)
	procAppendMenu.Call(menu, MF_STRING, IDM_EXIT, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("关闭(&X)"))))

	// Get cursor position
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	// Must set foreground window for the menu to work correctly
	procSetForegroundWindow.Call(t.hwnd)

	cmd, _, _ := procTrackPopupMenu.Call(menu, TPM_RIGHTBUTTON|TPM_RETURNCMD,
		uintptr(pt.X), uintptr(pt.Y), 0, t.hwnd, 0)

	procDestroyMenu.Call(menu)

	if cmd == IDM_EXIT {
		procPostQuitMessage.Call(0)
	}
}

// showBalloon shows a balloon tooltip notification.
func (t *Tray) showBalloon(title, text string) {
	titleUTF16, _ := syscall.UTF16FromString(title)
	textUTF16, _ := syscall.UTF16FromString(text)
	nid := notifyIconData{
		cbSize:      uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:        t.hwnd,
		uID:         t.iconID,
		uFlags:      NIF_INFO,
		uTimeout:    3000,
		dwInfoFlags: 1, // NIIF_INFO
	}
	copy(nid.szInfo[:], textUTF16)
	copy(nid.szInfoTitle[:], titleUTF16)
	procShellNotifyIcon.Call(uintptr(NIM_MODIFY), uintptr(unsafe.Pointer(&nid)))
}

// Quit returns a channel that closes when the user requests exit.
func (t *Tray) Quit() <-chan struct{} {
	return t.quit
}

// Shutdown cleanly removes the tray icon and stops the message loop.
func (t *Tray) Shutdown() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.hwnd != 0 {
		procPostQuitMessage.Call(0)
	}
}

// ---- Global helpers used by lifecycle_windows.go ----

var globalTray *Tray

// InitTray starts the system tray and stores it globally.
func InitTray() {
	globalTray = StartTray()
}

// UpdateTrayTooltip pushes a tooltip update to the tray icon.
func UpdateTrayTooltip(text string) {
	if globalTray != nil {
		globalTray.SetTooltip(text)
	}
}

// UpdateProgress pushes progress to the tray tooltip (replaces progress bar).
func UpdateProgress(percent int, status string) {
	tip := status
	if globalTray != nil {
		globalTray.SetTooltip(tip)
	}
}

// SetProgressUpdater is a no-op stub for backward compat.
func SetProgressUpdater(_ func(percent int, status string)) {}

// ClearProgressUpdater is a no-op stub for backward compat.
func ClearProgressUpdater() {}

// CloseProgress is a no-op stub (no window to close).
func CloseProgress() {}

// ShowTrayBalloon shows a balloon notification.
func ShowTrayBalloon(title, text string) {
	if globalTray != nil {
		globalTray.showBalloon(title, text)
	}
}

// WaitTrayQuit returns a channel that closes when user exits via tray.
func WaitTrayQuit() <-chan struct{} {
	if globalTray != nil {
		return globalTray.Quit()
	}
	ch := make(chan struct{})
	close(ch)
	return ch
}

// StopTray cleanly shuts down the tray.
func StopTray() {
	if globalTray != nil {
		globalTray.Shutdown()
		globalTray = nil
	}
}