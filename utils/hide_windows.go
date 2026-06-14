//go:build windows

package utils

import "syscall"

func hideWindowSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
