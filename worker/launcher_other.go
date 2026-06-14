//go:build !windows

package worker

import (
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
)

func hideWindowAttr() *syscall.SysProcAttr {
	return nil
}

func EnsureNodesJunction(_ string) {}

// ensureNodesSymlink creates config/nodes -> mihomo/nodes symlink
// so mihomo can find nodes through config/nodes/all.yaml
func ensureNodesSymlink(projectDir string) {
	linkPath := filepath.Join(projectDir, "config", "nodes")
	targetPath := filepath.Join(projectDir, "mihomo", "nodes")

	// If symlink already exists, skip
	if fi, err := os.Lstat(linkPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			return
		}
		// Exists but not a symlink - remove it
		os.RemoveAll(linkPath)
	}

	os.MkdirAll(filepath.Dir(linkPath), 0755)
	os.MkdirAll(targetPath, 0755)

	if err := os.Symlink(targetPath, linkPath); err != nil {
		slog.Warn("Failed to create nodes symlink, mihomo may not find nodes",
			"link", linkPath, "target", targetPath, "error", err)
	} else {
		slog.Info("Nodes symlink created", "link", linkPath, "target", targetPath)
	}
}
