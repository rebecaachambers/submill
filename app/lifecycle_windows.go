//go:build windows

package app

import (
	"log/slog"
	"time"

	"github.com/rebecaachambers/submill/check"
	"github.com/rebecaachambers/submill/worker"
)

func (app *App) onStartup() {
	slog.Info("=== Windows startup: cleaning residue ===")
	worker.KillResidue()

	// Start system-tray icon (network-style, right-click → close)
	worker.InitTray()

	// Wire check progress to tray tooltip updates
	check.ProgressCallback = func(phase string, percent int, status string) {
		worker.UpdateProgress(percent, status)
	}
}

func (app *App) beforeCheck() {
	// Update tray tooltip to show detection started
	worker.UpdateTrayTooltip("SubMill - 节点检测中...")
}

func (app *App) afterCheck() {
	// No window to close; tray stays
}

func (app *App) onMihomoReady() {
	time.Sleep(2 * time.Second)
	worker.SetSystemProxy()
	worker.UpdateTrayTooltip("SubMill - 代理已就绪 (127.0.0.1:20171)")
}

func (app *App) onShutdown() {
	slog.Info("=== Windows shutdown: cleaning up ===")
	worker.ClearSystemProxy()
	if app.mihomo != nil {
		app.mihomo.Stop()
	}
	worker.StopTray()
	worker.KillResidue()
	worker.CleanFiles(projectDir())
}