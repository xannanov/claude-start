@echo off
cd /d "%~dp0"

echo Building...
go build -o server.exe .\cmd\server\
if %ERRORLEVEL% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Init DB...
server.exe init-db
if %ERRORLEVEL% neq 0 (
    echo DB init failed!
    pause
    exit /b 1
)

echo Starting server http://localhost:8080
server.exe serve

echo Server stopped.
pause
