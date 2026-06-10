//go:build windows

package worker

import (
	"syscall"
	"unsafe"
)

// ShowReadyDialog pops up a message box indicating the proxy is ready.
func ShowReadyDialog() {
	user32 := syscall.NewLazyDLL("user32.dll")
	msgBox := user32.NewProc("MessageBoxW")
	msgBox.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("SubMill\n\n配置完成，可以正常访问！"))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("SubMill"))),
		uintptr(0x40), // MB_ICONINFORMATION
	)
}