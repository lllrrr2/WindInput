<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-13 | Updated: 2026-03-23 -->

# 安装程序目录 (installer/)

## 用途

包含构建安装程序和手工安装/卸载脚本。安装程序将清风输入法的所有组件（TSF DLL、Go 服务、词库、配置文件）部署到系统目录，并配置开机自启动。

## 主要文件

| 文件 | 描述 |
|------|------|
| `build_nsis.ps1` | 构建 NSIS 安装程序：编译产物校验 + 运行 `makensis` 生成 `.exe`（已从 .bat 迁移） |
| `install.ps1` | 手工安装脚本：复制文件、注册 COM、配置启动项、启动服务（已从 .bat 迁移） |
| `uninstall.ps1` | 手工卸载脚本：停止进程、注销 COM、删除文件和注册表（已从 .bat 迁移） |
| `install.bat` | 旧版安装脚本（保留兼容） |
| `uninstall.bat` | 旧版卸载脚本（保留兼容） |

## 子目录

| 目录 | 用途 |
|-----|------|
| `nsis/` | NSIS 脚本文件（见 `nsis/AGENTS.md`） |

## 安装流程

### 自动安装（build_nsis.ps1）

```
1. 验证 NSIS 工具链（makensis.exe 在 PATH）
2. 校验编译产物存在（wind_tsf.dll、wind_dwrite.dll、wind_input.exe 等）
3. 运行 makensis 生成 清风输入法-VERSION-Setup.exe
4. 输出到 build/installer/
```

使用方式：
```powershell
# 以管理员身份在 PowerShell 中执行
installer\build_nsis.ps1 -Version 0.1.0
installer\build_nsis.ps1 -Version 0.1.0 -SkipBuild  # 跳过编译，直接生成安装程序
```

### 手工安装（install.ps1）

```
1. 检查管理员权限
2. 停止旧的 wind_input.exe 进程
3. 创建 Program Files\WindInput 目录
4. 复制 wind_tsf.dll、wind_dwrite.dll、wind_input.exe、wind_setting.exe
5. 复制词库文件（pinyin/、wubi86/、common_chars.txt）
6. 复制 schema 配置文件（schemas/）
7. 注册 wind_tsf.dll COM 组件
8. 配置开机自启动注册表项
9. 启动 wind_input.exe 后台服务
10. 创建开始菜单快捷方式
```

使用方式：
```powershell
# 以管理员身份在 PowerShell 中执行
installer\install.ps1
```

### 手工卸载

对应的卸载脚本 `uninstall.ps1` 执行反向操作：
1. 停止 wind_input.exe
2. 注销 wind_tsf.dll COM
3. 删除 Program Files\WindInput
4. 清理注册表中的自启动项
5. 删除开始菜单快捷方式

## 工作指南

### 修改安装逻辑

编辑 `install.ps1` / `uninstall.ps1` 时注意：

1. **路径处理**：使用 `$InstallDir` 变量确保兼容 32/64 位和 UAC 环境
2. **错误处理**：使用 `$ErrorActionPreference = 'Stop'` 或检查 `$LASTEXITCODE`，失败时清晰提示
3. **文件锁定**：进程占用的 DLL 无法删除，需先停止 TSF 或改名备份
4. **权限问题**：某些系统盘文件可能受保护，需 fallback 方案（改名→重启→重试）
5. **回滚安全**：保留旧文件备份（`.old_*` 后缀），防止安装中断导致系统异常
6. **wind_dwrite.dll**：安装时需同时复制 `wind_tsf.dll` 和 `wind_dwrite.dll`

### 测试安装

```powershell
# 1. 模拟手工安装
installer\install.ps1

# 2. 验证注册（COM 对象是否生效）
regsvr32 /s "$env:ProgramFiles\WindInput\wind_tsf.dll"

# 3. 检查启动项
reg query "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" | find "WindInput"

# 4. 验证 wind_dwrite.dll 已复制
Test-Path "$env:ProgramFiles\WindInput\wind_dwrite.dll"

# 5. 启动输入法
Start-Process "$env:ProgramFiles\WindInput\wind_input.exe"

# 6. 在文本框中验证输入法可用
```

### 版本号管理

`build_nsis.ps1` 从版本字符串提取数值版本号：
- 输入：`0.1.0-dev`
- 提取：`0.1.0.0`（用于 Windows 文件属性）

确保版本号遵循 `MAJOR.MINOR.PATCH[-PRERELEASE]` 格式。

## 依赖关系

### 内部

- `../build/` - 编译产物目录（wind_tsf.dll、wind_dwrite.dll、wind_input.exe、wind_setting.exe、dict/、schemas/）
- `nsis/` - NSIS 脚本文件

### 外部

- **NSIS 3.x** - 安装程序生成工具（`makensis.exe` 必须在 PATH）
- **Windows Registry** - 系统注册表用于 COM 注册和启动项配置
- **UAC (User Access Control)** - 需要管理员权限

## 常见问题

### 为什么删除 DLL 时提示"文件被占用"？

TSF 框架可能仍在使用 DLL。解决方案：
1. 先注销：`regsvr32 /u /s wind_tsf.dll`
2. 等待片刻：`timeout /t 1`
3. 尝试删除
4. 失败则改名：`ren wind_tsf.dll wind_tsf.old_XXXX`
5. 需要时重启系统

### 如何跳过编译，直接生成安装程序？

```powershell
installer\build_nsis.ps1 -Version 0.1.0 -SkipBuild
```

### 安装后输入法不生效

1. 检查 COM 注册：
   ```
   regsvr32 /s "%ProgramFiles%\WindInput\wind_tsf.dll"
   ```
2. 重启输入法服务：
   ```
   taskkill /F /IM wind_input.exe
   start "%ProgramFiles%\WindInput\wind_input.exe"
   ```
3. 重启输入法管理器：设置 → 时间和语言 → 语言与地区 → 高级 → 语言选项

<!-- MANUAL: -->
