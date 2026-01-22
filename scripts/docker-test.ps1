# Docker Test Runner for Verman
# Runs verman tests in a clean Windows container environment
#
# Prerequisites:
#   - Docker Desktop for Windows
#   - Switch to Windows containers (right-click Docker icon > Switch to Windows containers)
#
# Usage:
#   .\scripts\docker-test.ps1           # Build and run tests
#   .\scripts\docker-test.ps1 -Shell    # Open interactive shell in container
#   .\scripts\docker-test.ps1 -Build    # Only build the image
#   .\scripts\docker-test.ps1 -FullTest # Run full tests including installs

param(
    [switch]$Shell,      # Open interactive shell instead of running tests
    [switch]$Build,      # Only build, don't run
    [switch]$FullTest,   # Include install tests (downloads ~100MB+)
    [switch]$NoBuild     # Skip build, use existing image
)

$ErrorActionPreference = "Stop"
$ImageName = "verman-test"

function Write-Info { param($msg) Write-Host "[INFO] $msg" -ForegroundColor Cyan }
function Write-Step { param($msg) Write-Host "`n=== $msg ===" -ForegroundColor Yellow }

# Check Docker is running and using Windows containers
Write-Step "Checking Docker"
try {
    $dockerInfo = (docker info 2>&1) | Out-String
    if ($dockerInfo -notmatch "OSType:\s*windows") {
        Write-Host "ERROR: Docker is not using Windows containers!" -ForegroundColor Red
        Write-Host "Right-click Docker Desktop icon and select 'Switch to Windows containers...'" -ForegroundColor Yellow
        exit 1
    }
    Write-Info "Docker is running with Windows containers"
} catch {
    Write-Host "ERROR: Docker is not running or not installed" -ForegroundColor Red
    exit 1
}

# Build the image
if (-not $NoBuild) {
    Write-Step "Building verman.exe"
    go build -o verman.exe
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Go build failed" -ForegroundColor Red
        exit 1
    }
    Write-Info "verman.exe built successfully"

    Write-Step "Building Docker image"
    Write-Info "First run pulls ~300MB image..."

    docker build -t $ImageName .
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Docker build failed" -ForegroundColor Red
        exit 1
    }
    Write-Info "Image built successfully"
}

if ($Build) {
    Write-Host "`nBuild complete. Run without -Build to execute tests." -ForegroundColor Green
    exit 0
}

# Run tests or shell
Write-Step "Running container"

if ($Shell) {
    Write-Info "Starting interactive PowerShell session..."
    Write-Host "Type 'exit' to leave the container`n" -ForegroundColor Yellow
    docker run --rm -it $ImageName
} elseif ($FullTest) {
    Write-Info "Running full e2e tests (including installations)..."
    docker run --rm $ImageName -File .\scripts\test-e2e.ps1
} else {
    Write-Info "Running e2e tests (skipping installations)..."
    docker run --rm $ImageName -File .\scripts\test-e2e.ps1 -SkipInstall
}

$exitCode = $LASTEXITCODE
if ($exitCode -eq 0) {
    Write-Host "`nAll tests passed!" -ForegroundColor Green
} else {
    Write-Host "`nTests failed with exit code $exitCode" -ForegroundColor Red
}

exit $exitCode
