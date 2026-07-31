@echo off
setlocal
cd /d "%~dp0"
title Tank-OCR Local Server

echo ===================================================
echo               Tank-OCR Local Server
echo ===================================================
echo.

if not exist "%~dp0tank-ocr.exe" (
    echo [ERROR] tank-ocr.exe was not found.
    echo Run build.bat first, or download a release build.
    pause
    exit /b 1
)

:: The OCR runtime is not bundled. It is located at startup, in this order:
::   1. %%TANK_OCR_ENGINE_DIR%%
::   2. engine-runtime\ next to this file (see scripts\fetch-engine.ps1)
::   3. the Chrome / Edge profile component directory
:: If none of these has it, the server still starts and says so in its log.

echo Starting http://127.0.0.1:8000/
echo Press Ctrl+C to stop the server.
echo.

start "" powershell -NoProfile -WindowStyle Hidden -Command "Start-Sleep -Seconds 2; Start-Process 'http://127.0.0.1:8000/'"
"%~dp0tank-ocr.exe"

if %errorlevel% neq 0 (
    echo.
    echo [ERROR] Tank-OCR stopped with exit code %errorlevel%.
    pause
)

endlocal
