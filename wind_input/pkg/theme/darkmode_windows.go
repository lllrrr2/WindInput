package theme

import (
	"log/slog"
	"sync"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	personalizeKeyPath = `SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	appsUseLightTheme  = "AppsUseLightTheme"
)

// IsSystemDarkMode checks whether the system is in dark mode by reading the registry.
// Returns true if dark mode is active, false for light mode or on error.
func IsSystemDarkMode() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, personalizeKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	val, _, err := key.GetIntegerValue(appsUseLightTheme)
	if err != nil {
		return false
	}
	// 0 = dark mode, 1 = light mode
	return val == 0
}

// DarkModeWatcher monitors system dark mode changes and invokes a callback.
type DarkModeWatcher struct {
	mu       sync.Mutex
	logger   *slog.Logger
	callback func(isDark bool)
	stopCh   chan struct{}
	stopped  bool
	lastMode bool
}

// NewDarkModeWatcher creates a watcher that calls callback when system dark mode changes.
func NewDarkModeWatcher(logger *slog.Logger, callback func(isDark bool)) *DarkModeWatcher {
	return &DarkModeWatcher{
		logger:   logger,
		callback: callback,
		stopCh:   make(chan struct{}),
		lastMode: IsSystemDarkMode(),
	}
}

// Start begins watching for dark mode changes in a background goroutine.
func (w *DarkModeWatcher) Start() {
	go w.watchLoop()
}

// Stop stops the watcher.
func (w *DarkModeWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.stopped {
		w.stopped = true
		close(w.stopCh)
	}
}

func (w *DarkModeWatcher) watchLoop() {
	// Open registry key with NOTIFY permission
	var hKey windows.Handle
	keyPath, _ := windows.UTF16PtrFromString(personalizeKeyPath)
	err := windows.RegOpenKeyEx(windows.HKEY_CURRENT_USER, keyPath,
		0, windows.KEY_QUERY_VALUE|windows.KEY_NOTIFY, &hKey)
	if err != nil {
		if w.logger != nil {
			w.logger.Warn("无法打开注册表监听系统主题变化", "error", err)
		}
		return
	}
	defer windows.RegCloseKey(hKey)

	// Create event for async registry notification
	event, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		if w.logger != nil {
			w.logger.Warn("无法创建事件对象", "error", err)
		}
		return
	}
	defer windows.CloseHandle(event)

	// Create a stop event (manual reset)
	stopEvent, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		if w.logger != nil {
			w.logger.Warn("无法创建停止事件", "error", err)
		}
		return
	}
	defer windows.CloseHandle(stopEvent)

	// Signal stop event when stopCh is closed
	go func() {
		<-w.stopCh
		windows.SetEvent(stopEvent)
	}()

	handles := []windows.Handle{event, stopEvent}

	for {
		// Register for async notification on registry value change
		err := windows.RegNotifyChangeKeyValue(hKey, false,
			windows.REG_NOTIFY_CHANGE_LAST_SET, event, true)
		if err != nil {
			if w.logger != nil {
				w.logger.Warn("注册表通知注册失败", "error", err)
			}
			return
		}

		// Wait for either registry change or stop signal
		result, err := windows.WaitForMultipleObjects(handles, false, windows.INFINITE)
		if err != nil {
			if w.logger != nil {
				w.logger.Warn("等待事件失败", "error", err)
			}
			return
		}

		switch result {
		case windows.WAIT_OBJECT_0:
			// Registry changed — check if dark mode actually changed
			w.checkAndNotify()
		case windows.WAIT_OBJECT_0 + 1:
			// Stop requested
			return
		default:
			return
		}
	}
}

func (w *DarkModeWatcher) checkAndNotify() {
	currentMode := IsSystemDarkMode()
	w.mu.Lock()
	changed := currentMode != w.lastMode
	w.lastMode = currentMode
	w.mu.Unlock()

	if changed {
		if w.logger != nil {
			w.logger.Info("系统主题变化", "isDark", currentMode)
		}
		if w.callback != nil {
			w.callback(currentMode)
		}
	}
}
