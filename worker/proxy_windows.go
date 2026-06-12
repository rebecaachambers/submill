//go build windows

package worker

import (
	"log/slog"
	"os/exec"
	"syscall"
)

func SetSystemProxy() {
	hide := &syscall.SysProcAttr{HideWindow: true}

	cmd := exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyServer", "/t", "REG_SZ", "/d", "127.0.0.1:20171", "/f",
	)
	cmd.SysProcAttr = hide
	cmd.Run()

	cmd2 := exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f",
	)
	cmd2.SysProcAttr = hide
	cmd2.Run()

	slog.Info("System proxy set: 127.0.0.1:20171")
}

func ClearSystemProxy() {
	hide := &syscall.SysProcAttr{HideWindow: true}

	cmd := exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f",
	)
	cmd.SysProcAttr = hide
	cmd.Run()

	cmd2 := exec.Command("reg", "delete",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyServer", "/f",
	)
	cmd2.SysProcAttr = hide
	cmd2.Run()

	slog.Info("System proxy cleared")
}
