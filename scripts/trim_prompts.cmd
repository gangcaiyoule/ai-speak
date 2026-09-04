@echo off
setlocal
set "SCRIPT=%~dp0trim_prompts.py"
if "%~1"=="" (
  set /p "TARGET=请输入提示词文件或目录路径："
) else (
  set "TARGET=%~1"
)
if "%TARGET%"=="" (
  echo 未提供路径。
  pause
  exit /b 2
)
python "%SCRIPT%" "%TARGET%"
if errorlevel 1 pause
