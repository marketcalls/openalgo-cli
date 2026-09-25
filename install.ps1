# OpenAlgo CLI installer for Windows.
#
#   irm https://raw.githubusercontent.com/marketcalls/openalgo-cli/main/install.ps1 | iex
#
# Environment overrides:
#   OPENALGO_CLI_VERSION   release tag to install (default: latest), e.g. v0.0.1
#   OPENALGO_INSTALL_DIR   install directory (default: %LOCALAPPDATA%\Programs\openalgo)
$ErrorActionPreference = 'Stop'

$repo = 'marketcalls/openalgo-cli'

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }

$version = $env:OPENALGO_CLI_VERSION
if (-not $version) {
    $version = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
}
$num = $version.TrimStart('v')

$archive = "openalgo-cli_${num}_windows_${arch}.zip"
$base = "https://github.com/$repo/releases/download/$version"

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("openalgo-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading openalgo $version (windows/$arch)..."
    Invoke-WebRequest "$base/$archive" -OutFile (Join-Path $tmp $archive) -UseBasicParsing
    Invoke-WebRequest "$base/checksums.txt" -OutFile (Join-Path $tmp 'checksums.txt') -UseBasicParsing

    $line = Get-Content (Join-Path $tmp 'checksums.txt') | Where-Object { $_ -match " $([regex]::Escape($archive))$" }
    if (-not $line) { throw "no checksum for $archive" }
    $expected = ($line -split '\s+')[0]
    $actual = (Get-FileHash (Join-Path $tmp $archive) -Algorithm SHA256).Hash.ToLower()
    if ($expected -ne $actual) { throw "checksum mismatch for $archive" }

    Expand-Archive (Join-Path $tmp $archive) -DestinationPath $tmp -Force

    $dir = $env:OPENALGO_INSTALL_DIR
    if (-not $dir) { $dir = Join-Path $env:LOCALAPPDATA 'Programs\openalgo' }
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    Copy-Item (Join-Path $tmp 'openalgo.exe') (Join-Path $dir 'openalgo.exe') -Force

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (($userPath -split ';') -notcontains $dir) {
        [Environment]::SetEnvironmentVariable('Path', "$userPath;$dir", 'User')
        Write-Host "Added $dir to your user PATH. Open a new terminal to use it."
    }
    Write-Host "Installed openalgo $version to $dir\openalgo.exe"
    Write-Host 'Next: openalgo profile login'
}
finally {
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
