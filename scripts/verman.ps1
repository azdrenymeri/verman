# Verman PowerShell Wrapper
# This provides shell integration for version switching

param(
    [Parameter(Position=0)]
    [string]$Command,

    [Parameter(Position=1, ValueFromRemainingArguments=$true)]
    [string[]]$Arguments
)

$VermanExe = Join-Path $PSScriptRoot "..\verman.exe"

if (-not (Test-Path $VermanExe)) {
    # Try to find in PATH
    $VermanExe = "verman"
}

switch ($Command) {
    "use" {
        # Run verman and update current session environment
        & $VermanExe use @Arguments
        if ($LASTEXITCODE -eq 0 -and $Arguments.Count -ge 2) {
            $lang = $Arguments[0]
            $currentPath = & $VermanExe which $lang 2>$null
            if ($currentPath) {
                # Update session environment based on language
                switch ($lang) {
                    "java" {
                        $env:JAVA_HOME = $currentPath
                        $env:PATH = "$currentPath\bin;$env:PATH"
                    }
                    "scala" {
                        $env:SCALA_HOME = $currentPath
                        $env:PATH = "$currentPath\bin;$env:PATH"
                    }
                    "node" {
                        $env:PATH = "$currentPath;$env:PATH"
                    }
                    "python" {
                        $env:PYTHON_HOME = $currentPath
                        $env:PATH = "$currentPath;$currentPath\Scripts;$env:PATH"
                    }
                    "ruby" {
                        $env:GEM_HOME = "$currentPath\gems"
                        $env:GEM_PATH = "$currentPath\gems"
                        $env:PATH = "$currentPath\bin;$currentPath\gems\bin;$env:PATH"
                    }
                    "go" {
                        $env:GOROOT = $currentPath
                        $env:PATH = "$currentPath\bin;$env:PATH"
                    }
                    "rust" {
                        $env:RUSTUP_HOME = "$currentPath\rustup"
                        $env:CARGO_HOME = "$currentPath\cargo"
                        $env:PATH = "$currentPath\cargo\bin;$env:PATH"
                    }
                    "dotnet" {
                        $env:DOTNET_ROOT = $currentPath
                        $env:PATH = "$currentPath;$env:PATH"
                    }
                }
            }
        }
    }

    "detect" {
        if ($Arguments -contains "--apply") {
            # Detect and apply versions
            $detected = & $VermanExe detect --json | ConvertFrom-Json
            foreach ($item in $detected) {
                & $VermanExe use $item.Language $item.Version
            }
        } else {
            & $VermanExe detect @Arguments
        }
    }

    default {
        & $VermanExe $Command @Arguments
    }
}
