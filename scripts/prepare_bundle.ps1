<#
.SYNOPSIS
    Prepares the full-bundle directory for SPECTRE Windows packaging.
.PARAMETER Version
    The release version string (e.g., v1.0.0).
.PARAMETER OutputDir
    Destination bundle directory (defaults to dist\bundle).
#>
[CmdletBinding()]
param(
    [string]$Version = "v1.0.0",
    [string]$OutputDir = "dist\bundle"
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Push-Location $ProjectRoot

try {
    Write-Host "==================================================" -ForegroundColor Cyan
    Write-Host "  Preparing SPECTRE Windows Full Bundle ($Version)  " -ForegroundColor Cyan
    Write-Host "==================================================" -ForegroundColor Cyan

    $FullOutputDir = Join-Path $ProjectRoot $OutputDir
    if (Test-Path $FullOutputDir) {
        Write-Host "[*] Cleaning existing bundle directory..." -ForegroundColor Yellow
        Remove-Item -Recurse -Force $FullOutputDir
    }
    New-Item -ItemType Directory -Force -Path $FullOutputDir | Out-Null

    # 1. Compile static Windows x64 binary
    Write-Host "[*] Compiling static 64-bit Windows binary (spectre.exe)..." -ForegroundColor Yellow
    $env:CGO_ENABLED = "0"
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"

    $TargetExe = Join-Path $FullOutputDir "spectre.exe"
    $verFlag = "-X github.com/Gokul-Eswar/Spectre/internal/version.Version=$Version"
    & go build -trimpath -ldflags="-s -w $verFlag" -o $TargetExe ./cmd/spectre
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to compile spectre.exe"
    }
    Write-Host "[+] Compiled: $TargetExe" -ForegroundColor Green

    # 2. Copy application assets into bundle
    $Assets = @(
        "configs",
        "web",
        "plugins",
        "analyzer",
        "LICENSE",
        "README.md",
        "registry_sample.json"
    )

    foreach ($item in $Assets) {
        $sourcePath = Join-Path $ProjectRoot $item
        if (Test-Path $sourcePath) {
            Write-Host "[*] Bundling: $item" -ForegroundColor Gray
            Copy-Item -Path $sourcePath -Destination $FullOutputDir -Recurse -Force
        }
    }

    $BundleSizeMB = (Get-ChildItem -Recurse $FullOutputDir | Measure-Object -Property Length -Sum).Sum / 1MB
    Write-Host ("[+] Full bundle staged at: {0} ({1:N2} MB)" -f $FullOutputDir, $BundleSizeMB) -ForegroundColor Green
}
finally {
    Pop-Location
}
