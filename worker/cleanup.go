package worker

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// KillResidue finds and kills leftover mihomo/submill processes from previous runs.
func KillResidue() {
	exe, _ := os.Executable()
	selfName := filepath.Base(exe)

	targets := []string{"mihomo", "mihomo.exe", "submill", "submill.exe"}
	for _, name := range targets {
		if strings.EqualFold(name, selfName) {
			continue // don't kill self
		}
		killByName(name)
	}
}

func killByName(name string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/IM", name, "/T")
	default:
		cmd = exec.Command("pkill", "-f", name)
	}
	if out, err := cmd.CombinedOutput(); err == nil {
		slog.Info("Cleaned up residue process", "name", name)
	} else if len(out) > 0 {
		slog.Debug(fmt.Sprintf("kill %s: %s", name, strings.TrimSpace(string(out))))
	}
}

// CleanFiles removes runtime cache and output files to leave no trace.
func CleanFiles(projectDir string) {
	slog.Info("Cleaning runtime files...")

	patterns := []string{
		filepath.Join(projectDir, "mihomo.log"),
		filepath.Join(projectDir, "mihomo", "cache.db"),
		filepath.Join(projectDir, "mihomo", "geoip.metadb"),
		filepath.Join(projectDir, "mihomo", "nodes", "*"),
		filepath.Join(projectDir, "config", "cache.db"),
		filepath.Join(projectDir, "config", "geoip.metadb"),
		filepath.Join(projectDir, "output", "*"),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			fi, err := os.Stat(m)
			if err != nil {
				continue
			}
			// Skip directories
			if fi.IsDir() {
				continue
			}
			// Skip config.yaml and submill.yaml
			base := filepath.Base(m)
			if base == "config.yaml" || base == "submill.yaml" || base == "config.example.yaml" {
				continue
			}
			if err := os.Remove(m); err == nil {
				slog.Debug("Removed", "file", m)
			}
		}
	}
}