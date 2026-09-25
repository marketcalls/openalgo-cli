# OpenAlgo CLI installer for Windows.
#
#   irm https://raw.githubusercontent.com/marketcalls/openalgo-cli/main/install.ps1 | iex
#
# Environment overrides:
#   OPENALGO_CLI_VERSION   release tag to install (default: latest), e.g. v0.0.1
#   OPENALGO_INSTALL_DIR   install directory (default: %LOCALAPPDATA%\Programs\openalgo)
$ErrorActionPreference = 'Stop'
# The progress bar makes Invoke-WebRequest many times slower in Windows PowerShell 5.1.
$ProgressPreference = 'SilentlyContinue'

$repo = 'marketcalls/openalgo-cli'

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }

# Older Windows PowerShell defaults can exclude TLS 1.2, which GitHub requires.
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

# Resolve the newest release from the github.com/.../releases/latest redirect,
# which is not subject to the GitHub API's anonymous rate limit; the API is
# the fallback.
function Get-LatestTag {
    try {
        $req = [System.Net.WebRequest]::Create("https://github.com/$repo/releases/latest")
        $req.AllowAutoRedirect = $false
        $resp = $req.GetResponse()
        $loc = $resp.Headers['Location']
        $resp.Close()
        if ($loc -match '/releases/tag/([^/]+)$') { return $Matches[1] }
    } catch { }
    return (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest" -UseBasicParsing).tag_name
}

$version = $env:OPENALGO_CLI_VERSION
if (-not $version) {
    $version = Get-LatestTag
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
    # .NET directly rather than Get-FileHash / Expand-Archive: those are
    # script-module cmdlets that fail to load when Windows PowerShell is
    # started from PowerShell 7 (which is how `openalgo update` can run).
    $sha = [System.Security.Cryptography.SHA256]::Create()
    $fs = [System.IO.File]::OpenRead((Join-Path $tmp $archive))
    try { $actual = -join ($sha.ComputeHash($fs) | ForEach-Object { $_.ToString('x2') }) } finally { $fs.Close() }
    if ($expected -ne $actual) { throw "checksum mismatch for $archive" }

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $unzipped = Join-Path $tmp 'unzipped'
    [System.IO.Compression.ZipFile]::ExtractToDirectory((Join-Path $tmp $archive), $unzipped)

    $dir = $env:OPENALGO_INSTALL_DIR
    if (-not $dir) { $dir = Join-Path $env:LOCALAPPDATA 'Programs\openalgo' }
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    $target = Join-Path $dir 'openalgo.exe'
    $old = "$target.old"
    # Windows locks a running executable against overwrite but allows a
    # rename, so `openalgo update` (which runs this script from inside
    # openalgo.exe) moves the running binary aside first. The leftover .old
    # is removed on the next install, once nothing is running it.
    Remove-Item $old -Force -ErrorAction SilentlyContinue
    if (Test-Path $target) {
        Move-Item $target $old -Force
    }
    Copy-Item (Join-Path $unzipped 'openalgo.exe') $target -Force
    Remove-Item $old -Force -ErrorAction SilentlyContinue

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
