@echo off
setlocal
title Tank-OCR Builder

echo ===================================================
echo                Tank-OCR Builder
echo ===================================================
echo.

:: 1. Locate the Go compiler.
::    Set GO_EXE beforehand to use a portable Go installation, e.g.
::        set GO_EXE=D:\go\bin\go.exe
::    Otherwise the one on PATH is used.
if defined GO_EXE goto :have_go

where go >nul 2>nul
if %errorlevel% equ 0 (
    set "GO_EXE=go"
    goto :have_go
)

echo [ERROR] Go compiler not found.
echo.
echo Install Go 1.22 or newer from https://go.dev/dl/ , or point GO_EXE at
echo a portable copy:
echo     set GO_EXE=D:\go\bin\go.exe
echo.
pause
exit /b 1

:have_go
echo [1/3] Go compiler: %GO_EXE%
"%GO_EXE%" version
if %errorlevel% neq 0 (
    echo [ERROR] "%GO_EXE%" is not a working Go compiler.
    pause
    exit /b 1
)
echo.

:: 2. Resolve dependencies.
echo [2/3] Resolving dependencies...
"%GO_EXE%" mod tidy
if %errorlevel% neq 0 (
    echo [ERROR] Failed to resolve dependencies.
    echo If this machine is offline, run 'go mod download' on a connected
    echo machine first and copy the module cache over.
    pause
    exit /b 1
)
echo.

:: 3. Build.
echo [3/3] Compiling tank-ocr.exe ...
"%GO_EXE%" build -ldflags "-s -w" -o tank-ocr.exe .
if %errorlevel% neq 0 (
    echo [ERROR] Compilation failed.
    pause
    exit /b 1
)

echo.
echo ===================================================
echo Built tank-ocr.exe
echo Double-click run_ocr.bat to start the server.
echo ===================================================
pause
