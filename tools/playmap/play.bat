@echo off
rem Tiled F5 / Shift+F5: play the map that is open in Tiled.
rem   play.bat [-noencounter] <map file path>
rem -noencounter (F5) = no random enemy encounters while walking.
cd /d "%~dp0..\.."
where go >nul 2>nul
if errorlevel 1 (
  echo Go is not installed. Install it from https://go.dev/dl/ and try again.
  pause
  exit /b 1
)
set EXTRA=
if /i "%~1"=="-noencounter" (
  set EXTRA=-noencounter
  shift
)
go run . -map "%~1" %EXTRA%
