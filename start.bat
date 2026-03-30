@echo off
chcp 65001 > nul
cd /d "%~dp0"
echo Building...
go build -o server.exe .\cmd\server\
echo Starting server at http://localhost:8080
server.exe serve
pause
