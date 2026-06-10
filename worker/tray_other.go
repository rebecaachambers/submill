//go:build !windows

package worker

// InitTray is a no-op on non-Windows platforms.
func InitTray() {}

// UpdateTrayTooltip is a no-op on non-Windows platforms.
func UpdateTrayTooltip(_ string) {}

// UpdateProgress is a no-op on non-Windows platforms.
func UpdateProgress(_ int, _ string) {}

// SetProgressUpdater is a no-op on non-Windows platforms.
func SetProgressUpdater(_ func(percent int, status string)) {}

// ClearProgressUpdater is a no-op on non-Windows platforms.
func ClearProgressUpdater() {}

// CloseProgress is a no-op on non-Windows platforms.
func CloseProgress() {}

// ShowTrayBalloon is a no-op on non-Windows platforms.
func ShowTrayBalloon(_, _ string) {}

// WaitTrayQuit returns a channel that never closes (Linux server runs forever).
func WaitTrayQuit() <-chan struct{} {
	return make(chan struct{})
}

// StopTray is a no-op on non-Windows platforms.
func StopTray() {}