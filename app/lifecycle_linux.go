//go:build !windows

package app

// onStartup is a no-op on Linux (services managed by systemd).
func (app *App) onStartup() {}

// onShutdown is a no-op on Linux (persistent 24/7 operation).
func (app *App) onShutdown() {}