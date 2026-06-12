//go:build windows

package worker

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	comctl32              = syscall.NewLazyDLL("comctl32.dll")
	procInitCommonCtrls2  = comctl32.NewProc("InitCommonControlsEx")
	procSendMessage       = user32.NewProc("SendMessageW")
	procSetWindowText     = user32.NewProc("SetWindowTextW")
	procUpdateWindow      = user32.NewProc("UpdateWindow")

	progressHWND    uintptr
	progressBarHWND uintptr
	statusHWND      uintptr
	progressDone    chan struct{}
	progressMu      sync.Mutex
)

const (
	WS_EX_TOPMOST      = 0x00000008
	WS_OVERLAPPED      = 0x00000000
	WS_CAPTION         = 0x00C00000
	WS_SYSMENU         = 0x00080000
	WS_MINIMIZEBOX     = 0x00020000
	WS_VISIBLE         = 0x10000000
	WS_CHILD           = 0x40000000
	CW_USEDEFAULT      = 0x80000000
	COLOR_WINDOW       = 5
	PBM_SETPOS         = 0x0402 + 2
	PBM_SETRANGE32     = 0x0402 + 6
	ICC_PROGRESS_CLASS = 0x00000020
	WM_NCLBUTTONDOWN   = 0x00A1
	WM_SYSCOMMAND      = 0x0112
	SC_MOVE            = 0xF010
	SC_SIZE            = 0xF020

	SS_LEFT       = 0x00000000
	SS_CENTER     = 0x00000001
	SS_CENTERIMAGE = 0x00000200

	WIN_W = 500
	WIN_H = 185
)

type initCommonCtrls struct {
	Size  uint32
	Flags uint32
}

// ShowProgress opens a native Windows progress bar window (topmost, non-draggable).
func ShowProgress(title string) (done chan struct{}) {
	progressDone = make(chan struct{})
	ready := make(chan struct{})

	go func() {
		icc := initCommonCtrls{Size: 8, Flags: ICC_PROGRESS_CLASS}
		procInitCommonCtrls2.Call(uintptr(unsafe.Pointer(&icc)))

		instance, _, _ := procGetModuleHandle.Call(0)

		className, _ := syscall.UTF16PtrFromString("SubMillProgress")
		wndProc := syscall.NewCallback(progressWndProc)

		wc := wndClassEx{
			Size:       uint32(unsafe.Sizeof(wndClassEx{})),
			WndProc:    wndProc,
			Instance:   instance,
			Background: COLOR_WINDOW + 1,
			ClassName:  className,
		}
		procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

		titlePtr, _ := syscall.UTF16PtrFromString(title)

		// Parent window: topmost + can't drag
		hwnd, _, _ := procCreateWindowEx.Call(
			uintptr(WS_EX_TOPMOST),
			uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(titlePtr)),
			WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU|WS_MINIMIZEBOX|WS_VISIBLE,
			CW_USEDEFAULT, CW_USEDEFAULT, WIN_W, WIN_H,
			0, 0, instance, 0,
		)

		progressMu.Lock()
		progressHWND = hwnd
		progressMu.Unlock()

		// Status text
		staticClass, _ := syscall.UTF16PtrFromString("static")
		statusPtr, _ := syscall.UTF16PtrFromString("Initializing...")
		statusHWND, _, _ = procCreateWindowEx.Call(
			0, uintptr(unsafe.Pointer(staticClass)), uintptr(unsafe.Pointer(statusPtr)),
			WS_CHILD|WS_VISIBLE|SS_CENTER,
			15, 12, WIN_W-30, 55,
			hwnd, 0, instance, 0,
		)

		// Progress bar
		barClass, _ := syscall.UTF16PtrFromString("msctls_progress32")
		barHWND, _, _ := procCreateWindowEx.Call(
			0, uintptr(unsafe.Pointer(barClass)), 0,
			WS_CHILD|WS_VISIBLE|1,
			20, 75, WIN_W-40, 28,
			hwnd, 0, instance, 0,
		)
		progressBarHWND = barHWND
		procSendMessage.Call(barHWND, PBM_SETRANGE32, 0, uintptr(100))

		procUpdateWindow.Call(hwnd)
		close(ready)

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

		close(progressDone)
	}()

	<-ready
	return progressDone
}

func progressWndProc(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_DESTROY, WM_CLOSE:
		return 0
	case WM_NCLBUTTONDOWN:
		// Block title-bar drag to prevent DefWindowProc modal-loop deadlock
		return 0
	case WM_SYSCOMMAND:
		cmd := wparam & 0xFFF0
		if cmd == SC_MOVE || cmd == SC_SIZE {
			return 0
		}
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wparam, lparam)
	return ret
}

// SetProgress updates the progress bar and status text.
func SetProgress(percent int, status string) {
	progressMu.Lock()
	defer progressMu.Unlock()
	if progressBarHWND != 0 {
		procSendMessage.Call(progressBarHWND, PBM_SETPOS, uintptr(percent), 0)
	}
	if statusHWND != 0 {
		full := status
		if percent > 0 {
			full = status + " (" + itoa(percent) + "%)"
		}
		text, _ := syscall.UTF16PtrFromString(full)
		procSetWindowText.Call(statusHWND, uintptr(unsafe.Pointer(text)))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// CloseProgress destroys the progress window.
func CloseProgress() {
	progressMu.Lock()
	defer progressMu.Unlock()
	if progressHWND != 0 {
		procDestroyWindow.Call(progressHWND)
		progressHWND = 0
		progressBarHWND = 0
		statusHWND = 0
	}
}