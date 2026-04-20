//go:build windows

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	if detectErr == nil {
		cleanOldFiles(cfg.RootDir)
	}

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
	cooldownUtil time.Time     // 启动冷却期截止时间
	wnd          *ui.Main
	tab          *ui.Tab
	// 运行 Tab
	status     *ui.Static
	detail     *ui.Static
	rootHint   *ui.Static
	btnStart   *ui.Button
	btnStop    *ui.Button
	btnSetting *ui.Button
	btnData    *ui.Button
	// 部署 Tab
	deployHint    *ui.Static
	btnUpdate     *ui.Button
	btnDeployCopy *ui.Button
	btnDeployZip  *ui.Button
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
			Size(ui.Dpi(470, 300)).
			ClassIconId(1),
	)

	// 全页 Tab 控件（扁平风格）
	defaultTab := 0
	if detectErr != nil {
		defaultTab = 1
	}
	tab := ui.NewTab(
		wnd,
		ui.OptsTab().
			Titles("运行", "部署").
			Position(ui.Dpi(4, 4)).
			Size(ui.Dpi(446, 258)).
			Selected(defaultTab).
			CtrlStyle(co.TCS_HOTTRACK),
	)

	// ── 运行 Tab ──
	runParent := tab.Item(0).Child()
	status := ui.NewStatic(
		runParent,
		ui.OptsStatic().
			Text("正在检查服务状态...").
			Position(ui.Dpi(10, 10)).
			Size(ui.Dpi(400, 20)).
			WndExStyle(co.WS_EX_TRANSPARENT),
	)
	detail := ui.NewStatic(
		runParent,
		ui.OptsStatic().
			Text("准备就绪").
			Position(ui.Dpi(10, 34)).
			Size(ui.Dpi(410, 50)).
			WndExStyle(co.WS_EX_TRANSPARENT),
	)
	rootHintText := ""
	if manager != nil {
		rootHintText = "目录: " + compactPath(manager.cfg.RootDir)
	}
	rootHint := ui.NewStatic(
		runParent,
		ui.OptsStatic().
			Text(rootHintText).
			Position(ui.Dpi(10, 90)).
			Size(ui.Dpi(410, 18)).
			WndExStyle(co.WS_EX_TRANSPARENT),
	)
	btnStart := ui.NewButton(
		runParent,
		ui.OptsButton().
			Text("启动服务").
			Position(ui.Dpi(10, 118)).
			Width(ui.DpiX(100)).
			Height(ui.DpiY(32)),
	)
	btnStop := ui.NewButton(
		runParent,
		ui.OptsButton().
			Text("停止服务").
			Position(ui.Dpi(120, 118)).
			Width(ui.DpiX(100)).
			Height(ui.DpiY(32)),
	)
	btnSetting := ui.NewButton(
		runParent,
		ui.OptsButton().
			Text("打开设置").
			Position(ui.Dpi(10, 160)).
			Width(ui.DpiX(100)).
			Height(ui.DpiY(32)),
	)
	btnData := ui.NewButton(
		runParent,
		ui.OptsButton().
			Text("打开 userdata").
			Position(ui.Dpi(120, 160)).
			Width(ui.DpiX(120)).
			Height(ui.DpiY(32)),
	)

	// ── 部署 Tab ──
	deployParent := tab.Item(1).Child()
	deployHint := ui.NewStatic(
		deployParent,
		ui.OptsStatic().
			Text("更新当前安装、复制当前文件到新目录、或从 ZIP 包部署到新目录。").
			Position(ui.Dpi(10, 10)).
			Size(ui.Dpi(410, 36)).
			WndExStyle(co.WS_EX_TRANSPARENT),
	)
	btnUpdate := ui.NewButton(
		deployParent,
		ui.OptsButton().
			Text("更新当前版本").
			Position(ui.Dpi(10, 56)).
			Width(ui.DpiX(128)).
			Height(ui.DpiY(32)),
	)
	btnDeployCopy := ui.NewButton(
		deployParent,
		ui.OptsButton().
			Text("复制到目录").
			Position(ui.Dpi(148, 56)).
			Width(ui.DpiX(128)).
			Height(ui.DpiY(32)),
	)
	btnDeployZip := ui.NewButton(
		deployParent,
		ui.OptsButton().
			Text("从 ZIP 部署").
			Position(ui.Dpi(286, 56)).
			Width(ui.DpiX(128)).
			Height(ui.DpiY(32)),
	)

	app := &launcherWindow{
		manager:       manager,
		detectErr:     detectErr,
		ctx:           ctx,
		cancel:        cancel,
		fastPoll:      make(chan struct{}, 1),
		theme:         theme,
		wnd:           wnd,
		tab:           tab,
		status:        status,
		detail:        detail,
		rootHint:      rootHint,
		btnStart:      btnStart,
		btnStop:       btnStop,
		btnSetting:    btnSetting,
		btnData:       btnData,
		deployHint:    deployHint,
		btnUpdate:     btnUpdate,
		btnDeployCopy: btnDeployCopy,
		btnDeployZip:  btnDeployZip,
	}

	app.bindEvents()
	app.bindTrayEvents()
	_ = wnd.RunAsMain()
}

func (w *launcherWindow) bindEvents() {
	// Tab 子容器：让 Static 控件背景透明，融入主题背景
	nullBrush, _ := win.GetStockObject(co.STOCK_NULL_BRUSH)
	staticBgHandler := func(p ui.Wm) uintptr {
		hdc := win.HDC(p.WParam)
		_, _ = hdc.SetBkMode(co.BKMODE_TRANSPARENT)
		return uintptr(nullBrush)
	}
	w.tab.Item(0).Child().On().Wm(co.WM_CTLCOLORSTATIC, staticBgHandler)
	w.tab.Item(1).Child().On().Wm(co.WM_CTLCOLORSTATIC, staticBgHandler)

	w.wnd.On().WmCreate(func(_ ui.WmCreate) int {
		w.applyWindowTheme()
		// 启用 Tab 页主题背景，使 Static 等子控件背景融入主题
		enableThemeDialogTexture(uintptr(w.tab.Item(0).Child().Hwnd()))
		enableThemeDialogTexture(uintptr(w.tab.Item(1).Child().Hwnd()))
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

	// 部署 Tab: 更新当前版本
	w.btnUpdate.On().BnClicked(func() {
		zipPath := openFileDialog(
			uintptr(w.wnd.Hwnd()),
			"选择便携版更新包",
			"ZIP 压缩包 (*.zip)|*.zip|所有文件 (*.*)|*.*",
		)
		if zipPath == "" {
			return
		}
		if err := validateZip(zipPath); err != nil {
			w.showError(fmt.Errorf("无效的更新包: %w", err))
			return
		}
		ret, _ := w.wnd.Hwnd().MessageBox(
			fmt.Sprintf("确认从以下文件更新便携版？\n\n%s", zipPath),
			"确认更新",
			co.MB_OKCANCEL|co.MB_ICONQUESTION,
		)
		if ret != co.ID_OK {
			return
		}
		w.setButtonsEnabled(false)
		w.status.Hwnd().SetWindowText("正在更新...")
		w.detail.Hwnd().SetWindowText("正在停止服务并准备更新")
		go func() {
			updateErr := w.performUpdate(zipPath)
			w.wnd.UiThread(func() {
				if updateErr != nil {
					w.showError(updateErr)
				}
				w.refreshStatus()
			})
		}()
	})

	// 部署 Tab: 复制当前文件到新目录
	w.btnDeployCopy.On().BnClicked(func() {
		sourceDir := findDeploySourceDir()
		if sourceDir == "" {
			w.showError(fmt.Errorf("未找到可复制的源文件"))
			return
		}
		targetDir := selectFolderDialog(
			uintptr(w.wnd.Hwnd()),
			"选择部署目标目录",
		)
		if targetDir == "" {
			return
		}
		if isProtectedDir(targetDir) {
			w.showError(fmt.Errorf("不能部署到系统保护目录: %s", targetDir))
			return
		}
		ret, _ := w.wnd.Hwnd().MessageBox(
			fmt.Sprintf("确认将当前文件复制到以下目录？\n\n源: %s\n目标: %s", sourceDir, targetDir),
			"确认部署",
			co.MB_OKCANCEL|co.MB_ICONQUESTION,
		)
		if ret != co.ID_OK {
			return
		}
		w.setButtonsEnabled(false)
		w.status.Hwnd().SetWindowText("正在部署...")
		w.detail.Hwnd().SetWindowText("正在复制文件到目标目录")
		go func() {
			deployErr := deployFromDirectory(sourceDir, targetDir)
			w.wnd.UiThread(func() {
				if deployErr != nil {
					w.showError(fmt.Errorf("部署失败: %w", deployErr))
				} else {
					w.detail.Hwnd().SetWindowText("已部署到: " + compactPath(targetDir))
					_, _ = w.wnd.Hwnd().MessageBox(
						fmt.Sprintf("便携版已成功部署到:\n\n%s\n\n请到该目录运行 wind_portable.exe 启动。", targetDir),
						"部署完成",
						co.MB_ICONINFORMATION,
					)
				}
				w.refreshStatus()
			})
		}()
	})

	// 部署 Tab: 从 ZIP 包部署到新目录
	w.btnDeployZip.On().BnClicked(func() {
		zipPath := openFileDialog(
			uintptr(w.wnd.Hwnd()),
			"选择便携版压缩包",
			"ZIP 压缩包 (*.zip)|*.zip|所有文件 (*.*)|*.*",
		)
		if zipPath == "" {
			return
		}
		if err := validateZip(zipPath); err != nil {
			w.showError(fmt.Errorf("无效的压缩包: %w", err))
			return
		}
		targetDir := selectFolderDialog(
			uintptr(w.wnd.Hwnd()),
			"选择部署目标目录",
		)
		if targetDir == "" {
			return
		}
		if isProtectedDir(targetDir) {
			w.showError(fmt.Errorf("不能部署到系统保护目录: %s", targetDir))
			return
		}
		ret, _ := w.wnd.Hwnd().MessageBox(
			fmt.Sprintf("确认将 ZIP 包部署到以下目录？\n\n%s", targetDir),
			"确认部署",
			co.MB_OKCANCEL|co.MB_ICONQUESTION,
		)
		if ret != co.ID_OK {
			return
		}
		w.setButtonsEnabled(false)
		w.status.Hwnd().SetWindowText("正在部署...")
		w.detail.Hwnd().SetWindowText("正在解压文件到目标目录")
		go func() {
			_, deployErr := deployFromZip(zipPath, targetDir)
			w.wnd.UiThread(func() {
				if deployErr != nil {
					w.showError(fmt.Errorf("部署失败: %w", deployErr))
				} else {
					w.detail.Hwnd().SetWindowText("已部署到: " + compactPath(targetDir))
					_, _ = w.wnd.Hwnd().MessageBox(
						fmt.Sprintf("便携版已成功部署到:\n\n%s\n\n请到该目录运行 wind_portable.exe 启动。", targetDir),
						"部署完成",
						co.MB_ICONINFORMATION,
					)
				}
				w.refreshStatus()
			})
		}()
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
	// 检测失败：禁用运行功能，部署 Tab 保持可用
	if w.detectErr != nil {
		w.status.Hwnd().SetWindowText("便携模式不可用")
		w.detail.Hwnd().SetWindowText(w.detectErr.Error())
		w.rootHint.Hwnd().SetWindowText("")
		w.setButtonsEnabled(false)
		// 部署 Tab 按钮保持可用
		w.btnUpdate.Hwnd().EnableWindow(false) // 无当前安装，不能更新
		w.btnDeployCopy.Hwnd().EnableWindow(findDeploySourceDir() != "")
		w.btnDeployZip.Hwnd().EnableWindow(true)
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
		// 运行 Tab 全部禁用
		w.btnStart.Hwnd().EnableWindow(false)
		w.btnStop.Hwnd().EnableWindow(false)
		w.btnSetting.Hwnd().EnableWindow(false)
		w.btnData.Hwnd().EnableWindow(false)
		// 部署 Tab: 不能更新当前版本，但可以部署到新目录
		w.btnUpdate.Hwnd().EnableWindow(false)
		w.btnDeployCopy.Hwnd().EnableWindow(findDeploySourceDir() != "")
		w.btnDeployZip.Hwnd().EnableWindow(true)
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
	// 运行 Tab
	w.btnStart.Hwnd().EnableWindow(!running)
	w.btnStop.Hwnd().EnableWindow(stoppable)
	w.btnSetting.Hwnd().EnableWindow(running)
	w.btnData.Hwnd().EnableWindow(true)
	// 部署 Tab
	w.btnUpdate.Hwnd().EnableWindow(true)
	w.btnDeployCopy.Hwnd().EnableWindow(true)
	w.btnDeployZip.Hwnd().EnableWindow(true)
	if w.tray != nil {
		w.tray.UpdateMenuState(running, stoppable, false)
	}
}

func (w *launcherWindow) setButtonsEnabled(enabled bool) {
	w.btnStart.Hwnd().EnableWindow(enabled)
	w.btnStop.Hwnd().EnableWindow(enabled)
	w.btnSetting.Hwnd().EnableWindow(enabled)
	w.btnData.Hwnd().EnableWindow(enabled)
	w.btnUpdate.Hwnd().EnableWindow(enabled)
	w.btnDeployCopy.Hwnd().EnableWindow(enabled)
	w.btnDeployZip.Hwnd().EnableWindow(enabled)
}

func (w *launcherWindow) showError(err error) {
	if err == nil {
		return
	}
	w.wnd.Hwnd().MessageBox(err.Error(), "清风输入法便携启动器", co.MB_ICONERROR)
}

func (w *launcherWindow) performUpdate(zipPath string) error {
	// Step 1: Set stopped flag
	w.wnd.UiThread(func() {
		w.detail.Hwnd().SetWindowText("正在设置守卫标志...")
	})
	_ = w.manager.setStoppedFlag()

	// Step 2: Stop service
	w.wnd.UiThread(func() {
		w.detail.Hwnd().SetWindowText("正在停止服务...")
	})
	_, _ = w.manager.stopService()

	// Step 3: Deploy from ZIP
	w.wnd.UiThread(func() {
		w.detail.Hwnd().SetWindowText("正在替换文件...")
	})
	needsRestart, err := deployFromZip(zipPath, w.manager.cfg.RootDir)
	if err != nil {
		_ = w.manager.clearStoppedFlag()
		return fmt.Errorf("文件替换失败: %w", err)
	}

	// Step 4: Clear stopped flag
	_ = w.manager.clearStoppedFlag()

	// Step 5: Restart service
	w.wnd.UiThread(func() {
		w.detail.Hwnd().SetWindowText("正在重新注册输入法...")
	})
	if startErr := w.manager.startService(); startErr != nil {
		return fmt.Errorf("重启服务失败: %w", startErr)
	}

	// Step 6: Handle self-update
	if needsRestart {
		w.wnd.UiThread(func() {
			ret, _ := w.wnd.Hwnd().MessageBox(
				"启动器已更新到新版本，需要重新启动。\n是否立即重启？",
				"更新完成",
				co.MB_YESNO|co.MB_ICONINFORMATION,
			)
			if ret == co.ID_YES {
				w.restartSelf()
			}
		})
	} else {
		w.wnd.UiThread(func() {
			_, _ = w.wnd.Hwnd().MessageBox("便携版更新完成！", "更新完成", co.MB_ICONINFORMATION)
		})
	}
	return nil
}

func (w *launcherWindow) restartSelf() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	_ = cmd.Start()
	w.cancel()
	w.wnd.Hwnd().DestroyWindow()
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
