//go:build !windows

package app

func (app *App) onStartup()     {}
func (app *App) onMihomoReady() {}
func (app *App) onShutdown()    {}