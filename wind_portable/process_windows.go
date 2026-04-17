//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procQueryFullProcessImageName = modKernel32.NewProc("QueryFullProcessImageNameW")
)

func terminateProcessByPath(targetPath string) (bool, error) {
	targetPath = filepath.Clean(targetPath)

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false, fmt.Errorf("创建进程快照失败: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		if err == syscall.ERROR_NO_MORE_FILES {
			return false, nil
		}
		return false, fmt.Errorf("读取进程列表失败: %w", err)
	}

	stopped := false
	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(name, filepath.Base(targetPath)) {
			match, err := processPathMatches(entry.ProcessID, targetPath)
			if err != nil {
				return stopped, err
			}
			if match {
				if err := terminatePID(entry.ProcessID); err != nil {
					return stopped, err
				}
				stopped = true
			}
		}

		err = windows.Process32Next(snapshot, &entry)
		if err == syscall.ERROR_NO_MORE_FILES {
			break
		}
		if err != nil {
			return stopped, fmt.Errorf("遍历进程列表失败: %w", err)
		}
	}

	return stopped, nil
}

func processPathMatches(pid uint32, targetPath string) (bool, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false, nil
	}
	defer windows.CloseHandle(handle)

	processPath, err := queryFullProcessImageName(handle)
	if err != nil {
		return false, nil
	}
	return strings.EqualFold(filepath.Clean(processPath), filepath.Clean(targetPath)), nil
}

func terminatePID(pid uint32) error {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return fmt.Errorf("打开进程失败(pid=%d): %w", pid, err)
	}
	defer windows.CloseHandle(handle)

	if err := windows.TerminateProcess(handle, 0); err != nil {
		return fmt.Errorf("停止进程失败(pid=%d): %w", pid, err)
	}

	_, _ = windows.WaitForSingleObject(handle, uint32((2 * time.Second).Milliseconds()))
	return nil
}

// processExistsByName 检查指定进程名是否正在运行（不区分路径）
func processExistsByName(exeName string) bool {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return false
	}

	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(name, exeName) {
			return true
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}
	return false
}

// processExistsByPath 检查指定路径的进程是否正在运行
func processExistsByPath(targetPath string) (bool, error) {
	targetPath = filepath.Clean(targetPath)

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false, err
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return false, nil
	}

	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(name, filepath.Base(targetPath)) {
			match, err := processPathMatches(entry.ProcessID, targetPath)
			if err == nil && match {
				return true, nil
			}
		}
		err = windows.Process32Next(snapshot, &entry)
		if err != nil {
			break
		}
	}
	return false, nil
}

func queryFullProcessImageName(handle windows.Handle) (string, error) {
	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	ret, _, err := procQueryFullProcessImageName.Call(
		uintptr(handle),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return "", err
	}
	return windows.UTF16ToString(buf[:size]), nil
}
