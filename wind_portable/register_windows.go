//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/huanfeng/wind_input/pkg/buildvariant"
	"golang.org/x/sys/windows/registry"
)

const (
	ilotUninstall = 0x00000001
)

// registerInputMethod 注册输入法，非管理员时自动通过 UAC 提权
func (m *launcherManager) registerInputMethod() error {
	if err := m.ensurePortableAvailable("注册输入法"); err != nil {
		return err
	}
	if err := m.ensurePortableLayout(); err != nil {
		return err
	}
	if m.cfg.TsfDll == "" {
		return fmt.Errorf("未找到 TSF DLL，请先构建 %s", dllName)
	}

	if isAdmin() {
		return m.registerInputMethodDirect()
	}
	// 非管理员：通过 UAC 启动提权子进程完成注册
	return runElevated("-elevate-register")
}

// unregisterInputMethod 注销输入法，非管理员时自动通过 UAC 提权
func (m *launcherManager) unregisterInputMethod() error {
	if err := m.ensurePortableAvailable("注销输入法"); err != nil {
		return err
	}

	if isAdmin() {
		return m.unregisterInputMethodDirect()
	}
	return runElevated("-elevate-unregister")
}

// registerInputMethodDirect 直接注册（需管理员权限）
func (m *launcherManager) registerInputMethodDirect() error {
	if m.cfg.TsfDll == "" {
		return fmt.Errorf("未找到 TSF DLL，请先构建 %s", dllName)
	}

	if err := grantAppPackagesReadExec(m.cfg.TsfDll); err != nil {
		return err
	}
	if err := regsvr32Register(m.cfg.TsfDll, false); err != nil {
		return err
	}

	if m.cfg.TsfDllX86 != "" {
		_ = grantAppPackagesReadExec(m.cfg.TsfDllX86)
		_ = regsvr32Register(m.cfg.TsfDllX86, true)
	}

	if err := installLayoutOrTip(profileStr, 0); err != nil {
		return err
	}
	return nil
}

// unregisterInputMethodDirect 直接注销（需管理员权限）
func (m *launcherManager) unregisterInputMethodDirect() error {
	if profileStr != "" {
		_ = installLayoutOrTip(profileStr, ilotUninstall)
	}
	if m.cfg.TsfDllX86 != "" {
		_ = regsvr32Unregister(m.cfg.TsfDllX86, true)
	}
	if m.cfg.TsfDll != "" {
		_ = regsvr32Unregister(m.cfg.TsfDll, false)
	}
	return nil
}

func (m *launcherManager) isRegistered() bool {
	path, _ := m.registeredDLLPath()
	return path != "" && samePath(path, m.cfg.TsfDll)
}

func (m *launcherManager) installedConflict() (bool, string) {
	// 检查 1：当前目录是否就是 NSIS 安装目录
	if installDir := m.nsisInstallLocation(); installDir != "" {
		if samePath(m.cfg.RootDir, installDir) {
			return true, "当前位于已安装目录，便携模式不可用。如需使用便携模式，请将文件复制到其他目录运行。"
		}
	}

	// 检查 2：系统是否已注册其他位置的 DLL
	path, err := m.registeredDLLPath()
	if err != nil || path == "" {
		return false, ""
	}
	if samePath(path, m.cfg.TsfDll) {
		return false, ""
	}
	return true, fmt.Sprintf("系统已注册其他位置的清风输入法：%s。为避免覆盖现有注册信息，便携模式已禁用。", path)
}

func (m *launcherManager) installedConflictPath() string {
	if installDir := m.nsisInstallLocation(); installDir != "" {
		if samePath(m.cfg.RootDir, installDir) {
			return installDir
		}
	}
	path, _ := m.registeredDLLPath()
	return path
}

// nsisInstallLocation 读取 NSIS 安装器写入的 InstallLocation 注册表值
func (m *launcherManager) nsisInstallLocation() string {
	uninstKey := `Software\Microsoft\Windows\CurrentVersion\Uninstall\` + m.nsisDisplayName()
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, uninstKey, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	val, _, err := k.GetStringValue("InstallLocation")
	if err != nil {
		return ""
	}
	return stringsTrimSpace(val)
}

func (m *launcherManager) nsisDisplayName() string {
	return buildvariant.DisplayName()
}

func (m *launcherManager) registeredDLLPath() (string, error) {
	if m.cfg.TsfDll == "" {
		return "", nil
	}
	const clsid = "{99C2EE30-5C57-45A2-9C63-FB54B34FD90A}"
	const clsidDebug = "{99C2DEB0-5C57-45A2-9C63-FB54B34FD90A}"

	clsidValue := clsid
	if buildvariant.IsDebug() {
		clsidValue = clsidDebug
	}

	candidates := []struct {
		root registry.Key
		path string
	}{
		{root: registry.CURRENT_USER, path: `Software\Classes\CLSID\` + clsidValue + `\InprocServer32`},
		{root: registry.LOCAL_MACHINE, path: `Software\Classes\CLSID\` + clsidValue + `\InprocServer32`},
		{root: registry.CLASSES_ROOT, path: `CLSID\` + clsidValue + `\InprocServer32`},
	}

	for _, candidate := range candidates {
		k, err := registry.OpenKey(candidate.root, candidate.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		val, _, err := k.GetStringValue("")
		_ = k.Close()
		if err == nil && stringsTrimSpace(val) != "" {
			return val, nil
		}
	}
	return "", nil
}

func regsvr32Register(dllPath string, x86 bool) error {
	return runRegsvr32(dllPath, x86, false)
}

func regsvr32Unregister(dllPath string, x86 bool) error {
	return runRegsvr32(dllPath, x86, true)
}

func runRegsvr32(dllPath string, x86, unregister bool) error {
	if dllPath == "" {
		return nil
	}
	if _, err := os.Stat(dllPath); err != nil {
		return fmt.Errorf("未找到 DLL: %s", dllPath)
	}

	var args []string
	if unregister {
		args = []string{"/u", "/s", dllPath}
	} else {
		args = []string{"/s", dllPath}
	}

	cmd := exec.Command(regsvr32Path(x86), args...)
	cmd.SysProcAttr = defaultHiddenProcessAttrs()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("regsvr32 执行失败 (%s): %w %s", filepath.Base(dllPath), err, stringsTrimSpace(string(out)))
	}
	return nil
}

func regsvr32Path(x86 bool) string {
	if x86 {
		return filepath.Join(os.Getenv("SystemRoot"), "SysWOW64", "regsvr32.exe")
	}
	return "regsvr32.exe"
}

func grantAppPackagesReadExec(path string) error {
	cmd := exec.Command("icacls", path, "/grant", "*S-1-15-2-1:(RX)", "/c")
	cmd.SysProcAttr = defaultHiddenProcessAttrs()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("设置 DLL 权限失败: %w %s", err, stringsTrimSpace(string(out)))
	}
	return nil
}

func samePath(a, b string) bool {
	return filepath.Clean(stringsToLower(a)) == filepath.Clean(stringsToLower(b))
}
