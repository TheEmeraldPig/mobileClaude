@echo off
rem Double-click this file to start the Day-Trading Advisor on Windows.
cd /d "%~dp0"
where py >nul 2>nul
if %errorlevel%==0 (py app.py) else (python app.py)
echo.
echo The advisor has stopped. If you saw an error above, see README.md.
pause
