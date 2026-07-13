@echo off
REM run_unit_tests.bat
REM Runs unit tests only (no build tags, no external dependencies).

echo Running unit tests...
echo.

go test -v -cover .

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo ==============================
    echo  Unit tests FAILED
    echo ==============================
    pause
    exit /b %ERRORLEVEL%
) else (
    echo.
    echo ==============================
    echo  Unit tests PASSED
    echo ==============================
    pause
)