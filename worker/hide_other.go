//go:build !windows

package worker

import "syscall"

func hideWindowSysProcAttr() *syscall.SysProcAttr {
	return nil
}
