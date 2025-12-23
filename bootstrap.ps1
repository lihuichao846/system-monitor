param(
  [string]$FrontendPort = "8041",
  [string]$BackendPort  = "8040"
)

function Ensure-Admin {
  $id = [Security.Principal.WindowsIdentity]::GetCurrent()
  $p = New-Object Security.Principal.WindowsPrincipal($id)
  if(-not $p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { Write-Error "Run as Administrator"; exit 1 }
}

function Install-Choco {
  if(-not (Get-Command choco -ErrorAction SilentlyContinue)) {
    Set-ExecutionPolicy Bypass -Scope Process -Force
    Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
  }
}

function Install-Packages {
  # Refresh environment variables to ensure choco is found
  $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
  
  Write-Host "Installing dependencies via Chocolatey..." -ForegroundColor Cyan
  choco install -y git go nodejs-lts nssm curl
  
  # Refresh PATH again to include newly installed tools (Go, Node, etc.)
  $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
  
  # Explicitly add common paths if not present, just in case
  if ($env:Path -notlike "*C:\Program Files\Go\bin*") { $env:Path += ";C:\Program Files\Go\bin" }
  if ($env:Path -notlike "*C:\Program Files\nodejs*") { $env:Path += ";C:\Program Files\nodejs" }
  
  # Verify installations
  Write-Host "Verifying installations..." -ForegroundColor Cyan
  go version
  node --version
  npm --version
  nssm version
}

function Setup-Backend {
  $root = $PSScriptRoot
  $backendDir = Join-Path $root 'backend'
  Push-Location $backendDir
  go env -w GOPATH=$env:USERPROFILE\go
  # Ensure Go modules are enabled
  $env:GO111MODULE = "on"
  $env:GOPROXY = "https://goproxy.cn,direct" # Use proxy for faster download in China, optional but recommended
  
  go mod download
  go build -o system-monitor-backend.exe .
  $svcDir = Join-Path $env:ProgramData 'SystemMonitor'
  New-Item -Force -ItemType Directory -Path $svcDir | Out-Null
  Copy-Item -Force ./system-monitor-backend.exe (Join-Path $svcDir 'system-monitor-backend.exe')
  
  # Handle GeoIP Database
  $geoDbName = "GeoLite2-City.mmdb"
  $geoDbSource = Join-Path $root $geoDbName
  $geoDbDest = Join-Path $svcDir $geoDbName
  $extraEnv = "PORT=$BackendPort"

  if (Test-Path $geoDbSource) {
    Copy-Item -Force $geoDbSource $geoDbDest
    $extraEnv += [Environment]::NewLine + "GEOIP_DB_PATH=$geoDbDest"
    Write-Host "GeoIP database found and deployed." -ForegroundColor Green
  } else {
    Write-Host "GeoIP database not found in project root. Skipping GeoIP setup." -ForegroundColor Yellow
  }

  nssm install SystemMonitorBackend (Join-Path $svcDir 'system-monitor-backend.exe')
  nssm set SystemMonitorBackend AppDirectory $svcDir
  nssm set SystemMonitorBackend AppParameters ""
  nssm set SystemMonitorBackend AppEnvironmentExtra $extraEnv
  nssm set SystemMonitorBackend Start SERVICE_AUTO_START
  netsh advfirewall firewall add rule name="SystemMonitorBackend" dir=in action=allow protocol=TCP localport=$BackendPort | Out-Null
  nssm start SystemMonitorBackend
  Pop-Location
}

function Setup-Frontend {
  $root = $PSScriptRoot
  $webDir = Join-Path $root 'frontend/system-monitor-web'
  Push-Location $webDir
  if(Test-Path package-lock.json) { npm ci } else { npm install }
  npm run build
  npm i -g serve
  $serveCmd = (Get-Command serve).Source
  if (-not $serveCmd) {
      $serveCmd = "C:\Program Files\nodejs\serve.cmd"
  }
  Write-Host "Using serve command at: $serveCmd" -ForegroundColor Cyan
  nssm install SystemMonitorFrontend $serveCmd "-s" "dist" "-l" $FrontendPort
  nssm set SystemMonitorFrontend AppDirectory (Get-Location).Path
  nssm set SystemMonitorFrontend Start SERVICE_AUTO_START
  netsh advfirewall firewall add rule name="SystemMonitorFrontend" dir=in action=allow protocol=TCP localport=$FrontendPort | Out-Null
  nssm start SystemMonitorFrontend
  Pop-Location
}

Ensure-Admin
Install-Choco
Install-Packages
Setup-Backend
Setup-Frontend

Write-Output "Backend: http://localhost:$BackendPort"
Write-Output "Frontend: http://localhost:$FrontendPort"
