@echo off
chcp 65001 > nul
cd /d "%~dp0"

echo Сборка...
go build -o server.exe .\cmd\server\
if %ERRORLEVEL% neq 0 (
    echo Ошибка сборки!
    pause
    exit /b 1
)

echo Инициализация БД...
server.exe init-db
if %ERRORLEVEL% neq 0 (
    echo Ошибка инициализации БД!
    pause
    exit /b 1
)

echo Запуск сервера http://localhost:8080
server.exe serve
pause
