@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0tools\mapsync\mapsync.ps1" send
pause
