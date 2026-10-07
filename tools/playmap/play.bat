@echo off
rem Tiled F5: play the map that is open in Tiled. %1 = map file path.
cd /d "%~dp0..\.."
where go >nul 2>nul
if errorlevel 1 (
  echo Go is not installed. Install it from https://go.dev/dl/ and try again.
  pause
  exit /b 1
)
go run . -map "%~1"