[CmdletBinding()]
param(
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$outputDirectory = Join-Path $root "dist\windows-amd64"

Push-Location $root
try {
    $goPlatform = (& go env GOOS GOARCH) -join "/"
    if ($LASTEXITCODE -ne 0) { throw "Unable to detect the Go platform" }
    if ($goPlatform -ne "windows/amd64") { throw "This script requires windows/amd64; current Go platform is $goPlatform" }

    Write-Host "Running tests..."
    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw "Tests failed" }

    New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null
    $buildVersion = $Version.TrimStart("v")
    $ldflags = "-s -w -X github.com/accloud-proj/x-cmd/internal/version.Version=$buildVersion"

    $output = Join-Path $outputDirectory "x-cmd.exe"
    Remove-Item $output -Force -ErrorAction SilentlyContinue
    Write-Host "Building windows/amd64..."
    & go build -trimpath -ldflags $ldflags -o $output .
    if ($LASTEXITCODE -ne 0) { throw "Build failed for windows/amd64" }

    Write-Host "Build complete: $outputDirectory"
}
finally {
    Pop-Location
}