@echo off
setlocal EnableExtensions

rem Resolve the repository root even when double-clicked from test/.
for %%I in ("%~dp0..") do set "ROOT=%%~fI"
set "BACKEND_DIR=%ROOT%\backend"
set "FRONTEND_DIR=%ROOT%\fronted"
set "BIN_DIR=%~dp0bin"
if "%BIN_DIR:~-1%"=="\" set "BIN_DIR=%BIN_DIR:~0,-1%"
set "NPM_CACHE_DIR=%BIN_DIR%\npm-cache"
set "BACKEND_EXE=%BIN_DIR%\submerge.exe"

echo.
echo === SubMerge local test runner ===
echo Root: %ROOT%

where go >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Go was not found in PATH.
  goto :fail
)

where npm >nul 2>nul
if errorlevel 1 (
  echo [ERROR] npm was not found in PATH.
  goto :fail
)

if not exist "%BACKEND_DIR%\go.mod" (
  echo [ERROR] Backend project was not found: %BACKEND_DIR%
  goto :fail
)

if not exist "%FRONTEND_DIR%\package.json" (
  echo [ERROR] Frontend project was not found: %FRONTEND_DIR%
  goto :fail
)

if not exist "%ROOT%\VERSION" (
  echo [ERROR] Version file was not found: %ROOT%\VERSION
  goto :fail
)

call :check_port 8080
if errorlevel 1 goto :port_in_use
call :check_port 4202
if errorlevel 1 goto :port_in_use

if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"

if not exist "%FRONTEND_DIR%\node_modules\.bin\vite.cmd" (
  echo Installing frontend dependencies...
  pushd "%FRONTEND_DIR%"
  call npm ci --cache "%NPM_CACHE_DIR%"
  if errorlevel 1 (
    popd
    echo [ERROR] Frontend dependency installation failed.
    goto :fail
  )
  popd
)

echo Compiling backend...
set /p "APP_VERSION=" < "%ROOT%\VERSION"
pushd "%BACKEND_DIR%"
go build -ldflags "-X github.com/submerge/submerge/backend/version.Value=%APP_VERSION%" -o "%BACKEND_EXE%" .
set "BUILD_RESULT=%ERRORLEVEL%"
popd
if not "%BUILD_RESULT%"=="0" (
  echo [ERROR] Backend compilation failed.
  goto :fail
)

echo Starting backend in a separate window...
start "SubMerge Backend" /D "%ROOT%" cmd /k ""%BACKEND_EXE%""

echo Starting frontend at http://localhost:4202
echo Close this window to stop the frontend. Close the "SubMerge Backend" window to stop the backend.
pushd "%FRONTEND_DIR%"
call npm start -- --open
set "FRONTEND_RESULT=%ERRORLEVEL%"
popd
if not "%FRONTEND_RESULT%"=="0" (
  echo [ERROR] Frontend server stopped with exit code %FRONTEND_RESULT%.
  goto :fail
)

endlocal
exit /b 0

:check_port
powershell -NoProfile -Command "if (Get-NetTCPConnection -LocalPort %~1 -State Listen -ErrorAction SilentlyContinue) { exit 1 }"
exit /b %ERRORLEVEL%

:port_in_use
echo [ERROR] Port 8080 or 4202 is already in use. Stop the existing service and try again.

:fail
if not defined TEST_RUN_NO_PAUSE pause
endlocal
exit /b 1
