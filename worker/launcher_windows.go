//go:build windows

package worker

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func hideWindowAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
}

// EnsureNodesJunction creates a directory junction from config/mihomo/nodes
// to mihomo/nodes so Mihomo can read nodes through the config directory path.
func EnsureNodesJunction(projectDir string) {
	junctionPath := filepath.Join(projectDir, "config", "mihomo", "nodes")
	targetPath := filepath.Join(projectDir, "mihomo", "nodes")
	
	// If junction already exists, skip
	if fi, err := os.Stat(junctionPath); err == nil && fi.IsDir() {
		return
	}
	// Ensure parent directories exist
	os.MkdirAll(filepath.Join(projectDir, "config", "mihomo"), 0755)
	os.MkdirAll(targetPath, 0755)
	
	cmd := exec.Command("cmd", "/c", "mklink", "/J", junctionPath, targetPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		slog.Warn("Failed to create nodes junction, Mihomo may not find nodes", "error", err)
	} else {
		slog.Info("Nodes junction created", "junction", junctionPath, "target", targetPath)
	}
}