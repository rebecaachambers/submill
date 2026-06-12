//go:build !windows

package worker

import "syscall"

func hideWindowAttr() *syscall.SysProcAttr {
	return nil
}

func EnsureNodesJunction(_ string) {}