@echo off
setlocal

cd /d "%~dp0"

where node >nul 2>nul
if errorlevel 1 (
  echo.
  echo [ERROR] Node.js was not found.
  echo Install Node.js, reopen this file, and try again.
  echo https://nodejs.org/
  echo.
  pause
  exit /b 1
)

echo Starting SpeakUp Practice Edition...
echo Open: http://127.0.0.1:18086/practice
echo Press Ctrl+C to stop the server.
echo.

start "" powershell.exe -NoProfile -WindowStyle Hidden -Command "Start-Sleep -Seconds 2; Start-Process 'http://127.0.0.1:18086/practice'"
node server.mjs

if errorlevel 1 (
  echo.
  echo The server stopped with an error.
  pause
)

endlocal
