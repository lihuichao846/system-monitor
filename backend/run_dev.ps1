# Run Backend with GeoIP Debugging enabled

# Get the script's current directory
$ScriptDir = $PSScriptRoot

# Set GeoIP Database Path to the GeoLite2-City.mmdb file in the same directory
$env:GEOIP_DB_PATH = Join-Path $ScriptDir "GeoLite2-City.mmdb"

Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "   System Monitor Backend - Dev Mode" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "GeoIP DB Path set to: $env:GEOIP_DB_PATH" -ForegroundColor Gray
Write-Host "Starting backend... (Press Ctrl+C to stop)" -ForegroundColor Yellow
Write-Host ""

# Check if the DB file actually exists
if (-not (Test-Path $env:GEOIP_DB_PATH)) {
    Write-Host "[ERROR] GeoLite2-City.mmdb not found in $ScriptDir" -ForegroundColor Red
    Write-Host "Please download it or ensure the file name is correct." -ForegroundColor Red
    exit 1
}

# Run the Go application
go run main.go
