//go:build windows

package app

import (
	"log/slog"
	"time"

	"github.com/rebecaachambers/submill/check"
	"github.com/rebecaachambers/submill/worker"
)

func (app *App) onStartup() {
	slog.Info("=== Windows startup ===")
	worker.KillResidue()
	worker.ClearSystemProxy()
	worker.InitTray()

	// Show progress window immediately — stays open through entire startup
	worker.ShowProgress("SubMill - 正在初始化...")

	// Wire progress callback to update bar + status
	check.ProgressCallback = func(phase string, percent int, status string) {
		worker.SetProgress(percent, status)
	}
}

func (app *App) beforeCheck() {
	worker.SetProgress(0, "正在拉取订阅、检测节点...")
}

func (app *App) afterCheck() {
	// Don't close the window yet — Mihomo still starting
	worker.SetProgress(95, "节点检测完成，正在启动 Mihomo...")
}

func (app *App) onMihomoReady() {
	slog.Info("Waiting for Mihomo to be ready...")
	for i := 0; i < 60; i++ {
		if worker.IsPortOpen("127.0.0.1", 20171) {
			slog.Info("Mihomo is ready, setting proxy")
			worker.SetSystemProxy()
			// Close progress, show completion
			worker.CloseProgress()
			worker.ShowReadyDialog()
			return
		}
		time.Sleep(1 * time.Second)
	}
	slog.Warn("Mihomo did not start within 60s")
	worker.SetSystemProxy()
	worker.CloseProgress()
	worker.ShowReadyDialog()
}

func (app *App) onShutdown() {
	slog.Info("=== Windows shutdown: cleaning up ===")
	worker.ClearSystemProxy()
	if app.mihomo != nil {
		app.mihomo.Stop()
	}
	worker.CloseProgress()
	worker.StopTray()
	worker.KillResidue()
	worker.CleanFiles(projectDir())
}