package worker

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	source    string
	targetDir string
	mu        sync.Mutex
	done      chan struct{}
	firstSync chan struct{}
	synced    bool
}

func New(projectDir string) *Watcher {
	return &Watcher{
		source:    filepath.Join(projectDir, "output", "all.yaml"),
		targetDir: filepath.Join(projectDir, "mihomo", "nodes"),
		done:      make(chan struct{}),
		firstSync: make(chan struct{}),
	}
}

func (w *Watcher) Start() error {
	os.MkdirAll(w.targetDir, 0755)
	w.sync()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	outputDir := filepath.Dir(w.source)
	if err := watcher.Add(outputDir); err != nil {
		watcher.Close()
		return err
	}
	go w.loop(watcher)
	return nil
}

func (w *Watcher) Stop() {
	close(w.done)
}

func (w *Watcher) WaitFirstSync() {
	<-w.firstSync
}

func (w *Watcher) loop(watcher *fsnotify.Watcher) {
	defer watcher.Close()
	var debounce *time.Timer
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok { return }
			if filepath.Base(event.Name) == "all.yaml" &&
				(event.Op&fsnotify.Write != 0 || event.Op&fsnotify.Create != 0) {
				if debounce != nil { debounce.Stop() }
				debounce = time.AfterFunc(2*time.Second, w.sync)
			}
		case err, ok := <-watcher.Errors:
			if !ok { return }
			slog.Warn("worker: watch error", "error", err)
		case <-w.done:
			if debounce != nil { debounce.Stop() }
			return
		}
	}
}

func (w *Watcher) sync() {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := os.ReadFile(w.source)
	if err != nil { return }
	if len(data) == 0 { return }

	target := filepath.Join(w.targetDir, "all.yaml")
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		slog.Error("worker: write temp failed", "error", err)
		return
	}
	if err := os.Rename(tmp, target); err != nil {
		data2, _ := os.ReadFile(tmp)
		os.WriteFile(target, data2, 0644)
		os.Remove(tmp)
	}
	slog.Info("worker: synced nodes", "size", len(data))
	if !w.synced {
		w.synced = true
		close(w.firstSync)
	}
}