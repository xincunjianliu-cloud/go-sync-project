@echo off
rem Import the dialogues from Google Sheets, refresh rpg.tiled-project's lists, then run the checks.
cd /d "%~dp0"
echo [1/3] import dialogues
go run ./tools/dialoguegen || goto fail
echo [2/3] Tiled lists
set UPDATE_TILED_PROJECT=1
go test -count=1 -run TestTiledProjectUpToDate . >nul || goto fail
set UPDATE_TILED_PROJECT=
echo [3/3] checks
go test -count=1 ./... || goto fail
echo.
echo ==== OK ====
echo Commit and Push with GitHub Desktop.
pause
exit /b 0
:fail
echo.
echo ==== FAILED ====
echo Nothing is broken yet. In GitHub Desktop, right-click the changed files and choose "Discard changes".
pause
exit /b 1
