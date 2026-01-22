# Verman End-to-End Test Script
# Run from project root: .\scripts\test-e2e.ps1

param(
    [switch]$SkipBuild,
    [switch]$SkipInstall,
    [switch]$Cleanup
)

$ErrorActionPreference = "Stop"
$TestDir = "$env:TEMP\verman-e2e-test"
$OriginalPath = $env:PATH
$OriginalHome = $env:USERPROFILE

# Colors for output
function Write-Success { param($msg) Write-Host "[PASS] $msg" -ForegroundColor Green }
function Write-Failure { param($msg) Write-Host "[FAIL] $msg" -ForegroundColor Red }
function Write-Info { param($msg) Write-Host "[INFO] $msg" -ForegroundColor Cyan }
function Write-Step { param($msg) Write-Host "`n=== $msg ===" -ForegroundColor Yellow }

# Track results
$script:passed = 0
$script:failed = 0

function Test-Assert {
    param($condition, $message)
    if ($condition) {
        Write-Success $message
        $script:passed++
    } else {
        Write-Failure $message
        $script:failed++
    }
}

function Cleanup-TestEnv {
    Write-Info "Cleaning up test environment..."
    if (Test-Path $TestDir) {
        Remove-Item -Recurse -Force $TestDir -ErrorAction SilentlyContinue
    }
    # Remove test .verman folder
    $testVerman = "$TestDir\.verman"
    if (Test-Path $testVerman) {
        # Remove junctions first
        Get-ChildItem "$testVerman\versions" -Recurse -Directory |
            Where-Object { $_.Attributes -match 'ReparsePoint' } |
            ForEach-Object { cmd /c rmdir $_.FullName }
        Remove-Item -Recurse -Force $testVerman -ErrorAction SilentlyContinue
    }
}

# Main test execution
try {
    Write-Host "`n=====================================" -ForegroundColor Magenta
    Write-Host "  Verman End-to-End Tests" -ForegroundColor Magenta
    Write-Host "=====================================`n" -ForegroundColor Magenta

    # Setup
    Write-Step "Setting up test environment"

    if ($Cleanup) {
        Cleanup-TestEnv
        Write-Info "Cleanup complete"
        exit 0
    }

    Cleanup-TestEnv
    New-Item -ItemType Directory -Path $TestDir -Force | Out-Null
    Write-Info "Test directory: $TestDir"

    # Build
    if (-not $SkipBuild) {
        Write-Step "Building verman"
        $buildOutput = & go build -o verman.exe 2>&1
        Test-Assert ($LASTEXITCODE -eq 0) "Build successful"
    }

    $verman = "$(Get-Location)\verman.exe"
    Test-Assert (Test-Path $verman) "verman.exe exists"

    # Test 1: Help command
    Write-Step "Test: Help command"
    $helpOutput = & $verman --help 2>&1
    Test-Assert ($LASTEXITCODE -eq 0) "Help command runs"
    Test-Assert ($helpOutput -match "universal version manager") "Help shows description"

    # Test 2: List (empty)
    Write-Step "Test: List command (empty)"
    $listOutput = & $verman list 2>&1
    Test-Assert ($LASTEXITCODE -eq 0) "List command runs"

    # Test 3: Current (empty)
    Write-Step "Test: Current command (empty)"
    $currentOutput = & $verman current 2>&1
    Test-Assert ($LASTEXITCODE -eq 0) "Current command runs"

    # Test 4: Detect (no version files)
    Write-Step "Test: Detect command (no files)"
    Push-Location $TestDir
    $detectOutput = & $verman detect 2>&1
    Test-Assert ($LASTEXITCODE -eq 0) "Detect command runs"
    Pop-Location

    # Test 5: Detect with version file
    Write-Step "Test: Detect with .nvmrc"
    $projectDir = "$TestDir\my-project"
    New-Item -ItemType Directory -Path $projectDir -Force | Out-Null
    Set-Content -Path "$projectDir\.nvmrc" -Value "20.10.0"
    Set-Content -Path "$projectDir\.java-version" -Value "21"

    Push-Location $projectDir
    $detectOutput = & $verman detect 2>&1
    Test-Assert ($detectOutput -match "node.*20") "Detects Node version from .nvmrc"
    Test-Assert ($detectOutput -match "java.*21") "Detects Java version from .java-version"
    Pop-Location

    # Test 6: Detect --json
    Write-Step "Test: Detect --json output"
    Push-Location $projectDir
    $jsonOutput = & $verman detect --json 2>&1
    Test-Assert ($jsonOutput -match '"Language"') "JSON output contains Language field"
    Pop-Location

    # Test 7: Setup --path-only
    Write-Step "Test: Setup --path-only"
    $setupOutput = & $verman setup --path-only 2>&1
    Test-Assert ($LASTEXITCODE -eq 0) "Setup --path-only runs"
    Test-Assert ($setupOutput -match "PATH") "Setup mentions PATH"

    # Test 8: Init powershell
    Write-Step "Test: Init powershell"
    $initOutput = & $verman init powershell 2>&1
    Test-Assert ($LASTEXITCODE -eq 0) "Init powershell runs"
    Test-Assert ($initOutput -match "JAVA_HOME") "Init script contains JAVA_HOME"

    # Test 9: Which (no version set)
    Write-Step "Test: Which command (no version)"
    $whichOutput = & $verman which java 2>&1
    Test-Assert ($whichOutput -match "No.*version") "Which shows no version message"

    # Installation tests (optional - takes time and bandwidth)
    if (-not $SkipInstall) {
        Write-Step "Test: Install Node.js 20.10.0"
        Write-Info "This will download ~25MB..."

        $installOutput = & $verman install node 20.10.0 2>&1
        if ($LASTEXITCODE -eq 0) {
            Test-Assert $true "Node.js 20.10.0 installed"

            # Test use
            Write-Step "Test: Use Node.js"
            $useOutput = & $verman use node 20.10.0 2>&1
            Test-Assert ($LASTEXITCODE -eq 0) "Use node 20.10.0 works"

            # Test current
            $currentOutput = & $verman current node 2>&1
            Test-Assert ($currentOutput -match "20.10.0") "Current shows 20.10.0"

            # Test which
            $whichOutput = & $verman which node 2>&1
            Test-Assert ($whichOutput -match "node") "Which shows node path"

            # Test list
            $listOutput = & $verman list node 2>&1
            Test-Assert ($listOutput -match "20.10.0") "List shows installed version"

            # Test actual node execution
            Write-Step "Test: Execute installed Node"
            $nodePath = & $verman which node 2>&1
            $nodeExe = Join-Path $nodePath "node.exe"
            if (Test-Path $nodeExe) {
                $nodeVersion = & $nodeExe --version 2>&1
                Test-Assert ($nodeVersion -match "v20") "Node executes and shows v20"
            } else {
                Write-Failure "node.exe not found at $nodeExe"
                $script:failed++
            }

            # Test uninstall
            Write-Step "Test: Uninstall Node.js"
            $uninstallOutput = & $verman uninstall node 20.10.0 2>&1
            Test-Assert ($LASTEXITCODE -eq 0) "Uninstall works"

        } else {
            Write-Failure "Node.js installation failed: $installOutput"
            $script:failed++
        }
    } else {
        Write-Info "Skipping install tests (use without -SkipInstall to run)"
    }

    # Summary
    Write-Host "`n=====================================" -ForegroundColor Magenta
    Write-Host "  Test Results" -ForegroundColor Magenta
    Write-Host "=====================================`n" -ForegroundColor Magenta

    Write-Host "Passed: $script:passed" -ForegroundColor Green
    Write-Host "Failed: $script:failed" -ForegroundColor $(if ($script:failed -gt 0) { "Red" } else { "Green" })

    if ($script:failed -gt 0) {
        Write-Host "`nSome tests failed!" -ForegroundColor Red
        exit 1
    } else {
        Write-Host "`nAll tests passed!" -ForegroundColor Green
        exit 0
    }

} catch {
    Write-Failure "Test script error: $_"
    exit 1
} finally {
    # Restore environment
    $env:PATH = $OriginalPath
    Pop-Location -ErrorAction SilentlyContinue
}
