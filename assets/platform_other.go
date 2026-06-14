//go:build !windows

package assets

import "os/exec"

func setPlatformSysProcAttr(cmd *exec.Cmd) {
	// no-op on non-Windows
}
