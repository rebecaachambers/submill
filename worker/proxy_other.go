//go:build !windows

package worker

import "log/slog"

func SetSystemProxy() {
	slog.Debug("system proxy: set (not implemented for this OS)")
}

func ClearSystemProxy() {
	slog.Debug("system proxy: clear (not implemented for this OS)")
}