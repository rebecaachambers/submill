package worker

import (
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Launcher manages the Mihomo subprocess lifecycle.
// On Windows, Mihomo runs hidden (no console popup).
type Launcher struct {
	binPath   string
	configDir string
	logFile   string
	cmd       *exec.Cmd
	mu        sync.Mutex
	running   bool
	started   chan struct{} // closed when Mihomo is first launched
}

// NewLauncher creates a Mihomo process launcher.
func NewLauncher(projectDir string) *Launcher {
	binName := "mihomo/mihomo"
	if runtime.GOOS == "windows" {
		binName = "mihomo.exe"
	}
	return &Launcher{
		binPath:   filepath.Join(projectDir, binName),
		configDir: filepath.Join(projectDir, "config"),
		logFile:   filepath.Join(projectDir, "mihomo.log"),
		started:   make(chan struct{}),
	}
}

// StartOnce launches Mihomo exactly once. Subsequent calls are no-ops.
// Returns true if this call actually started Mihomo.
func (l *Launcher) StartOnce() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.running {
		return false
	}

	if _, err := os.Stat(l.binPath); os.IsNotExist(err) {
		slog.Warn("mihomo binary not found, skip launch", "path", l.binPath)
		return false
	}

	nodesDir := filepath.Join(filepath.Dir(l.binPath), "mihomo", "nodes")
	os.MkdirAll(nodesDir, 0755)

	// Windows: ensure nodes junction so Mihomo can find nodes within config/
	if runtime.GOOS == "windows" {
		EnsureNodesJunction(filepath.Dir(l.binPath))
	}

	l.cmd = exec.Command(l.binPath, "-d", l.configDir)

	// Redirect Mihomo output to log file (silent, no extra window)
	logWriter, err := os.OpenFile(l.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		l.cmd.Stdout = logWriter
		l.cmd.Stderr = logWriter
	} else {
		l.cmd.Stdout = io.Discard
		l.cmd.Stderr = io.Discard
	}

	// Windows: hide console window
	if runtime.GOOS == "windows" {
		l.cmd.SysProcAttr = hideWindowAttr()
	}

	if err := l.cmd.Start(); err != nil {
		return false
	}

	l.running = true
	close(l.started)
	slog.Info("Mihomo launched silently", "pid", l.cmd.Process.Pid, "log", l.logFile)

	go func() {
		err := l.cmd.Wait()
		l.mu.Lock()
		l.running = false
		l.mu.Unlock()
		if err != nil {
			slog.Warn("Mihomo exited", "error", err)
		}
	}()

	time.Sleep(1 * time.Second)
	return true
}

// Stop gracefully terminates Mihomo.
func (l *Launcher) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.cmd == nil || !l.running {
		return
	}

	slog.Info("Stopping Mihomo...")
	if err := l.cmd.Process.Signal(os.Interrupt); err != nil {
		l.cmd.Process.Kill()
	}

	done := make(chan error, 1)
	go func() { done <- l.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		l.cmd.Process.Kill()
	}

	l.running = false
	slog.Info("Mihomo stopped")
}

// IsRunning returns whether Mihomo is currently running.
func (l *Launcher) IsRunning() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.running
}

// WaitStarted blocks until Mihomo has been launched at least once.
func (l *Launcher) WaitStarted() {
	<-l.started
}
