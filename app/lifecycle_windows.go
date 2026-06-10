//go:build windows

package app

import (
	"log/slog"
	"time"

	"github.com/rebecaachambers/submill/worker"
)

// onStartup is called at application start (Windows only).
// Kills leftover processes from previous runs.
func (app *App) onStartup() {
	slog.Info("=== Windows startup: cleaning residue ===")
	worker.KillResidue()
}

// onMihomoReady is called after Mihomo has been launched and is listening.
// Sets system proxy now that the proxy server is actually ready.
func (app *App) onMihomoReady() {
	time.Sleep(2 * time.Second) // give mihomo time to bind ports
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