# Full Integration Test Script for Docker
# Tests actual installation and usage of language runtimes
# Run inside Docker container: powershell -File .\scripts\test-docker-full.ps1

$ErrorActionPreference = "Stop"

function Write-Success { param($msg) Write-Host "[PASS] $msg" -ForegroundColor Green }
function Write-Failure { param($msg) Write-Host "[FAIL] $msg" -ForegroundColor Red }
function Write-Info { param($msg) Write-Host "[INFO] $msg" -ForegroundColor Cyan }
function Write-Step { param($msg) Write-Host "`n=== $msg ===" -ForegroundColor Yellow }

$script:passed = 0
$script:failed = 0

function Test-Assert {
    param($condition, $message)
    if ($condition) {
        Write-Success $message
        $script:passed++
        return $true
    } else {
        Write-Failure $message
        $script:failed++
        return $false
    }
}

$verman = "C:\verman\verman.exe"

Write-Host "`n=============================================" -ForegroundColor Magenta
Write-Host "  Verman Full Integration Tests (Docker)" -ForegroundColor Magenta
Write-Host "=============================================`n" -ForegroundColor Magenta

# Test 1: Basic commands
Write-Step "Basic Commands"
& $verman --help | Out-Null
Test-Assert ($LASTEXITCODE -eq 0) "Help command works"

& $verman list | Out-Null
Test-Assert ($LASTEXITCODE -eq 0) "List command works"

& $verman current | Out-Null
Test-Assert ($LASTEXITCODE -eq 0) "Current command works"

# Test 2: Node.js Installation
Write-Step "Node.js Installation Test"
Write-Info "Installing Node.js 20.10.0 (this downloads ~25MB)..."

& $verman install node 20.10.0
if (Test-Assert ($LASTEXITCODE -eq 0) "Node.js 20.10.0 installed") {

    & $verman use node 20.10.0
    Test-Assert ($LASTEXITCODE -eq 0) "Switched to Node.js 20.10.0"

    $currentNode = & $verman current node 2>&1
    Test-Assert ($currentNode -match "20.10.0") "Current shows Node.js 20.10.0"

    # Test actual execution
    $nodePath = & $verman which node 2>&1
    $nodeExe = Join-Path $nodePath "node.exe"
    if (Test-Path $nodeExe) {
        $nodeVersion = & $nodeExe --version 2>&1
        Test-Assert ($nodeVersion -match "v20.10.0") "Node.js executes correctly"

        # Test npm
        $npmExe = Join-Path $nodePath "npm.cmd"
        if (Test-Path $npmExe) {
            $npmVersion = & $npmExe --version 2>&1
            Test-Assert ($npmVersion -match "\d+\.\d+") "npm works"
        }
    } else {
        Write-Failure "node.exe not found at expected path"
        $script:failed++
    }
}

# Test 3: Java Installation (Adoptium)
Write-Step "Java Installation Test"
Write-Info "Installing Java 21 from Adoptium (this downloads ~180MB)..."

& $verman install java 21
if (Test-Assert ($LASTEXITCODE -eq 0) "Java 21 installed") {

    & $verman use java 21
    Test-Assert ($LASTEXITCODE -eq 0) "Switched to Java 21"

    $currentJava = & $verman current java 2>&1
    Test-Assert ($currentJava -match "21") "Current shows Java 21"

    # Test actual execution
    $javaPath = & $verman which java 2>&1
    $javaExe = Join-Path $javaPath "bin\java.exe"
    if (Test-Path $javaExe) {
        $javaVersion = & $javaExe -version 2>&1
        Test-Assert ($javaVersion -match "21") "Java executes correctly"
    } else {
        Write-Failure "java.exe not found at expected path"
        $script:failed++
    }
}

# Test 4: Go Installation
Write-Step "Go Installation Test"
Write-Info "Installing Go 1.21.5 (this downloads ~65MB)..."

& $verman install go 1.21.5
if (Test-Assert ($LASTEXITCODE -eq 0) "Go 1.21.5 installed") {

    & $verman use go 1.21.5
    Test-Assert ($LASTEXITCODE -eq 0) "Switched to Go 1.21.5"

    $goPath = & $verman which go 2>&1
    $goExe = Join-Path $goPath "bin\go.exe"
    if (Test-Path $goExe) {
        $goVersion = & $goExe version 2>&1
        Test-Assert ($goVersion -match "go1.21") "Go executes correctly"
    } else {
        Write-Failure "go.exe not found at expected path"
        $script:failed++
    }
}

# Test 5: Version Detection
Write-Step "Version Detection Test"
$testProject = "C:\test-project"
New-Item -ItemType Directory -Path $testProject -Force | Out-Null
Set-Content -Path "$testProject\.nvmrc" -Value "20.10.0"
Set-Content -Path "$testProject\.java-version" -Value "21"
Set-Content -Path "$testProject\.go-version" -Value "1.21.5"

Push-Location $testProject
$detected = & $verman detect 2>&1
Test-Assert ($detected -match "node.*20") "Detects Node from .nvmrc"
Test-Assert ($detected -match "java.*21") "Detects Java from .java-version"
Test-Assert ($detected -match "go.*1.21") "Detects Go from .go-version"

# Test --apply
$applyOutput = & $verman detect --apply 2>&1
Test-Assert ($LASTEXITCODE -eq 0) "Detect --apply works"
Pop-Location

# Test 6: List installed versions
Write-Step "List Installed Versions"
$listAll = & $verman list 2>&1
Test-Assert ($listAll -match "node.*20.10.0") "List shows Node.js"
Test-Assert ($listAll -match "java.*21") "List shows Java"
Test-Assert ($listAll -match "go.*1.21") "List shows Go"

# Test 7: Cleanup (Uninstall)
Write-Step "Uninstall Test"
& $verman uninstall node 20.10.0
Test-Assert ($LASTEXITCODE -eq 0) "Uninstall Node.js works"

# Summary
Write-Host "`n=============================================" -ForegroundColor Magenta
Write-Host "  Test Results" -ForegroundColor Magenta
Write-Host "=============================================`n" -ForegroundColor Magenta

Write-Host "Passed: $script:passed" -ForegroundColor Green
Write-Host "Failed: $script:failed" -ForegroundColor $(if ($script:failed -gt 0) { "Red" } else { "Green" })

if ($script:failed -gt 0) {
    Write-Host "`nSome tests failed!" -ForegroundColor Red
    exit 1
} else {
    Write-Host "`nAll tests passed!" -ForegroundColor Green
    exit 0
}
