//go:build windows

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/huanfeng/wind_input/pkg/buildvariant"
	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/ui"
	"github.com/rodrigocfd/windigo/win"
	"golang.org/x/sys/windows"
)

// version 通过 ldflags 注入: -X main.version=x.y.z
var version = "dev"

var mutexName = "Local\\WindPortable" + buildvariant.Suffix() + "Launcher"

func main() {
	cfg, detectErr := detectPortableConfig()

	opts := parseCLI()

	// CLI 模式下检测失败直接报错退出
	if detectErr != nil && opts.hasAction() && !opts.UI {
		fmt.Fprintln(os.Stderr, detectErr)
		os.Exit(1)
	}

	var manager *launcherManager
	if detectErr == nil {
		manager = newLauncherManager(cfg)

		if opts.hasAction() && !opts.UI {
			if err := runCLI(manager, opts); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
	}

	// GUI 模式单例检查：已有实例运行时激活其窗口
	if !acquireSingleInstance() {
		activateExistingWindow()
		return
	}

	runtime.LockOSThread()
	runMainWindow(manager, detectErr)
}

type cliOptions struct {
	Start             bool
	Stop              bool
	Status            bool
	Settings          bool
	Userdata          bool
	UI                bool
	ElevateRegister   bool
	ElevateUnregister bool
}

func (o cliOptions) hasAction() bool {
	return o.Start || o.Stop || o.Status || o.Settings || o.Userdata ||
		o.ElevateRegister || o.ElevateUnregister
}

func parseCLI() cliOptions {
	var opts cliOptions
	flag.BoolVar(&opts.Start, "start", false, "Start wind_input.exe")
	flag.BoolVar(&opts.Stop, "stop", false, "Stop wind_input.exe")
	flag.BoolVar(&opts.Status, "status", false, "Print service status")
	flag.BoolVar(&opts.Settings, "settings", false, "Open wind_setting.exe")
	flag.BoolVar(&opts.Userdata, "userdata", false, "Open userdata directory")
	flag.BoolVar(&opts.UI, "ui", false, "Force showing the minimal UI window")
	flag.BoolVar(&opts.ElevateRegister, "elevate-register", false, "")
	flag.BoolVar(&opts.ElevateUnregister, "elevate-unregister", false, "")
	flag.Parse()
	return opts
}

func runCLI(manager *launcherManager, opts cliOptions) error {
	// 提权子命令：由非管理员实例通过 UAC 启动，执行注册/注销后退出
	if opts.ElevateRegister {
		return manager.registerInputMethodDirect()
	}
	if opts.ElevateUnregister {
		return manager.unregisterInputMethodDirect()
	}

	if opts.Start {
		if err := manager.startService(); err != nil {
			return err
		}
		fmt.Println("service started")
	}
	if opts.Stop {
		stopped, err := manager.stopService()
		if err != nil {
			return err
		}
		if stopped {
			fmt.Println("service stopped")
		} else {
			fmt.Println("service not running")
		}
	}
	if opts.Status {
		service := "stopped"
		if manager.serviceRunning() {
			service = "running"
		}
		conflict, reason := manager.installedConflict()
		if conflict {
			fmt.Printf("service=%s mode=conflict reason=%q\n", service, reason)
			return nil
		}
		fmt.Printf("service=%s\n", service)
	}
	if opts.Settings {
		if err := manager.openSettings(); err != nil {
			return err
		}
		fmt.Println("settings opened")
	}
	if opts.Userdata {
		if err := manager.openUserdataDir(); err != nil {
			return err
		}
		fmt.Println("userdata opened")
	}
	return nil
}

type launcherWindow struct {
	manager      *launcherManager
	detectErr    error // detectPortableConfig 检测失败的错误（非 nil 时禁用所有功能）
	tray         *trayIcon
	theme        *windowTheme
	ctx          context.Context
	cancel       context.CancelFunc
	fastPoll     chan struct{} // 触发短时间快速轮询
	cooldownUtil time.Time    // 启动冷却期截止时间
	wnd          *ui.Main
	title      *ui.Static
	status     *ui.Static
	detail     *ui.Static
	rootHint   *ui.Static
	btnStart   *ui.Button
	btnStop    *ui.Button
	btnSetting *ui.Button
	btnData    *ui.Button
}

func runMainWindow(manager *launcherManager, detectErr error) {
	ctx, cancel := context.WithCancel(context.Background())
	theme, err := newWindowTheme()
	if err != nil {
		fmt.Fprintln(os.Stderr, "init theme:", err)
		os.Exit(1)
	}

	windowTitle := "清风输入法便携启动器"
	if buildvariant.IsDebug() {
		windowTitle += " (Debug)"
	}
	if version != "" && version != "dev" {
		windowTitle += " v" + version
	}

	wnd := ui.NewMain(
		ui.OptsMain().
			ClassName("WindInputPortableLauncher").
			Title(windowTitle).
			Size(ui.Dpi(448, 210)).
			ClassIconId(1).
			ClassBrush(theme.bgBrush),
	)

	title := ui.NewStatic(
		wnd,
		ui.OptsStatic().
			Text("清风输入法便携启动器").
			Position(ui.Dpi(16, 14)).
			Size(ui.Dpi(220, 24)),
	)
	status := ui.NewStatic(
		wnd,
		ui.OptsStatic().
			Text("正在检查服务状态...").
			Position(ui.Dpi(16, 46)).
			Size(ui.Dpi(320, 22)),
	)
	detail := ui.NewStatic(
		wnd,
		ui.OptsStatic().
			Text("准备就绪").
			Position(ui.Dpi(16, 74)).
			Size(ui.Dpi(404, 20)),
	)
	rootHintText := ""
	if manager != nil {
		rootHintText = "目录: " + compactPath(manager.cfg.RootDir)
	}
	rootHint := ui.NewStatic(
		wnd,
		ui.OptsStatic().
			Text(rootHintText).
			Position(ui.Dpi(16, 108)).
			Size(ui.Dpi(404, 18)),
	)

	btnStart := ui.NewButton(
		wnd,
		ui.OptsButton().
			Text("启动服务").
			Position(ui.Dpi(16, 146)).
			Width(ui.DpiX(96)).
			Height(ui.DpiY(30)).
			CtrlStyle(co.BS_PUSHBUTTON|co.BS_FLAT),
	)
	btnStop := ui.NewButton(
		wnd,
		ui.OptsButton().
			Text("停止服务").
			Position(ui.Dpi(122, 146)).
			Width(ui.DpiX(112)).
			Height(ui.DpiY(30)).
			CtrlStyle(co.BS_PUSHBUTTON|co.BS_FLAT),
	)
	btnSetting := ui.NewButton(
		wnd,
		ui.OptsButton().
			Text("打开设置").
			Position(ui.Dpi(244, 146)).
			Width(ui.DpiX(92)).
			Height(ui.DpiY(30)).
			CtrlStyle(co.BS_PUSHBUTTON|co.BS_FLAT),
	)
	btnData := ui.NewButton(
		wnd,
		ui.OptsButton().
			Text("打开 userdata").
			Position(ui.Dpi(346, 146)).
			Width(ui.DpiX(86)).
			Height(ui.DpiY(30)).
			CtrlStyle(co.BS_PUSHBUTTON|co.BS_FLAT),
	)

	app := &launcherWindow{
		manager:    manager,
		detectErr:  detectErr,
		ctx:        ctx,
		cancel:     cancel,
		fastPoll:   make(chan struct{}, 1),
		theme:      theme,
		wnd:        wnd,
		title:      title,
		status:     status,
		detail:     detail,
		rootHint:   rootHint,
		btnStart:   btnStart,
		btnStop:    btnStop,
		btnSetting: btnSetting,
		btnData:    btnData,
	}

	app.bindEvents()
	app.bindTrayEvents()
	_ = wnd.RunAsMain()
}

func (w *launcherWindow) bindEvents() {
	w.wnd.On().WmCreate(func(_ ui.WmCreate) int {
		w.applyWindowTheme()
		if w.manager != nil {
			if conflict, _ := w.manager.installedConflict(); !conflict {
				tray, err := newTrayIcon(w)
				if err != nil {
					w.showError(fmt.Errorf("初始化托盘失败: %w", err))
				} else {
					w.tray = tray
				}
			}
		}
		w.refreshStatus()
		if w.detectErr == nil {
			go w.pollStatus()
		}
		go w.listenShowEvent()
		return 0
	})

	w.wnd.On().WmClose(func() {
		if w.detectErr != nil || w.manager == nil {
			w.cancel()
			w.wnd.Hwnd().DestroyWindow()
			return
		}
		if conflict, _ := w.manager.installedConflict(); conflict {
			w.cancel()
			w.wnd.Hwnd().DestroyWindow()
			return
		}
		w.hideToTray()
	})

	w.wnd.On().WmSize(func(p ui.WmSize) {
		if p.Request() == co.SIZE_REQ_MINIMIZED {
			if w.detectErr != nil || w.manager == nil {
				return
			}
			if conflict, _ := w.manager.installedConflict(); conflict {
				return
			}
			w.hideToTray()
		}
	})

	w.wnd.On().WmNcDestroy(func() {
		w.cancel()
		if w.tray != nil {
			w.tray.Close()
			w.tray = nil
		}
		if w.theme != nil {
			w.theme.Close()
			w.theme = nil
		}
		win.PostQuitMessage(0)
	})

	w.wnd.On().Wm(co.WM_CTLCOLORSTATIC, func(p ui.Wm) uintptr {
		if w.theme == nil {
			return 0
		}
		hdc := win.HDC(p.WParam)
		hCtl := win.HWND(p.LParam)
		_, _ = hdc.SetBkMode(co.BKMODE_TRANSPARENT)
		_, _ = hdc.SetBkColor(w.theme.bgColor)

		switch hCtl {
		case w.title.Hwnd():
			_, _ = hdc.SetTextColor(w.theme.titleColor)
		case w.status.Hwnd():
			_, _ = hdc.SetTextColor(w.theme.statusColor)
		default:
			_, _ = hdc.SetTextColor(w.theme.mutedColor)
		}
		return uintptr(w.theme.bgBrush)
	})

	w.btnStart.On().BnClicked(func() {
		w.setButtonsEnabled(false)
		w.status.Hwnd().SetWindowText("正在启动服务...")
		go func() {
			err := w.manager.startService()
			w.wnd.UiThread(func() {
				if err != nil {
					w.showError(err)
					w.refreshStatus()
				} else {
					// 启动成功：设置 5 秒冷却期
					w.cooldownUtil = time.Now().Add(5 * time.Second)
					w.status.Hwnd().SetWindowText("服务启动中...")
					w.detail.Hwnd().SetWindowText("等待服务就绪")
				}
				w.requestFastPoll()
			})
		}()
	})

	w.btnStop.On().BnClicked(func() {
		w.setButtonsEnabled(false)
		w.status.Hwnd().SetWindowText("正在停止服务...")
		go func() {
			_, err := w.manager.stopService()
			w.wnd.UiThread(func() {
				if err != nil {
					w.showError(err)
				}
				w.refreshStatus()
				w.requestFastPoll()
			})
		}()
	})

	w.btnSetting.On().BnClicked(func() {
		if err := w.manager.openSettings(); err != nil {
			w.showError(err)
		}
	})

	w.btnData.On().BnClicked(func() {
		if err := w.manager.openUserdataDir(); err != nil {
			w.showError(err)
		}
	})

}

func (w *launcherWindow) pollStatus() {
	fastUntil := time.Time{}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.fastPoll:
			// 操作后 20 秒内 1 秒轮询一次
			fastUntil = time.Now().Add(20 * time.Second)
			ticker.Reset(1 * time.Second)
		case <-ticker.C:
			w.wnd.UiThread(func() {
				w.refreshStatus()
			})
			// 快速轮询期结束后切回 5 秒
			if !fastUntil.IsZero() && time.Now().After(fastUntil) {
				fastUntil = time.Time{}
				ticker.Reset(5 * time.Second)
			}
		}
	}
}

// listenShowEvent 监听命名事件，收到信号时在 UI 线程恢复窗口
func (w *launcherWindow) listenShowEvent() {
	namePtr, _ := syscall.UTF16PtrFromString(showEventName)
	// 创建自动重置事件（初始非信号状态）
	handle, err := windows.CreateEvent(nil, 0, 0, namePtr)
	if err != nil {
		return
	}
	defer windows.CloseHandle(handle)

	for {
		event, err := windows.WaitForSingleObject(handle, windows.INFINITE)
		if err != nil || event != windows.WAIT_OBJECT_0 {
			return
		}
		select {
		case <-w.ctx.Done():
			return
		default:
			w.wnd.UiThread(func() {
				w.showFromTray()
			})
		}
	}
}

// requestFastPoll 请求 20 秒快速轮询（1秒/次）
func (w *launcherWindow) requestFastPoll() {
	select {
	case w.fastPoll <- struct{}{}:
	default:
	}
}

func (w *launcherWindow) refreshStatus() {
	// 检测失败：显示错误，禁用所有功能
	if w.detectErr != nil {
		w.status.Hwnd().SetWindowText("便携模式不可用")
		w.detail.Hwnd().SetWindowText(w.detectErr.Error())
		w.rootHint.Hwnd().SetWindowText("")
		w.setButtonsEnabled(false)
		if w.tray != nil {
			w.tray.UpdateMenuState(false, false, true)
		}
		return
	}

	// 冷却期内：显示启动中状态，不刷新按钮
	if time.Now().Before(w.cooldownUtil) {
		w.status.Hwnd().SetWindowText("服务启动中...")
		w.detail.Hwnd().SetWindowText("等待服务就绪")
		w.setButtonsEnabled(false)
		if w.tray != nil {
			w.tray.UpdateMenuState(false, false, false)
		}
		return
	}

	running := w.manager.serviceRunning()
	stoppable := running || w.manager.isRegistered()
	conflict, _ := w.manager.installedConflict()
	if conflict {
		w.status.Hwnd().SetWindowText("检测到已安装正式版")
		w.detail.Hwnd().SetWindowText("为避免覆盖现有输入法注册信息，便携模式已禁用")
		w.rootHint.Hwnd().SetWindowText("已安装位置: " + compactPathWithMax(w.manager.installedConflictPath(), 42))
		w.btnStart.Hwnd().EnableWindow(false)
		w.btnStop.Hwnd().EnableWindow(false)
		w.btnSetting.Hwnd().EnableWindow(false)
		w.btnData.Hwnd().EnableWindow(false)
		if w.tray != nil {
			w.tray.UpdateMenuState(false, false, true)
		}
		return
	}
	if running {
		w.status.Hwnd().SetWindowText("服务状态: 运行中")
		w.detail.Hwnd().SetWindowText("输入法服务正在运行")
	} else {
		w.status.Hwnd().SetWindowText("服务状态: 已停止")
		w.detail.Hwnd().SetWindowText("点击启动服务后会自动注册并启动")
	}
	w.rootHint.Hwnd().SetWindowText("目录: " + compactPathWithMax(w.manager.cfg.RootDir, 52))
	w.btnStart.Hwnd().EnableWindow(!running)
	w.btnStop.Hwnd().EnableWindow(stoppable)
	w.btnSetting.Hwnd().EnableWindow(running) // 服务未运行时设置不可用
	w.btnData.Hwnd().EnableWindow(true)
	if w.tray != nil {
		w.tray.UpdateMenuState(running, stoppable, false)
	}
}

func (w *launcherWindow) setButtonsEnabled(enabled bool) {
	w.btnStart.Hwnd().EnableWindow(enabled)
	w.btnStop.Hwnd().EnableWindow(enabled)
	w.btnSetting.Hwnd().EnableWindow(enabled)
	w.btnData.Hwnd().EnableWindow(enabled)
}

func (w *launcherWindow) showError(err error) {
	if err == nil {
		return
	}
	w.wnd.Hwnd().MessageBox(err.Error(), "清风输入法便携启动器", co.MB_ICONERROR)
}

func (w *launcherWindow) hideToTray() {
	w.wnd.Hwnd().ShowWindow(co.SW_HIDE)
}

func (w *launcherWindow) showFromTray() {
	hwnd := w.wnd.Hwnd()
	hwnd.ShowWindow(co.SW_SHOW)
	hwnd.ShowWindow(co.SW_RESTORE)
	hwnd.SetForegroundWindow()
}

func compactPath(path string) string {
	return compactPathWithMax(path, 58)
}

func compactPathWithMax(path string, maxLen int) string {
	if maxLen <= 0 || len(path) <= maxLen {
		return path
	}
	parts := strings.Split(path, `\`)
	if len(parts) < 4 {
		if maxLen <= 3 {
			return path[:maxLen]
		}
		return path[:maxLen-3] + "..."
	}
	tail := strings.Join(parts[len(parts)-2:], `\`)
	head := parts[0] + `\...\`
	if len(head)+len(tail) <= maxLen {
		return head + tail
	}
	if len(tail) > maxLen-6 {
		tail = "..." + tail[len(tail)-(maxLen-6):]
	}
	return head + tail
}

type windowTheme struct {
	bgColor     win.COLORREF
	titleColor  win.COLORREF
	statusColor win.COLORREF
	mutedColor  win.COLORREF
	bgBrush     win.HBRUSH
}

func newWindowTheme() (*windowTheme, error) {
	bg := win.RGB(248, 250, 252)
	brush, err := win.CreateBrushIndirect(&win.LOGBRUSH{
		LbStyle: co.BRS_SOLID,
		LbColor: bg,
	})
	if err != nil {
		return nil, err
	}
	return &windowTheme{
		bgColor:     bg,
		titleColor:  win.RGB(15, 23, 42),
		statusColor: win.RGB(37, 99, 235),
		mutedColor:  win.RGB(100, 116, 139),
		bgBrush:     brush,
	}, nil
}

func (t *windowTheme) Close() {
	if t == nil || t.bgBrush == 0 {
		return
	}
	_ = t.bgBrush.DeleteObject()
	t.bgBrush = 0
}

func (w *launcherWindow) applyWindowTheme() {
	if w.theme == nil {
		return
	}
	_ = w.wnd.Hwnd().DwmSetWindowAttribute(win.DwmAttrCaptionColor(win.RGB(241, 245, 249)))
	_ = w.wnd.Hwnd().DwmSetWindowAttribute(win.DwmAttrTextColor(win.RGB(15, 23, 42)))
	_ = w.wnd.Hwnd().DwmSetWindowAttribute(win.DwmAttrBorderColor(win.RGB(203, 213, 225)))
}

// acquireSingleInstance 尝试获取全局互斥体，成功返回 true
func acquireSingleInstance() bool {
	handle, err := createMutexW(mutexName)
	if err != nil {
		return false
	}
	// 保持 handle 不关闭，进程退出时自动释放
	_ = handle
	return true
}

var showEventName = "Local\\WindPortable" + buildvariant.Suffix() + "ShowEvent"

// activateExistingWindow 通过命名事件通知已有实例显示窗口
func activateExistingWindow() {
	namePtr, _ := syscall.UTF16PtrFromString(showEventName)
	handle, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, namePtr)
	if err != nil {
		return
	}
	windows.SetEvent(handle)
	windows.CloseHandle(handle)
}
