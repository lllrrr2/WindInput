@echo off
setlocal enabledelayedexpansion

echo ======================================
echo WindInput - Build All
echo ======================================
echo.

REM Get script directory
set SCRIPT_DIR=%~dp0

echo [1/3] Building Go service (wind_input.exe)...
cd "%SCRIPT_DIR%wind_input"
go build -o ../build/wind_input.exe ./cmd/service
if %errorLevel% neq 0 (
    echo [ERROR] Go build failed
    pause
    exit /b 1
)
echo Go service built successfully
echo.

echo [2/3] Building C++ DLL (wind_tsf.dll)...
cd "%SCRIPT_DIR%wind_tsf\build"
if not exist "%SCRIPT_DIR%wind_tsf\build" (
    mkdir "%SCRIPT_DIR%wind_tsf\build"
    cd "%SCRIPT_DIR%wind_tsf\build"
    cmake ..
)
cmake --build . --config Release
if %errorLevel% neq 0 (
    echo [ERROR] C++ build failed
    pause
    exit /b 1
)
echo C++ DLL built successfully
echo.

echo [3/3] Checking output files...
if not exist "%SCRIPT_DIR%build\wind_tsf.dll" (
    echo [ERROR] wind_tsf.dll not found
    pause
    exit /b 1
)

if not exist "%SCRIPT_DIR%build\wind_input.exe" (
    echo [ERROR] wind_input.exe not found
    pause
    exit /b 1
)

echo.
echo ======================================
echo Build Complete!
echo ======================================
echo.
echo Output files:
echo - build\wind_tsf.dll (TSF Bridge)
echo - build\wind_input.exe (IME Service with Native UI)
echo.
echo Next step: Run installer\install.bat as Administrator
echo.

