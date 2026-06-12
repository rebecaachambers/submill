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

	// Clear any stale system proxy from previous crashed session
	worker.ClearSystemProxy()

	// Start system-tray icon (network-style, right-click → close)
	worker.InitTray()

	// Wire check progress to progress window + tray tooltip
	check.ProgressCallback = func(phase string, percent int, status string) {
		worker.SetProgress(percent, status)
		worker.UpdateProgress(percent, status)
	}
}

func (app *App) beforeCheck() {
	// Show progress bar window before each check
	worker.ShowProgress("SubMill - 节点检测中...")
}

func (app *App) afterCheck() {
	// Close progress window after check completes
	worker.CloseProgress()
}

func (app *App) onMihomoReady() {
	// Poll until Mihomo is actually listening on 20171
	slog.Info("Waiting for Mihomo to be ready...")
	worker.UpdateTrayTooltip("SubMill - 等待 Mihomo 启动...")
	for i := 0; i < 60; i++ {
		if worker.IsPortOpen("127.0.0.1", 20171) {
			slog.Info("Mihomo is ready, setting proxy")
			worker.SetSystemProxy()
			worker.UpdateTrayTooltip("SubMill - 代理已就绪 (127.0.0.1:20171)")
			// Pop up completion dialog
			worker.ShowReadyDialog()
			return
		}
		time.Sleep(1 * time.Second)
	}
	slog.Warn("Mihomo did not start within 60s, setting proxy anyway")
	worker.SetSystemProxy()
}

func (app *App) onShutdown() {
	slog.Info("=== Windows shutdown: cleaning up ===")
	worker.ClearSystemProxy()
	if app.mihomo != nil {
		app.mihomo.Stop()
	}
	worker.StopTray()
	worker.CloseProgress()
	worker.KillResidue()
	worker.CleanFiles(projectDir())
}