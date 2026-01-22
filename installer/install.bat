@echo off
setlocal enabledelayedexpansion

echo ======================================
echo WindInput IME Installer
echo ======================================
echo.

REM Check administrator privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo [ERROR] Please run this script as Administrator!
    echo Right-click this file and select "Run as administrator"
    pause
    exit /b 1
)

REM Get script directory
set SCRIPT_DIR=%~dp0
set BUILD_DIR=%SCRIPT_DIR%..\build
set DICT_DIR=%SCRIPT_DIR%..\dict

echo [1/6] Checking files...
if not exist "%BUILD_DIR%\wind_tsf.dll" (
    echo [ERROR] wind_tsf.dll not found
    echo Please run build_all.bat first
    pause
    exit /b 1
)

if not exist "%BUILD_DIR%\wind_input.exe" (
    echo [ERROR] wind_input.exe not found
    echo Please run build_all.bat first
    pause
    exit /b 1
)

echo [2/6] Stopping old processes...
taskkill /F /IM wind_input.exe >nul 2>&1
timeout /t 1 /nobreak >nul

echo [3/6] Creating install directory...
set INSTALL_DIR=%ProgramFiles%\WindInput
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

REM Generate random suffix based on time
set RANDOM_SUFFIX=%TIME:~6,2%%TIME:~9,2%%RANDOM%

REM Handle old DLL - try to delete, if fails rename it
if exist "%INSTALL_DIR%\wind_tsf.dll" (
    REM First unregister the old DLL
    regsvr32 /u /s "%INSTALL_DIR%\wind_tsf.dll" >nul 2>&1

    del /F "%INSTALL_DIR%\wind_tsf.dll" >nul 2>&1
    if exist "%INSTALL_DIR%\wind_tsf.dll" (
        echo [WARN] Cannot delete old wind_tsf.dll, renaming to wind_tsf.dll.old_%RANDOM_SUFFIX%
        ren "%INSTALL_DIR%\wind_tsf.dll" "wind_tsf.dll.old_%RANDOM_SUFFIX%" >nul 2>&1
        if exist "%INSTALL_DIR%\wind_tsf.dll" (
            echo [WARN] Cannot rename either, trying alternative name...
            ren "%INSTALL_DIR%\wind_tsf.dll" "wind_tsf_%RANDOM_SUFFIX%.dll.bak" >nul 2>&1
        )
    )
)

REM Handle old wind_input.exe
if exist "%INSTALL_DIR%\wind_input.exe" (
    del /F "%INSTALL_DIR%\wind_input.exe" >nul 2>&1
    if exist "%INSTALL_DIR%\wind_input.exe" (
        echo [WARN] Cannot delete old wind_input.exe, renaming to wind_input.exe.old_%RANDOM_SUFFIX%
        ren "%INSTALL_DIR%\wind_input.exe" "wind_input.exe.old_%RANDOM_SUFFIX%" >nul 2>&1
    )
)

echo [5/6] Copying files...
copy /Y "%BUILD_DIR%\wind_tsf.dll" "%INSTALL_DIR%\" >nul
if %errorLevel% neq 0 (
    echo [ERROR] Failed to copy wind_tsf.dll
    pause
    exit /b 1
)

copy /Y "%BUILD_DIR%\wind_input.exe" "%INSTALL_DIR%\" >nul
if %errorLevel% neq 0 (
    echo [ERROR] Failed to copy wind_input.exe
    pause
    exit /b 1
)

xcopy /Y /E /I "%DICT_DIR%" "%INSTALL_DIR%\dict\" >nul 2>&1

echo [6/6] Registering COM component...
regsvr32 /s "%INSTALL_DIR%\wind_tsf.dll"
if %errorLevel% neq 0 (
    echo [ERROR] COM registration failed
    pause
    exit /b 1
)

REM Clean up old backup files (try to delete .old_* and .bak files)
echo.
echo Cleaning up old backup files...
for %%f in ("%INSTALL_DIR%\*.old_*") do (
    del /F "%%f" >nul 2>&1
)
for %%f in ("%INSTALL_DIR%\*.bak") do (
    del /F "%%f" >nul 2>&1
)

echo.
echo ======================================
echo Installation Complete!
echo ======================================
echo.
echo Install directory: %INSTALL_DIR%
echo.
echo Components installed:
echo - wind_tsf.dll (TSF Bridge)
echo - wind_input.exe (IME Service)
echo - dict\ (Dictionary files)
echo.
echo The service will start automatically when you use the IME.
echo.
echo Usage:
echo 1. Press Win+Space to switch input method
echo 2. Select "WindInput" from the input method list
echo 3. Start typing pinyin (e.g., ni, hao, zhongguo)
echo.
echo Note: If old files could not be deleted, restart your
echo computer and run this installer again to clean them up.
echo.
