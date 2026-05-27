param(
  [string]$Version = "",
  [string]$Repo = "iKeilo/flvxt2"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($Version)) {
  $tag = (git -C $root tag --sort=-creatordate | Select-Object -First 1).Trim()
  if ([string]::IsNullOrWhiteSpace($tag)) {
    throw "No git tag found. Pass -Version explicitly."
  }
  $Version = $tag
}

$artifactsRoot = Join-Path $root "artifacts"
$releaseDir = Join-Path $artifactsRoot $Version
$goGostDir = Join-Path $root "go-gost"
$goTmpDir = Join-Path $root ".release-tmp"
$goCacheDir = Join-Path $root ".release-gocache"

if (Test-Path $releaseDir) {
  Remove-Item -Recurse -Force $releaseDir
}

New-Item -ItemType Directory -Path $releaseDir | Out-Null
New-Item -ItemType Directory -Path $goTmpDir -Force | Out-Null
New-Item -ItemType Directory -Path $goCacheDir -Force | Out-Null

Push-Location $goGostDir
try {
  $env:CGO_ENABLED = "0"
  $env:GOOS = "linux"
  $env:GOTMPDIR = $goTmpDir
  $env:GOCACHE = $goCacheDir

  $env:GOARCH = "amd64"
  go build "-ldflags=-s -w -X main.version=$Version" -o (Join-Path $releaseDir "gost-amd64")

  $env:GOARCH = "arm64"
  go build "-ldflags=-s -w -X main.version=$Version" -o (Join-Path $releaseDir "gost-arm64")
}
finally {
  Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
  Remove-Item Env:GOOS -ErrorAction SilentlyContinue
  Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
  Remove-Item Env:GOTMPDIR -ErrorAction SilentlyContinue
  Remove-Item Env:GOCACHE -ErrorAction SilentlyContinue
  Pop-Location
}

$shaAmd64 = (Get-FileHash -Algorithm SHA256 (Join-Path $releaseDir "gost-amd64")).Hash.ToLower()
$shaArm64 = (Get-FileHash -Algorithm SHA256 (Join-Path $releaseDir "gost-arm64")).Hash.ToLower()
Set-Content -NoNewline -Path (Join-Path $releaseDir "gost-amd64.sha256") -Value "$shaAmd64  gost-amd64"
Set-Content -NoNewline -Path (Join-Path $releaseDir "gost-arm64.sha256") -Value "$shaArm64  gost-arm64"

Copy-Item (Join-Path $root "install.sh") (Join-Path $releaseDir "install.sh")
Copy-Item (Join-Path $root "panel_install.sh") (Join-Path $releaseDir "panel_install.sh")
Copy-Item (Join-Path $root "docker-compose-v4.yml") (Join-Path $releaseDir "docker-compose-v4.yml")
Copy-Item (Join-Path $root "docker-compose-v6.yml") (Join-Path $releaseDir "docker-compose-v6.yml")

$installPath = Join-Path $releaseDir "install.sh"
$panelInstallPath = Join-Path $releaseDir "panel_install.sh"

$installContent = Get-Content $installPath -Raw
$installContent = $installContent -replace 'PINNED_VERSION=""', "PINNED_VERSION=`"$Version`""
$installContent = $installContent -replace 'REPO="[^"]+"', "REPO=`"$Repo`""
Set-Content $installPath $installContent

$panelInstallContent = Get-Content $panelInstallPath -Raw
$panelInstallContent = $panelInstallContent -replace 'PINNED_VERSION=""', "PINNED_VERSION=`"$Version`""
$panelInstallContent = $panelInstallContent -replace 'REPO="[^"]+"', "REPO=`"$Repo`""
Set-Content $panelInstallPath $panelInstallContent

$offlineAmd64Dir = Join-Path $releaseDir "offline-amd64"
$offlineArm64Dir = Join-Path $releaseDir "offline-arm64"
New-Item -ItemType Directory -Path $offlineAmd64Dir | Out-Null
New-Item -ItemType Directory -Path $offlineArm64Dir | Out-Null

Copy-Item (Join-Path $releaseDir "gost-amd64") (Join-Path $offlineAmd64Dir "flux_agent")
Copy-Item $installPath (Join-Path $offlineAmd64Dir "offline.sh")
Copy-Item (Join-Path $releaseDir "gost-arm64") (Join-Path $offlineArm64Dir "flux_agent")
Copy-Item $installPath (Join-Path $offlineArm64Dir "offline.sh")

Compress-Archive -Path (Join-Path $offlineAmd64Dir "*") -DestinationPath (Join-Path $releaseDir "offline-amd64.zip")
Compress-Archive -Path (Join-Path $offlineArm64Dir "*") -DestinationPath (Join-Path $releaseDir "offline-arm64.zip")

Remove-Item -Recurse -Force $offlineAmd64Dir, $offlineArm64Dir

$releaseNotes = @"
## Release $Version

- Repo: https://github.com/$Repo
- Packages:
  - ghcr.io/ikeilo/flvx-svc-backend:$Version
  - ghcr.io/ikeilo/flvx-svc-frontend:$Version

## Assets

- gost-amd64
- gost-arm64
- gost-amd64.sha256
- gost-arm64.sha256
- install.sh
- panel_install.sh
- docker-compose-v4.yml
- docker-compose-v6.yml
- offline-amd64.zip
- offline-arm64.zip
"@

Set-Content -Path (Join-Path $releaseDir "RELEASE_NOTES.md") -Value $releaseNotes

Write-Host "Release artifacts generated at: $releaseDir"
