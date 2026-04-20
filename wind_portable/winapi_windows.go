//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modInput                     = windows.NewLazySystemDLL("input.dll")
	procInstallLayoutOrTip       = modInput.NewProc("InstallLayoutOrTip")
	modShell32                   = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteW            = modShell32.NewProc("ShellExecuteW")
	modKernel32Ext               = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMutexW             = modKernel32Ext.NewProc("CreateMutexW")
	modComdlg32                  = windows.NewLazySystemDLL("comdlg32.dll")
	modUxTheme                   = windows.NewLazySystemDLL("uxtheme.dll")
	procEnableThemeDialogTexture = modUxTheme.NewProc("EnableThemeDialogTexture")
)

// enableThemeDialogTexture 启用窗口的主题 Tab 页背景纹理，
// 使子控件（Static 等）的背景与 Tab 页主题一致。
func enableThemeDialogTexture(hwnd uintptr) {
	const ETDT_ENABLETAB = 0x00000006 // ETDT_ENABLE | ETDT_USETABTEXTURE
	procEnableThemeDialogTexture.Call(hwnd, ETDT_ENABLETAB)
}

// openFileDialog displays an Open File dialog and returns the selected path.
// Uses PowerShell to invoke the .NET WinForms dialog, which is reliable on 64-bit Windows.
// Returns empty string if the user cancels.
func openFileDialog(owner uintptr, title string, filter string) string {
	titleEscaped := strings.ReplaceAll(title, "'", "''")
	// WinForms Filter 格式与 Win32 一致，使用 | 分隔，去除末尾多余的 |
	filter = strings.TrimRight(filter, "|")
	filterEscaped := strings.ReplaceAll(filter, "'", "''")

	ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; $d = [System.Windows.Forms.OpenFileDialog]::new(); $d.Title = '%s'; $d.Filter = '%s'; if($d.ShowDialog() -eq 'OK') { $d.FileName } else { '' }`, titleEscaped, filterEscaped)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps)
	cmd.SysProcAttr = defaultHiddenProcessAttrs()
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// selectFolderDialog displays a folder browser dialog and returns the selected path.
// Returns empty string if the user cancels.
func selectFolderDialog(owner uintptr, title string) string {
	titleEscaped := strings.ReplaceAll(title, "'", "''")

	ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; $d = [System.Windows.Forms.FolderBrowserDialog]::new(); $d.Description = '%s'; $d.ShowNewFolderButton = $true; if($d.ShowDialog() -eq 'OK') { $d.SelectedPath } else { '' }`, titleEscaped)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps)
	cmd.SysProcAttr = defaultHiddenProcessAttrs()
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// createMutexW 创建命名互斥体，如果已存在则返回错误
func createMutexW(name string) (uintptr, error) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	handle, _, callErr := procCreateMutexW.Call(
		0,
		0,
		uintptr(unsafe.Pointer(namePtr)),
	)
	if handle == 0 {
		return 0, callErr
	}
	// ERROR_ALREADY_EXISTS = 183
	if callErr == syscall.Errno(183) {
		windows.CloseHandle(windows.Handle(handle))
		return 0, fmt.Errorf("mutex already exists")
	}
	return handle, nil
}

// isAdmin 检查当前进程是否以管理员权限运行
func isAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)
	member, err := windows.Token(0).IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

// runElevated 以管理员权限运行自身并传递指定参数，等待子进程退出
func runElevated(args string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败: %w", err)
	}

	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	exePtr, _ := syscall.UTF16PtrFromString(exePath)
	argsPtr, _ := syscall.UTF16PtrFromString(args)

	ret, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		0,
		0, // SW_HIDE
	)
	if ret <= 32 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return fmt.Errorf("请求管理员权限失败: %w", callErr)
		}
		return fmt.Errorf("请求管理员权限失败 (错误码 %d)", ret)
	}
	return nil
}

func installLayoutOrTip(profile string, flags uint32) error {
	if strings.TrimSpace(profile) == "" {
		return fmt.Errorf("profile 为空")
	}
	ptr, err := syscall.UTF16PtrFromString(profile)
	if err != nil {
		return err
	}
	ret, _, callErr := procInstallLayoutOrTip.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(flags),
	)
	if ret == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return fmt.Errorf("InstallLayoutOrTip 失败: %w", callErr)
		}
		return fmt.Errorf("InstallLayoutOrTip 失败")
	}
	return nil
}

func defaultHiddenProcessAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}

func stringsTrimSpace(s string) string {
	return strings.TrimSpace(s)
}

func stringsToLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
