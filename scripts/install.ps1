[CmdletBinding()]
param(
    [string]$Version = "latest",
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA "x-cmd\bin"),
    [string]$GitHubMirror = ""
)

$ErrorActionPreference = "Stop"
$repository = "accloud-proj/x-cmd"
if ($GitHubMirror) {
    if ($GitHubMirror -notmatch '^https?://') { $GitHubMirror = "https://$GitHubMirror" }
    $GitHubMirror = $GitHubMirror.TrimEnd('/')
}
$architecture = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()) {
    "X64" { "amd64" }
    "Arm64" { "arm64" }
    "X86" { "386" }
    default { throw "Unsupported architecture: $_" }
}
$asset = "x-cmd_windows_$architecture.zip"

if ($Version -eq "latest") {
    $releasePath = "latest/download"
}
else {
    if (-not $Version.StartsWith("v")) { $Version = "v$Version" }
    $releasePath = "download/$Version"
}

function Get-GitHubUrl([string]$FileName) {
    if ($GitHubMirror) { return "$GitHubMirror/$repository/releases/$releasePath/$FileName" }
    return "https://github.com/$repository/releases/$releasePath/$FileName"
}

$temporaryDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("x-cmd-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $temporaryDirectory | Out-Null
try {
    $archive = Join-Path $temporaryDirectory $asset
    $checksums = Join-Path $temporaryDirectory "checksums.txt"
    Write-Host "Downloading $asset..."
    Invoke-WebRequest -Uri (Get-GitHubUrl $asset) -OutFile $archive
    Invoke-WebRequest -Uri (Get-GitHubUrl "checksums.txt") -OutFile $checksums

    $checksumLine = Get-Content $checksums | Where-Object { $_ -match "\s\*?$([regex]::Escape($asset))$" } | Select-Object -First 1
    if (-not $checksumLine) { throw "No checksum found for $asset" }
    $expected = ($checksumLine -split "\s+")[0].ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 $archive).Hash.ToLowerInvariant()
    if ($expected -ne $actual) { throw "Checksum verification failed for $asset" }

    Expand-Archive -Path $archive -DestinationPath $temporaryDirectory -Force
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $installedPath = Join-Path $InstallDir "x-cmd.exe"
    Copy-Item (Join-Path $temporaryDirectory "x-cmd.exe") $installedPath -Force
    if ($GitHubMirror) {
        & $installedPath config set --github-mirror $GitHubMirror
        if ($LASTEXITCODE -ne 0) { throw "Failed to save GitHub routing settings" }
    }
    Write-Host "Installed x-cmd to $installedPath"
    if (($env:Path -split ';') -notcontains $InstallDir) {
        Write-Host "Add $InstallDir to PATH to run x-cmd directly."
    }
}
finally {
    Remove-Item $temporaryDirectory -Recurse -Force -ErrorAction SilentlyContinue
}