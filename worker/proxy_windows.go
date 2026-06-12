//go:build windows

package worker

import (
	"log/slog"
	"os/exec"
)

// SetSystemProxy enables Windows system-wide proxy on 127.0.0.1:20171.
func SetSystemProxy() {
	// Set proxy server
	exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyServer", "/t", "REG_SZ", "/d", "127.0.0.1:20171", "/f",
	).Run()

	// Enable proxy
	exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f",
	).Run()

	slog.Info("System proxy set: 127.0.0.1:20171")
}

// ClearSystemProxy disables Windows system-wide proxy.
func ClearSystemProxy() {
	// Disable proxy
	exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f",
	).Run()

	// Clear proxy server
	exec.Command("reg", "delete",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyServer", "/f",
	).Run()

	slog.Info("System proxy cleared")
}