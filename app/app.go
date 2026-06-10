package app

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/rebecaachambers/submill/app/monitor"
	"github.com/rebecaachambers/submill/assets"
	"github.com/rebecaachambers/submill/config"
	"github.com/rebecaachambers/submill/check"
	"github.com/rebecaachambers/submill/save"
	"github.com/rebecaachambers/submill/utils"
	"github.com/rebecaachambers/submill/worker"
	"github.com/fsnotify/fsnotify"
	"github.com/robfig/cron/v3"
)

type App struct {
	configPath string
	interval   int
	watcher    *fsnotify.Watcher
	checkChan  chan struct{}
	checking   atomic.Bool
	ticker     *time.Ticker
	done       chan struct{}
	cron       *cron.Cron
	version    string
	nodeWorker *worker.Watcher
	mihomo     *worker.Launcher
}

// projectDir returns the executable's directory.
func projectDir() string {
	ex, _ := os.Executable()
	return utils.GetExecutablePath()
}

func New(version string) *App {
	configPath := flag.String("f", "", "config file path")
	flag.Parse()
	return &App{
		configPath: *configPath,
		checkChan:  make(chan struct{}),
		done:       make(chan struct{}),
		version:    version,
	}
}

func (app *App) Initialize() error {
	if err := app.initConfigPath(); err != nil {
		return fmt.Errorf("init config path: %w", err)
	}
	if err := app.loadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := initResolver(); err != nil {
		return fmt.Errorf("init DNS: %w", err)
	}
	if err := config.WriteMihomoConfig(); err != nil {
		slog.Warn(fmt.Sprintf("Failed to write Mihomo config: %v", err))
	}
	if err := app.initConfigWatcher(); err != nil {
		return fmt.Errorf("init config watcher: %w", err)
	}
	if config.GlobalConfig.Proxy != "" {
		os.Setenv("HTTP_PROXY", config.GlobalConfig.Proxy)
		os.Setenv("HTTPS_PROXY", config.GlobalConfig.Proxy)
	}
	app.interval = func() int {
		if config.GlobalConfig.CheckInterval <= 0 { return 1 }
		return config.GlobalConfig.CheckInterval
	}()
	if config.GlobalConfig.ListenPort != "" {
		if err := app.initHttpServer(); err != nil {
			return fmt.Errorf("init HTTP server: %w", err)
		}
	}
	if config.GlobalConfig.SubStorePort != "" {
		if runtime.GOOS == "linux" && runtime.GOARCH == "386" {
			slog.Warn("node does not support linux 386, skipping sub-store")
		}
		go assets.RunSubStoreService()
		time.Sleep(500 * time.Millisecond)
	}
	monitor.StartMemoryMonitor()
	app.nodeWorker = worker.New(projectDir())
	app.mihomo = worker.NewLauncher(projectDir())
	slog.Info("Worker + Mihomo launcher ready")
	utils.SetupSignalHandler(check.RequestCancel)
	return nil
}

func (app *App) Run() {
	defer func() {
		app.watcher.Close()
		if app.ticker != nil { app.ticker.Stop() }
		if app.cron != nil { app.cron.Stop() }
		app.onShutdown()
	}()

	app.onStartup()
	app.setTimer()

	if config.GlobalConfig.CronExpression != "" {
		slog.Warn("Using cron expression, no immediate check on startup")
	} else {
		app.triggerCheck()
	}

	if app.nodeWorker != nil {
		if err := app.nodeWorker.Start(); err != nil {
			slog.Warn(fmt.Sprintf("Worker start failed: %v", err))
		} else {
			slog.Info("Worker started", "source", "output/all.yaml", "target", "mihomo/nodes/")
		}
	}

	go func() {
		app.nodeWorker.WaitFirstSync()
		time.Sleep(1 * time.Second)
		app.mihomo.StartOnce()
	}()

	for range app.checkChan {
		go app.triggerCheck()
	}
}

func (app *App) setTimer() {
	if app.ticker != nil {
		close(app.done)
		app.done = make(chan struct{})
		app.ticker.Stop()
		app.ticker = nil
	}
	if app.cron != nil {
		app.cron.Stop()
		app.cron = nil
	}
	if config.GlobalConfig.CronExpression != "" {
		slog.Info(fmt.Sprintf("Using cron: %s", config.GlobalConfig.CronExpression))
		app.cron = cron.New()
		_, err := app.cron.AddFunc(config.GlobalConfig.CronExpression, func() {
			app.triggerCheck()
		})
		if err != nil {
			slog.Error(fmt.Sprintf("cron parse failed: %v, using interval", err))
			app.useIntervalTimer()
		} else {
			app.cron.Start()
		}
	} else {
		app.useIntervalTimer()
	}
}

func (app *App) useIntervalTimer() {
	app.ticker = time.NewTicker(time.Duration(app.interval) * time.Minute)
	done := app.done
	go func() {
		for {
			select {
			case <-app.ticker.C:
				app.triggerCheck()
			case <-done:
				return
			}
		}
	}()
}

func (app *App) TriggerCheck() {
	select {
	case app.checkChan <- struct{}{}:
		slog.Info("Manual check triggered")
	default:
		slog.Warn("Check already in progress, skipping")
	}
}

func (app *App) triggerCheck() {
	if !app.checking.CompareAndSwap(false, true) {
		slog.Warn("Check already in progress, skipping")
		return
	}
	defer app.checking.Store(false)

	if err := app.checkProxies(); err != nil {
		slog.Error(fmt.Sprintf("Proxy check failed: %v", err))
		os.Exit(1)
	}
	if app.ticker != nil {
		app.ticker.Reset(time.Duration(app.interval) * time.Minute)
		nextCheck := time.Now().Add(time.Duration(app.interval) * time.Minute)
		slog.Info(fmt.Sprintf("Next check: %s", nextCheck.Format("2006-01-02 15:04:05")))
	} else if app.cron != nil {
		entries := app.cron.Entries()
		if len(entries) > 0 {
			slog.Info(fmt.Sprintf("Next check: %s", entries[0].Next.Format("2006-01-02 15:04:05")))
		}
	}
	debug.FreeOSMemory()
}

func (app *App) checkProxies() error {
	slog.Info("Preparing proxy check", "show_progress", config.GlobalConfig.PrintProgress)
	if config.GlobalConfig.KeepDays > 0 {
		if hp := save.LoadHistoryProxies(); len(hp) > 0 {
			config.GlobalProxies = append(config.GlobalProxies, hp...)
		}
	}
	results, err := check.Check()
	if err != nil {
		return fmt.Errorf("proxy check failed: %w", err)
	}
	slog.Info("Check complete")
	save.SaveConfig(results)
	utils.SendNotify(len(results))
	utils.UpdateSubs()
	utils.ExecuteCallback(len(results))
	return nil
}