//go:build windows

package main

import (
	"syscall"

	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/ui"
	"github.com/rodrigocfd/windigo/win"
)

const (
	trayIconID          = 1
	menuCmdShow  uint16 = 1001
	menuCmdStart uint16 = 1002
	menuCmdStop  uint16 = 1003
	menuCmdSet   uint16 = 1004
	menuCmdData  uint16 = 1005
	menuCmdExit  uint16 = 1006
)

var trayCallbackMsg = co.WM_APP + 1

type trayIcon struct {
	app     *launcherWindow
	menu    win.HMENU
	nid     win.NOTIFYICONDATA
	hIcon   win.HICON
	visible bool
}

func newTrayIcon(app *launcherWindow) (*trayIcon, error) {
	hIcon, err := loadTrayIcon(app.manager.cfg.IconPath)
	if err != nil {
		return nil, err
	}

	menu, err := buildTrayMenu()
	if err != nil {
		return nil, err
	}

	t := &trayIcon{
		app:   app,
		menu:  menu,
		hIcon: hIcon,
	}

	t.nid.SetCbSize()
	t.nid.HWnd = app.wnd.Hwnd()
	t.nid.UID = trayIconID
	t.nid.UFlags = co.NIF_MESSAGE | co.NIF_ICON | co.NIF_TIP
	t.nid.UCallbackMessage = trayCallbackMsg
	t.nid.HIcon = hIcon
	t.nid.SetSzTip("清风输入法便携启动器")

	if err := win.Shell_NotifyIcon(co.NIM_ADD, &t.nid); err != nil {
		_ = t.menu.DestroyMenu()
		return nil, err
	}
	t.visible = true
	conflict, _ := app.manager.installedConflict()
	t.UpdateMenuState(app.manager.serviceRunning(), app.manager.serviceRunning() || app.manager.isRegistered(), conflict)
	return t, nil
}

func loadTrayIcon(iconPath string) (win.HICON, error) {
	hBig, hSmall, err := loadAppIcons(iconPath)
	if err == nil {
		if hSmall != 0 {
			return hSmall, nil
		}
		if hBig != 0 {
			return hBig, nil
		}
	}
	return win.HINSTANCE(0).LoadIcon(win.IconResIdi(co.IDI_APPLICATION))
}

func loadAppIcons(iconPath string) (big win.HICON, small win.HICON, err error) {
	// 使用系统推荐的图标尺寸（高 DPI 下会大于 16x16）
	smCxRaw := win.GetSystemMetrics(co.SM_CXSMICON)
	smCyRaw := win.GetSystemMetrics(co.SM_CYSMICON)
	if smCxRaw == 0 {
		smCxRaw = 16
	}
	if smCyRaw == 0 {
		smCyRaw = 16
	}
	smCx := int(smCxRaw)
	smCy := int(smCyRaw)

	hInst, _ := win.GetModuleHandle("")
	if hInst != 0 {
		// 加载大图标（用于 Alt-Tab 等场景）
		if hGdi, loadErr := hInst.LoadImage(win.ResIdInt(1), co.IMAGE_ICON, 0, 0, co.LR_DEFAULTSIZE|co.LR_SHARED); loadErr == nil {
			big = win.HICON(hGdi)
		}
		// 加载适配系统 DPI 的小图标（用于托盘）
		if hGdi, loadErr := hInst.LoadImage(win.ResIdInt(1), co.IMAGE_ICON, smCx, smCy, co.LR_DEFAULTCOLOR|co.LR_SHARED); loadErr == nil {
			small = win.HICON(hGdi)
		}
	}
	if big != 0 || small != 0 {
		if big == 0 {
			big = small
		}
		if small == 0 {
			small = big
		}
		return big, small, nil
	}

	if iconPath != "" {
		if hGdi, loadErr := win.HINSTANCE(0).LoadImage(win.ResIdStr(iconPath), co.IMAGE_ICON, 0, 0, co.LR_DEFAULTSIZE|co.LR_LOADFROMFILE); loadErr == nil {
			big = win.HICON(hGdi)
		}
		if hGdi, loadErr := win.HINSTANCE(0).LoadImage(win.ResIdStr(iconPath), co.IMAGE_ICON, smCx, smCy, co.LR_LOADFROMFILE); loadErr == nil {
			small = win.HICON(hGdi)
		}
		if big != 0 || small != 0 {
			if big == 0 {
				big = small
			}
			if small == 0 {
				small = big
			}
			return big, small, nil
		}
	}

	return 0, 0, syscall.Errno(0)
}

func (t *trayIcon) UpdateMenuState(running bool, stoppable bool, conflict bool) {
	enableActions := !conflict
	_ = t.menu.EnableMenuItemByCmd(enableActions && !running, menuCmdStart)
	_ = t.menu.EnableMenuItemByCmd(enableActions && stoppable, menuCmdStop)
	_ = t.menu.EnableMenuItemByCmd(enableActions, menuCmdSet)
	_ = t.menu.EnableMenuItemByCmd(enableActions, menuCmdData)
}

func (t *trayIcon) ShowMenu() {
	conflict, _ := t.app.manager.installedConflict()
	t.UpdateMenuState(t.app.manager.serviceRunning(), t.app.manager.serviceRunning() || t.app.manager.isRegistered(), conflict)
	t.app.wnd.Hwnd().SetForegroundWindow()

	pos := trayAnchorPosition(t.nid)
	_, _ = t.menu.TrackPopupMenu(
		co.TPM_LEFTALIGN|co.TPM_BOTTOMALIGN|co.TPM_RIGHTBUTTON,
		int(pos.X),
		int(pos.Y),
		t.app.wnd.Hwnd(),
	)
	t.app.wnd.Hwnd().PostMessage(co.WM_NULL, 0, 0)
	_ = win.Shell_NotifyIcon(co.NIM_SETFOCUS, &t.nid)
}

func (t *trayIcon) Close() {
	if t.visible {
		_ = win.Shell_NotifyIcon(co.NIM_DELETE, &t.nid)
		t.visible = false
	}
	if t.menu != 0 {
		_ = t.menu.DestroyMenu()
		t.menu = 0
	}
}

func (w *launcherWindow) bindTrayEvents() {
	w.wnd.On().Wm(trayCallbackMsg, func(p ui.Wm) uintptr {
		switch uintptr(p.LParam) {
		case uintptr(co.WM_CONTEXTMENU), uintptr(co.WM_RBUTTONUP):
			if w.tray != nil {
				w.tray.ShowMenu()
			}
		case uintptr(co.WM_LBUTTONUP), uintptr(co.WM_LBUTTONDBLCLK), uintptr(co.NIN_SELECT):
			w.showFromTray()
		}
		return 0
	})

	w.wnd.On().WmCommandAccelMenu(menuCmdShow, func() {
		w.showFromTray()
	})
	w.wnd.On().WmCommandAccelMenu(menuCmdStart, func() {
		if err := w.manager.startService(); err != nil {
			w.showError(err)
			return
		}
		w.refreshStatus()
	})
	w.wnd.On().WmCommandAccelMenu(menuCmdStop, func() {
		if _, err := w.manager.stopService(); err != nil {
			w.showError(err)
			return
		}
		w.refreshStatus()
	})
	w.wnd.On().WmCommandAccelMenu(menuCmdSet, func() {
		if err := w.manager.openSettings(); err != nil {
			w.showError(err)
		}
	})
	w.wnd.On().WmCommandAccelMenu(menuCmdData, func() {
		if err := w.manager.openUserdataDir(); err != nil {
			w.showError(err)
		}
	})
	w.wnd.On().WmCommandAccelMenu(menuCmdExit, func() {
		if w.manager != nil && (w.manager.serviceRunning() || w.manager.isRegistered()) {
			ret, _ := w.wnd.Hwnd().MessageBox(
				"输入法服务正在运行，退出启动器将同时停止服务并注销输入法。\n\n确定要退出吗？",
				"清风输入法便携启动器",
				co.MB_OKCANCEL|co.MB_ICONQUESTION,
			)
			if ret != co.ID_OK {
				return
			}
			w.setButtonsEnabled(false)
			w.status.Hwnd().SetWindowText("正在停止服务...")
			if _, err := w.manager.stopService(); err != nil {
				w.showError(err)
				w.refreshStatus()
				return
			}
		}
		w.cancel()
		w.wnd.Hwnd().DestroyWindow()
	})
}

func buildTrayMenu() (win.HMENU, error) {
	menu, err := win.CreatePopupMenu()
	if err != nil {
		return 0, err
	}

	for _, item := range []struct {
		id   uint16
		text string
		sep  bool
	}{
		{id: menuCmdShow, text: "显示窗口"},
		{sep: true},
		{id: menuCmdStart, text: "启动服务"},
		{id: menuCmdStop, text: "停止服务"},
		{id: menuCmdSet, text: "打开设置"},
		{id: menuCmdData, text: "打开 userdata"},
		{sep: true},
		{id: menuCmdExit, text: "退出"},
	} {
		if item.sep {
			if err := appendSeparator(menu); err != nil {
				_ = menu.DestroyMenu()
				return 0, err
			}
			continue
		}
		if err := appendMenuItem(menu, item.id, item.text); err != nil {
			_ = menu.DestroyMenu()
			return 0, err
		}
	}

	return menu, nil
}

func appendMenuItem(menu win.HMENU, id uint16, text string) error {
	var mii win.MENUITEMINFO
	mii.SetCbSize()
	mii.FMask = co.MIIM_ID | co.MIIM_STRING | co.MIIM_STATE
	mii.FState = co.MFS_ENABLED
	mii.WId = uint32(id)
	ptr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return err
	}
	mii.DwTypeData = ptr
	mii.Cch = uint32(len(text))
	count, _ := menu.GetMenuItemCount()
	return menu.InsertMenuItemByPos(count, &mii)
}

func appendSeparator(menu win.HMENU) error {
	var mii win.MENUITEMINFO
	mii.SetCbSize()
	mii.FMask = co.MIIM_FTYPE
	mii.FType = co.MFT_SEPARATOR
	count, _ := menu.GetMenuItemCount()
	return menu.InsertMenuItemByPos(count, &mii)
}

func trayAnchorPosition(nid win.NOTIFYICONDATA) win.POINT {
	var nii win.NOTIFYICONIDENTIFIER
	nii.SetCbSize()
	nii.HWnd = nid.HWnd
	nii.UID = nid.UID

	if rect, err := win.Shell_NotifyIconGetRect(&nii); err == nil {
		return win.POINT{X: rect.Left, Y: rect.Top}
	}

	if pos, err := win.GetCursorPos(); err == nil {
		return pos
	}
	return win.POINT{}
}
