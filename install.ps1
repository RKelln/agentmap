# install.ps1 — install agentmap on Windows
# Usage: iwr https://raw.githubusercontent.com/RKelln/agentmap/main/install.ps1 | iex
# Or: iwr https://raw.githubusercontent.com/RKelln/agentmap/main/install.ps1 -OutFile install.ps1; .\install.ps1

[CmdletBinding()]
param(
    [switch]$Yes,
    [string]$Version = 'latest',
    [string]$BinDir = (Join-Path $env:LOCALAPPDATA 'agentmap')
)

$ErrorActionPreference = 'Stop'

$Repo    = 'RKelln/agentmap'
$BinName = 'agentmap.exe'

function Write-Info  { param($Msg) Write-Host "  $Msg" -ForegroundColor Green }
function Write-Warn  { param($Msg) Write-Host "WARN: $Msg" -ForegroundColor Yellow }
function Write-Err   { param($Msg) Write-Host "ERROR: $Msg" -ForegroundColor Red; exit 1 }

# --- arch detection ---
$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'x86_64' }

# --- resolve latest version ---
if ($Version -eq 'latest') {
    Write-Info 'Fetching latest release...'
    try {
        $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        $Version = $Release.tag_name
    } catch {
        Write-Warn 'No stable release found; checking latest prerelease...'
        try {
            $Releases = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases?per_page=1"
            if ($Releases -and $Releases.Count -gt 0) {
                $Version = $Releases[0].tag_name
            }
        } catch {
            Write-Err "Failed to fetch latest version: $_"
        }
    }
    if (-not $Version) {
        Write-Err 'Failed to resolve latest version from GitHub API. Try -Version vX.Y.Z'
    }
}

$Archive  = "agentmap_Windows_$Arch.zip"
$BaseUrl  = "https://github.com/$Repo/releases/download/$Version"
$DownloadUrl  = "$BaseUrl/$Archive"
$ChecksumUrl  = "$BaseUrl/checksums.txt"

Write-Host ""
Write-Host "Installing agentmap $Version" -ForegroundColor Green -NoNewline
Write-Host ""
Write-Info "OS/Arch:  Windows/$Arch"
Write-Info "Archive:  $Archive"
Write-Info "Install:  $BinDir\agentmap.exe"
Write-Host ""

# Confirm if not --yes and running interactively.
if (-not $Yes -and [Environment]::UserInteractive) {
    $answer = Read-Host 'Proceed with installation? [y/N]'
    if ($answer -notmatch '^[yY]$') {
        Write-Info 'Installation cancelled.'
        exit 0
    }
}

# Create temp directory with cleanup.
$TmpDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
try {
    # Download archive.
    Write-Info "Downloading $Archive..."
    Invoke-WebRequest -Uri $DownloadUrl -OutFile (Join-Path $TmpDir $Archive) -UseBasicParsing

    # Download + verify checksums.
    Write-Info 'Verifying checksum...'
    $ChecksumFile = Join-Path $TmpDir 'checksums.txt'
    Invoke-WebRequest -Uri $ChecksumUrl -OutFile $ChecksumFile -UseBasicParsing

    $Expected = (Get-Content $ChecksumFile | Where-Object { $_ -match "  $Archive$" }) -replace ' .*', ''
    if (-not $Expected) {
        Write-Err "Checksum for $Archive not found in checksums.txt"
    }
    $Actual = (Get-FileHash -Algorithm SHA256 (Join-Path $TmpDir $Archive)).Hash.ToLower()
    if ($Expected -ne $Actual) {
        Write-Err "Checksum mismatch.`n  Expected: $Expected`n  Actual:   $Actual"
    }

    # Extract.
    Write-Info 'Extracting...'
    Expand-Archive -Path (Join-Path $TmpDir $Archive) -DestinationPath $TmpDir -Force

    # Create install dir if needed.
    if (-not (Test-Path $BinDir)) {
        New-Item -ItemType Directory -Path $BinDir | Out-Null
    }

    # Copy binary.
    $Src  = Join-Path $TmpDir 'agentmap.exe'
    $Dest = Join-Path $BinDir 'agentmap.exe'
    Copy-Item -Path $Src -Destination $Dest -Force

    Write-Info "agentmap $Version installed to $Dest"

    # Add BinDir to user PATH if not already present.
    $UserPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    if ($UserPath -notlike "*$BinDir*") {
        [Environment]::SetEnvironmentVariable('PATH', "$UserPath;$BinDir", 'User')
        Write-Info "Added $BinDir to your user PATH."
        Write-Warn 'Restart your terminal for the PATH change to take effect.'
    }

    Write-Host ""
    Write-Host 'Done! Run: agentmap --help' -ForegroundColor Green
    Write-Host ""

} finally {
    Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
}
