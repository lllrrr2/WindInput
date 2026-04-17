//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/huanfeng/wind_input/pkg/buildvariant"
	"github.com/huanfeng/wind_input/pkg/config"
	"github.com/huanfeng/wind_input/pkg/rpcapi"
)

// 文件名由编译期 buildvariant 静态决定
var (
	serviceName = "wind_input" + buildvariant.Suffix() + ".exe"
	settingName = "wind_setting" + buildvariant.Suffix() + ".exe"
	dllName     = "wind_tsf" + buildvariant.Suffix() + ".dll"
	dllNameX86  = "wind_tsf" + buildvariant.Suffix() + "_x86.dll"
	profileStr  = func() string {
		if buildvariant.IsDebug() {
			return "0804:{99C2DEB0-5C57-45A2-9C63-FB54B34FD90A}{99C2DEB1-5C57-45A2-9C63-FB54B34FD90A}"
		}
		return "0804:{99C2EE30-5C57-45A2-9C63-FB54B34FD90A}{99C2EE31-5C57-45A2-9C63-FB54B34FD90A}"
	}()
)

type portableConfig struct {
	RootDir        string
	UserdataDir    string
	AppDataDir     string
	PortableMarker string
	IconPath       string
	ServiceExe     string
	SettingExe     string
	TsfDll         string
	TsfDllX86      string
}

type launcherManager struct {
	cfg    portableConfig
	client *rpcapi.Client
}

func newLauncherManager(cfg portableConfig) *launcherManager {
	return &launcherManager{
		cfg:    cfg,
		client: rpcapi.NewClient(), // 管道名由 buildvariant.Suffix() 自动决定
	}
}

func (m *launcherManager) serviceRunning() bool {
	return m.client.IsAvailable()
}

func (m *launcherManager) startService() error {
	if err := m.ensurePortableAvailable("启动服务"); err != nil {
		return err
	}
	if m.serviceRunning() {
		return nil
	}
	if err := m.ensurePortableLayout(); err != nil {
		return err
	}
	if !m.isRegistered() {
		if err := m.registerInputMethod(); err != nil {
			return fmt.Errorf("注册输入法失败: %w", err)
		}
	}
	if _, err := os.Stat(m.cfg.ServiceExe); err != nil {
		return fmt.Errorf("未找到服务程序: %s", m.cfg.ServiceExe)
	}

	cmd := exec.Command(m.cfg.ServiceExe)
	cmd.Dir = filepath.Dir(m.cfg.ServiceExe)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动服务失败: %w", err)
	}
	return nil
}

func (m *launcherManager) stopService() (bool, error) {
	if err := m.ensurePortableAvailable("停止服务"); err != nil {
		return false, err
	}
	wasRunning := m.serviceRunning()
	wasRegistered := m.isRegistered()

	if wasRunning {
		graceful := false
		if err := m.client.SystemShutdown(); err == nil {
			for i := 0; i < 6; i++ {
				time.Sleep(500 * time.Millisecond)
				if !m.serviceRunning() {
					graceful = true
					break
				}
			}
		}
		if !graceful && m.serviceRunning() {
			if _, err := terminateProcessByPath(m.cfg.ServiceExe); err != nil {
				return false, err
			}
		}
	}
	if wasRegistered {
		if err := m.unregisterInputMethod(); err != nil {
			return false, fmt.Errorf("注销输入法失败: %w", err)
		}
	}
	return wasRunning || wasRegistered, nil
}

func (m *launcherManager) openSettings() error {
	if err := m.ensurePortableAvailable("打开设置"); err != nil {
		return err
	}
	if _, err := os.Stat(m.cfg.SettingExe); err != nil {
		return fmt.Errorf("未找到设置程序: %s", m.cfg.SettingExe)
	}
	cmd := exec.Command(m.cfg.SettingExe)
	cmd.Dir = filepath.Dir(m.cfg.SettingExe)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("打开设置失败: %w", err)
	}
	return nil
}

func (m *launcherManager) openUserdataDir() error {
	if err := m.ensurePortableAvailable("打开 userdata"); err != nil {
		return err
	}
	target := m.cfg.AppDataDir
	if _, err := os.Stat(target); os.IsNotExist(err) {
		target = m.cfg.UserdataDir
	}
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return fmt.Errorf("userdata 目录尚未创建，请先启动一次服务: %s", m.cfg.UserdataDir)
	}
	cmd := exec.Command("explorer.exe", target)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("打开 userdata 目录失败: %w", err)
	}
	return nil
}

// isProtectedDir 检查路径是否在 Windows 系统保护目录下
func isProtectedDir(dir string) bool {
	lower := strings.ToLower(filepath.Clean(dir))
	protectedPrefixes := []string{
		strings.ToLower(os.Getenv("ProgramFiles")),
		strings.ToLower(os.Getenv("ProgramFiles(x86)")),
		strings.ToLower(os.Getenv("ProgramW6432")),
		strings.ToLower(os.Getenv("SystemRoot")),
	}
	for _, prefix := range protectedPrefixes {
		if prefix != "" && strings.HasPrefix(lower, prefix+`\`) {
			return true
		}
	}
	return false
}

func detectPortableConfig() (portableConfig, error) {
	exePath, err := os.Executable()
	if err != nil {
		return portableConfig{}, fmt.Errorf("无法获取当前程序路径: %w", err)
	}

	exeDir := filepath.Dir(exePath)

	if isProtectedDir(exeDir) {
		return portableConfig{}, fmt.Errorf("当前位于系统保护目录(%s)，不支持便携模式。\n请将便携包复制到其他目录运行。", exeDir)
	}

	wd, _ := os.Getwd()

	rootCandidates := uniquePaths([]string{
		exeDir,
		filepath.Dir(exeDir),
		wd,
		filepath.Dir(wd),
	})

	for _, root := range rootCandidates {
		svcExe := firstExistingPath([]string{
			filepath.Join(root, serviceName),
			filepath.Join(root, "build", serviceName),
			filepath.Join(root, "build_debug", serviceName),
		})
		if svcExe == "" {
			continue
		}

		setExe := firstExistingPath([]string{
			filepath.Join(root, settingName),
			filepath.Join(root, "build", settingName),
			filepath.Join(root, "build_debug", settingName),
		})
		if setExe == "" {
			setExe = filepath.Join(root, settingName)
		}

		tsfDll := firstExistingPath([]string{
			filepath.Join(root, dllName),
			filepath.Join(root, "build", dllName),
			filepath.Join(root, "build_debug", dllName),
		})
		tsfDllX86 := firstExistingPath([]string{
			filepath.Join(root, dllNameX86),
			filepath.Join(root, "build", dllNameX86),
			filepath.Join(root, "build_debug", dllNameX86),
		})

		userdataDir := filepath.Join(root, config.PortableDataDir)
		return portableConfig{
			RootDir:        root,
			UserdataDir:    userdataDir,
			AppDataDir:     userdataDir, // 便携模式下 userdata 直接就是用户数据根目录
			PortableMarker: filepath.Join(root, config.PortableMarkerName),
			IconPath: firstExistingPath([]string{
				filepath.Join(root, "wind_portable", "res", "wind_input_portable.ico"),
				filepath.Join(root, "res", "wind_input_portable.ico"),
				filepath.Join(root, "wind_tsf", "res", "wind_input.ico"),
			}),
			ServiceExe: svcExe,
			SettingExe: setExe,
			TsfDll:     tsfDll,
			TsfDllX86:  tsfDllX86,
		}, nil
	}

	return portableConfig{}, fmt.Errorf("未找到 %s，请先构建主服务或将 launcher 放到打包目录中", serviceName)
}

func firstExistingPath(paths []string) string {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	ret := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		clean := filepath.Clean(p)
		key := strings.ToLower(clean)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ret = append(ret, clean)
	}
	return ret
}

func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("创建目录失败 %s: %w", path, err)
	}
	return nil
}

func (m *launcherManager) ensurePortableAvailable(action string) error {
	if conflict, reason := m.installedConflict(); conflict {
		return fmt.Errorf("%s失败：%s", action, reason)
	}
	return nil
}

func (m *launcherManager) ensurePortableLayout() error {
	for _, dir := range []string{
		m.cfg.UserdataDir,
		m.cfg.AppDataDir,
		filepath.Join(m.cfg.AppDataDir, "logs"),
		filepath.Join(m.cfg.AppDataDir, "cache"),
		filepath.Join(m.cfg.AppDataDir, "themes"),
	} {
		if err := ensureDir(dir); err != nil {
			return err
		}
	}
	if _, err := os.Stat(m.cfg.PortableMarker); os.IsNotExist(err) {
		if err := os.WriteFile(m.cfg.PortableMarker, []byte("wind_portable=1\n"), 0o644); err != nil {
			return fmt.Errorf("写入 portable 标记失败: %w", err)
		}
	}
	return nil
}
