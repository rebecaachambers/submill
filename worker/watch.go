// Package worker provides an embedded file watcher that bridges SubMill output
// to Mihomo nodes directory. Works on all platforms (Linux, macOS, Windows).
package worker

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors SubMill's output file and syncs to Mihomo's nodes directory.
type Watcher struct {
	source    string    // SubMill output: output/all.yaml
	targetDir string    // Mihomo nodes:   mihomo/nodes/
	mu        sync.Mutex
	done      chan struct{}
}

// New creates a file watcher that syncs SubMill output to Mihomo nodes.
func New(projectDir string) *Watcher {
	return &Watcher{
		source:    filepath.Join(projectDir, "output", "all.yaml"),
		targetDir: filepath.Join(projectDir, "mihomo", "nodes"),
		done:      make(chan struct{}),
	}
}

// Start begins watching and performs an initial sync.
// Returns immediately; runs in background goroutine.
func (w *Watcher) Start() error {
	os.MkdirAll(w.targetDir, 0755)

	// Initial sync
	w.sync()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("worker: create watcher failed: %w", err)
	}

	// Watch the output directory (not just the file, handle renames on Windows too)
	outputDir := filepath.Dir(w.source)
	if err := watcher.Add(outputDir); err != nil {
		watcher.Close()
		return fmt.Errorf("worker: watch dir failed: %w", err)
	}

	go w.loop(watcher)
	return nil
}

// Stop shuts down the watcher.
func (w *Watcher) Stop() {
	close(w.done)
}

func (w *Watcher) loop(watcher *fsnotify.Watcher) {
	defer watcher.Close()

	// Debounce timer to avoid multiple rapid syncs
	var debounce *time.Timer

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// React to writes and renames targeting all.yaml
			if filepath.Base(event.Name) == "all.yaml" &&
				(event.Op&fsnotify.Write != 0 || event.Op&fsnotify.Create != 0) {
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(2*time.Second, w.sync)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Warn("worker: watch error", "error", err)

		case <-w.done:
			if debounce != nil {
				debounce.Stop()
			}
			return
		}
	}
}

// sync copies SubMill's output/all.yaml to mihomo/nodes/all.yaml with basic validation.
func (w *Watcher) sync() {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := os.ReadFile(w.source)
	if err != nil {
		// File doesn't exist yet - normal during startup
		return
	}

	if len(data) == 0 {
		return
	}

	target := filepath.Join(w.targetDir, "all.yaml")
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, data, 0644); err != nil {
		slog.Error("worker: write temp failed", "error", err)
		return
	}

	if err := os.Rename(tmp, target); err != nil {
		// Fallback: some filesystems don't support Rename across devices
		if copyErr := copyFile(tmp, target); copyErr != nil {
			slog.Error("worker: copy failed", "error", copyErr)
		}
		os.Remove(tmp)
	}

	slog.Info("worker: synced nodes", "size", len(data))
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}