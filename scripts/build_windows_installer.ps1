<#
.SYNOPSIS
    Builds the single-executable Windows Installer (Spectre-Setup-x64.exe) using Inno Setup,
    computes SHA256 checksum, and optionally signs the binary.
.PARAMETER Version
    The release version (e.g. v1.0.0).
.PARAMETER CertificatePath
    Path to PFX code-signing certificate (optional).
.PARAMETER CertificatePassword
    Password for PFX certificate (optional).
#>
[CmdletBinding()]
param(
    [string]$Version = "v1.0.0",
    [string]$CertificatePath = "",
    [string]$CertificatePassword = ""
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Push-Location $ProjectRoot

try {
    Write-Host "==================================================" -ForegroundColor Cyan
    Write-Host "  SPECTRE Windows Inno Setup Pipeline ($Version)   " -ForegroundColor Cyan
    Write-Host "==================================================" -ForegroundColor Cyan

    $DistDir = Join-Path $ProjectRoot "dist"
    $BundleDir = Join-Path $DistDir "bundle"

    # Step 1: Prepare the bundle directory
    & (Join-Path $PSScriptRoot "prepare_bundle.ps1") -Version $Version -OutputDir "dist/bundle"

    # Step 2: Locate Inno Setup Compiler (ISCC)
    $IsccCandidates = @(
        (Get-Command iscc.exe -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source -ErrorAction SilentlyContinue),
        "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe",
        "${env:ProgramFiles}\Inno Setup 6\ISCC.exe",
        "${env:ProgramData}\chocolatey\bin\iscc.exe"
    )

    $IsccPath = $null
    foreach ($cand in $IsccCandidates) {
        if ($cand -and (Test-Path $cand)) {
            $IsccPath = $cand
            break
        }
    }

    if (-not $IsccPath) {
        Write-Error @"
Inno Setup Compiler (ISCC.exe) was not found.
To install Inno Setup, run either:
  choco install innosetup -y
or
  winget install JRSoftware.InnoSetup --silent
"@
        exit 1
    }

    Write-Host "[*] Using Inno Setup Compiler: $IsccPath" -ForegroundColor Green

    # Clean version for Inno Setup (strip leading 'v')
    $CleanVersion = $Version.TrimStart('v')
    $IssPath = Join-Path $ProjectRoot "packaging\windows\spectre.iss"

    # Step 3: Compile Inno Setup Installer
    Write-Host "[*] Compiling installer..." -ForegroundColor Yellow
    & "$IsccPath" "/DMyAppVersion=$CleanVersion" "$IssPath"
    if ($LASTEXITCODE -ne 0) {
        throw "Inno Setup compilation failed with exit code $LASTEXITCODE"
    }

    $InstallerPath = Join-Path $DistDir "Spectre-Setup-x64.exe"
    if (-not (Test-Path $InstallerPath)) {
        throw "Installer artifact not found at $InstallerPath"
    }

    # Step 4: Code Signing (Optional / If configured)
    if ($CertificatePath -and (Test-Path $CertificatePath)) {
        Write-Host "[*] Code-signing installer..." -ForegroundColor Yellow
        $SignToolCandidates = @(
            (Get-Command signtool.exe -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source -ErrorAction SilentlyContinue),
            "${env:ProgramFiles(x86)}\Windows Kits\10\bin\10.0.22621.0\x64\signtool.exe",
            "${env:ProgramFiles(x86)}\Windows Kits\10\bin\x64\signtool.exe",
            "${env:ProgramFiles}\Windows Kits\10\bin\x64\signtool.exe"
        )
        $SignTool = $SignToolCandidates | Where-Object { $_ -and (Test-Path $_) } | Select-Object -First 1

        if ($SignTool) {
            $SignArgs = @("sign", "/fd", "SHA256", "/tr", "http://timestamp.digicert.com", "/td", "SHA256", "/f", $CertificatePath)
            if ($CertificatePassword) {
                $SignArgs += @("/p", $CertificatePassword)
            }
            $SignArgs += $InstallerPath
            & "$SignTool" @SignArgs
            if ($LASTEXITCODE -eq 0) {
                Write-Host "[+] Code signing successful." -ForegroundColor Green
            } else {
                Write-Warning "Code signing failed. Proceeding with unsigned installer."
            }
        } else {
            Write-Warning "signtool.exe not found in Windows SDK paths. Skipping signing."
        }
    } else {
        Write-Host "[i] No code signing certificate provided. Installer is unsigned." -ForegroundColor DarkGray
    }

    # Step 5: Compute SHA256 Checksum
    $Hash = (Get-FileHash -Path $InstallerPath -Algorithm SHA256).Hash.ToUpper()
    $Sha256Path = "$InstallerPath.sha256"
    "$Hash  Spectre-Setup-x64.exe" | Out-File -FilePath $Sha256Path -Encoding ascii -Force

    $InstallerSizeMB = (Get-Item $InstallerPath).Length / 1MB
    Write-Host "`n==================================================" -ForegroundColor Green
    Write-Host " 🎉 Windows Full-Bundle Installer Ready!" -ForegroundColor Green
    Write-Host (" Installer: {0} ({1:N2} MB)" -f $InstallerPath, $InstallerSizeMB) -ForegroundColor Cyan
    Write-Host (" SHA256:    {0}" -f $Hash) -ForegroundColor Cyan
    Write-Host (" Checksum:  {0}" -f $Sha256Path) -ForegroundColor Cyan
    Write-Host "==================================================" -ForegroundColor Green

    Write-Host "`nUser Copy-Paste PowerShell Command (multi-step):" -ForegroundColor Yellow
    Write-Host @"
`$url = 'https://github.com/Gokul-Eswar/Spectre/releases/download/$Version/Spectre-Setup-x64.exe'
`$out = "`$env:TEMP\Spectre-Setup.exe"
Invoke-WebRequest -Uri `$url -OutFile `$out
`$expected = '$Hash'
if ((Get-FileHash -Path `$out -Algorithm SHA256).Hash -ne `$expected) { Write-Error 'Checksum mismatch — aborting'; exit 1 }
Start-Process -FilePath `$out -ArgumentList '/VERYSILENT','/NORESTART' -Wait
"@ -ForegroundColor Gray

    Write-Host "`nUser One-Liner PowerShell Command:" -ForegroundColor Yellow
    Write-Host @"
powershell -NoProfile -Command "`$u='https://github.com/Gokul-Eswar/Spectre/releases/download/$Version/Spectre-Setup-x64.exe';`$o=`"`$env:TEMP\Spectre-Setup.exe`";Invoke-WebRequest `$u -OutFile `$o; if((Get-FileHash `$o -Algorithm SHA256).Hash -ne '$Hash'){Write-Error 'Checksum mismatch'; exit 1}; Start-Process -FilePath `$o -ArgumentList '/VERYSILENT','/NORESTART' -Wait"
"@ -ForegroundColor Gray
}
finally {
    Pop-Location
}
