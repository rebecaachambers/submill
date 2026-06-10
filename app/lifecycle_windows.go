//go:build windows

package app

import (
	"log/slog"

	"github.com/rebecaachambers/submill/worker"
)

// onStartup is called at application start (Windows only).
// Kills leftover processes from previous runs and sets system proxy.
func (app *App) onStartup() {
	slog.Info("=== Windows startup: cleaning residue ===")
	worker.KillResidue()
	worker.SetSystemProxy()
}

// onShutdown is called when the application exits (Windows only).
// Clears system proxy, kills Mihomo, and removes runtime files.
func (app *App) onShutdown() {
	slog.Info("=== Windows shutdown: cleaning up ===")
	worker.ClearSystemProxy()
	if app.mihomo != nil {
		app.mihomo.Stop()
	}
	worker.KillResidue()
	worker.CleanFiles(projectDir())
}