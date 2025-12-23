# Combined Development Script
# Starts both Backend (with GeoIP) and Frontend (Vite) in separate windows

$Root = $PSScriptRoot

# 1. Start Backend
$BackendDir = Join-Path $Root "backend"
$BackendScript = Join-Path $BackendDir "run_dev.ps1"

Write-Host "Starting Backend..." -ForegroundColor Cyan
if (Test-Path $BackendScript) {
    # Start in a new window to keep logs separate
    Start-Process powershell -ArgumentList "-NoExit", "-File", "`"$BackendScript`"" -WorkingDirectory $BackendDir
} else {
    Write-Error "Backend script not found at $BackendScript"
}

# 2. Start Frontend
$FrontendDir = Join-Path $Root "frontend/system-monitor-web"

Write-Host "Starting Frontend..." -ForegroundColor Cyan
if (Test-Path $FrontendDir) {
    # Start in a new window
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "npm run dev" -WorkingDirectory $FrontendDir
} else {
    Write-Error "Frontend directory not found at $FrontendDir"
}

Write-Host "All services started in new windows." -ForegroundColor Green
