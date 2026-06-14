//go:build !windows

package monitor

import "syscall"

func hideWindowSysProcAttr() *syscall.SysProcAttr {
	return nil
}
