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

echo [1/7] Checking files...
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

echo [2/7] Stopping old processes...
taskkill /F /IM wind_input.exe >nul 2>&1
timeout /t 1 /nobreak >nul

echo [3/7] Creating install directory...
echo [4/7] Handling existing files...
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

echo [5/7] Copying files...
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

echo [6/7] Copying dictionaries from build directory...
REM Create dict directories
if not exist "%INSTALL_DIR%\dict\pinyin" mkdir "%INSTALL_DIR%\dict\pinyin"
if not exist "%INSTALL_DIR%\dict\wubi" mkdir "%INSTALL_DIR%\dict\wubi"

REM Copy pinyin dictionary from build
if exist "%BUILD_DIR%\dict\pinyin\pinyin.txt" (
    copy /Y "%BUILD_DIR%\dict\pinyin\pinyin.txt" "%INSTALL_DIR%\dict\pinyin\pinyin.txt" >nul
    echo   - Pinyin dictionary: pinyin.txt
) else (
    echo [WARN] Pinyin dictionary not found in build directory
    echo        Please run build_all.bat first
)

REM Copy wubi dictionary from build
if exist "%BUILD_DIR%\dict\wubi\wubi86.txt" (
    copy /Y "%BUILD_DIR%\dict\wubi\wubi86.txt" "%INSTALL_DIR%\dict\wubi\wubi86.txt" >nul
    echo   - Wubi dictionary: wubi86.txt
) else (
    echo [WARN] Wubi dictionary not found in build directory
    echo        Please run build_all.bat first
)

echo [7/7] Registering COM component...
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
echo - dict\pinyin\pinyin.txt (Pinyin dictionary)
echo - dict\wubi\wubi86.txt (Wubi86 dictionary)
echo.
echo The service will start automatically when you use the IME.
echo.
echo Usage:
echo 1. Press Win+Space to switch input method
echo 2. Select "WindInput" from the input method list
echo 3. Start typing (default: Pinyin mode)
echo.
echo Hotkeys:
echo - Shift: Toggle Chinese/English mode
echo - Ctrl+`: Switch between Pinyin and Wubi engine
echo.
echo Config location: %%APPDATA%%\WindInput\config.yaml
echo.
echo Note: If old files could not be deleted, restart your
echo computer and run this installer again to clean them up.
echo.
pause