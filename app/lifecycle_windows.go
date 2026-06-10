//go:build windows

package app

import (
	"log/slog"
	"time"

	"github.com/rebecaachambers/submill/worker"
)

func (app *App) onStartup() {
	slog.Info("=== Windows startup: cleaning residue ===")
	worker.KillResidue()
}

func (app *App) onMihomoReady() {
	time.Sleep(2 * time.Second)
	worker.SetSystemProxy()
	worker.ShowReadyDialog()
}

func (app *App) onShutdown() {
	slog.Info("=== Windows shutdown: cleaning up ===")
	worker.ClearSystemProxy()
	if app.mihomo != nil {
		app.mihomo.Stop()
	}
	worker.KillResidue()
	worker.CleanFiles(projectDir())
}