param(
    [Parameter(Position=0)]
    [string]$Choice = ""
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$ScriptPath = $MyInvocation.MyCommand.Definition

function Ensure-Admin {
    $isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    if (-not $isAdmin) {
        Write-Host "[INFO] 需要管理员权限，正在请求提升..."
        # 提权窗口中用 & 执行脚本，完成后 pause 防止窗口闪退
        $cmd = "& '$ScriptPath' -Choice '$Choice'; Write-Host ''; Write-Host '按任意键关闭窗口...'; `$null = `$Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')"
        Start-Process powershell -Verb RunAs -ArgumentList "-NoProfile -ExecutionPolicy Bypass -Command `"$cmd`""
        exit 0
    }
}

# ============ 正式版操作 ============

function Do-BuildRelease {
    & "$ScriptDir\build_all.ps1" -WailsMode release
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-BuildDebug {
    & "$ScriptDir\build_all.ps1" -WailsMode debug
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-Install {
    & "$ScriptDir\installer\install.ps1"
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-Uninstall {
    & "$ScriptDir\installer\uninstall.ps1"
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-BuildInstaller {
    & "$ScriptDir\installer\build_nsis.ps1"
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-BuildInstallerSkip {
    & "$ScriptDir\installer\build_nsis.ps1" -SkipBuild
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-BuildModule {
    param([string[]]$Modules)
    & "$ScriptDir\build_all.ps1" -Module $Modules -WailsMode release
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-DeployModule {
    param([string[]]$Modules)
    & "$ScriptDir\installer\install.ps1" -Module $Modules
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

# ============ 调试版操作 ============

function Do-BuildReleaseDebugVariant {
    & "$ScriptDir\build_all.ps1" -WailsMode release -DebugVariant
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-BuildDebugDebugVariant {
    & "$ScriptDir\build_all.ps1" -WailsMode debug -DebugVariant
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-InstallDebugVariant {
    & "$ScriptDir\installer\install.ps1" -DebugVariant
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-UninstallDebugVariant {
    & "$ScriptDir\installer\uninstall.ps1" -DebugVariant
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-BuildModuleDebugVariant {
    param([string[]]$Modules)
    & "$ScriptDir\build_all.ps1" -Module $Modules -WailsMode release -DebugVariant
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

function Do-DeployModuleDebugVariant {
    param([string[]]$Modules)
    & "$ScriptDir\installer\install.ps1" -Module $Modules -DebugVariant
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

# ============ 交互式菜单 ============

if (-not $Choice) {
    Write-Host "======================================"
    Write-Host "WindInput - Dev Menu"
    Write-Host "======================================"
    Write-Host ""
    Write-Host "  --- 正式版 ---"
    Write-Host "  1.  卸载 / 构建(Release) / 安装"
    Write-Host "  2.  卸载 / 构建(Debug) / 安装"
    Write-Host "  3.  构建(Release)"
    Write-Host "  4.  构建(Debug)"
    Write-Host "  5.  安装"
    Write-Host "  6.  卸载"
    Write-Host "  7.  卸载 / 安装"
    Write-Host "  8.  生成安装包(Release)"
    Write-Host "  9.  生成安装包(跳过编译)"
    Write-Host ""
    Write-Host "  --- 模块快速部署（需已完整安装）---"
    Write-Host "  m1.   [TSF DLL]          构建 + 部署"
    Write-Host "  m2.   [GO 服务]          构建 + 部署"
    Write-Host "  m3.   [设置]             构建 + 部署"
    Write-Host "  m12.  [TSF DLL+GO 服务]  构建 + 部署"
    Write-Host "  m13.  [TSF DLL+设置]     构建 + 部署"
    Write-Host "  m23.  [GO 服务+设置]     构建 + 部署"
    Write-Host "  m123. [全部模块]         构建 + 部署"
    Write-Host ""
    Write-Host "  --- 调试版 ---"
    Write-Host "  d1. 卸载 / 构建(Release) / 安装"
    Write-Host "  d2. 卸载 / 构建(Debug) / 安装"
    Write-Host "  d3. 构建(Release)"
    Write-Host "  d4. 构建(Debug)"
    Write-Host "  d5. 安装"
    Write-Host "  d6. 卸载"
    Write-Host "  d7. 卸载 / 安装"
    Write-Host ""
    Write-Host "  --- 调试版模块快速部署（需已完整安装）---"
    Write-Host "  dm1.   [TSF DLL]          构建 + 部署"
    Write-Host "  dm2.   [GO 服务]          构建 + 部署"
    Write-Host "  dm3.   [设置]             构建 + 部署"
    Write-Host "  dm12.  [TSF DLL+GO 服务]  构建 + 部署"
    Write-Host "  dm13.  [TSF DLL+设置]     构建 + 部署"
    Write-Host "  dm23.  [GO 服务+设置]     构建 + 部署"
    Write-Host "  dm123. [全部模块]         构建 + 部署"
    Write-Host ""
    $Choice = Read-Host "请选择"
    if (-not $Choice) { exit 1 }
}

switch ($Choice) {
    # 正式版
    "1"  { Ensure-Admin; Do-Uninstall; Do-BuildRelease; Do-Install }
    "2"  { Ensure-Admin; Do-Uninstall; Do-BuildDebug; Do-Install }
    "3"  { Do-BuildRelease }
    "4"  { Do-BuildDebug }
    "5"  { Ensure-Admin; Do-Install }
    "6"  { Ensure-Admin; Do-Uninstall }
    "7"  { Ensure-Admin; Do-Uninstall; Do-Install }
    "8"  { Do-BuildInstaller }
    "9"  { Do-BuildInstallerSkip }
    # 模块快速部署（正式版）
    { $_ -match '^m[123]+$' } {
        Ensure-Admin
        $digits = $_ -replace '^m', ''
        $modules = @()
        if ($digits -match '1') { $modules += "dll" }
        if ($digits -match '2') { $modules += "service" }
        if ($digits -match '3') { $modules += "setting" }
        Do-BuildModule -Modules $modules
        Do-DeployModule -Modules $modules
    }

    # 调试版
    "d1" { Ensure-Admin; Do-UninstallDebugVariant; Do-BuildReleaseDebugVariant; Do-InstallDebugVariant }
    "d2" { Ensure-Admin; Do-UninstallDebugVariant; Do-BuildDebugDebugVariant; Do-InstallDebugVariant }
    "d3" { Do-BuildReleaseDebugVariant }
    "d4" { Do-BuildDebugDebugVariant }
    "d5" { Ensure-Admin; Do-InstallDebugVariant }
    "d6" { Ensure-Admin; Do-UninstallDebugVariant }
    "d7" { Ensure-Admin; Do-UninstallDebugVariant; Do-InstallDebugVariant }

    # 模块快速部署（调试版）
    { $_ -match '^dm[123]+$' } {
        Ensure-Admin
        $digits = $_ -replace '^dm', ''
        $modules = @()
        if ($digits -match '1') { $modules += "dll" }
        if ($digits -match '2') { $modules += "service" }
        if ($digits -match '3') { $modules += "setting" }
        Do-BuildModuleDebugVariant -Modules $modules
        Do-DeployModuleDebugVariant -Modules $modules
    }

    default {
        Write-Host "[ERROR] 无效选项: $Choice" -ForegroundColor Red
        exit 1
    }
}
