//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modInput               = windows.NewLazySystemDLL("input.dll")
	procInstallLayoutOrTip = modInput.NewProc("InstallLayoutOrTip")
	modShell32             = windows.NewLazySystemDLL("shell32.dll")
	procShellExecuteW      = modShell32.NewProc("ShellExecuteW")
	modKernel32Ext         = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMutexW       = modKernel32Ext.NewProc("CreateMutexW")
)

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
