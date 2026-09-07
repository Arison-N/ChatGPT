@echo off
setlocal
cd /d "%~dp0"

where py >nul 2>&1
if %ERRORLEVEL%==0 (
  py -3 paste_curl_download.py
  if not errorlevel 1 goto :eof
)

where python >nul 2>&1
if %ERRORLEVEL%==0 (
  python paste_curl_download.py
  if not errorlevel 1 goto :eof
)

where python3 >nul 2>&1
if %ERRORLEVEL%==0 (
  python3 paste_curl_download.py
  if not errorlevel 1 goto :eof
)

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0PasteCurlDownload.ps1"
