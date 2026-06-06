param(
  [string]$Version = "",
  [string]$Remote = "origin",
  [string]$Branch = "main",
  [int]$TimeoutMinutes = 90,
  [switch]$CommitAll,
  [string]$CommitMessage = "",
  [switch]$SkipWait
)

$ErrorActionPreference = "Stop"

function Run-Git {
  param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Args)
  & git @Args
  if ($LASTEXITCODE -ne 0) {
    throw "git $($Args -join ' ') failed with exit code $LASTEXITCODE"
  }
}

function Get-RepoPath {
  param([string]$RemoteName)
  $url = (& git remote get-url $RemoteName).Trim()
  if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($url)) {
    throw "Unable to read git remote '$RemoteName'."
  }

  if ($url -match "github\.com[:/](?<owner>[^/]+)/(?<repo>[^/.]+)(\.git)?$") {
    return "$($Matches.owner)/$($Matches.repo)"
  }

  throw "Remote '$RemoteName' does not look like a GitHub repository: $url"
}

function Invoke-GitHubApi {
  param([string]$Uri)
  $headers = @{ "User-Agent" = "flvxt2-release-script" }
  $token = $env:GITHUB_TOKEN
  if ([string]::IsNullOrWhiteSpace($token)) {
    $token = $env:GH_TOKEN
  }
  if (-not [string]::IsNullOrWhiteSpace($token)) {
    $headers["Authorization"] = "Bearer $token"
  }
  Invoke-RestMethod -Uri $Uri -Headers $headers
}

function Convert-VersionParts {
  param([string]$Tag)
  $normalized = $Tag.TrimStart("v")
  if ($normalized -notmatch "^\d+(\.\d+){0,3}([-.].*)?$") {
    return $null
  }
  try {
    return [version](($normalized -split "[-+]")[0])
  } catch {
    return $null
  }
}

function Get-NextPatchVersion {
  param([string]$RepoPath)
  $tags = @()
  try {
    $releases = Invoke-GitHubApi "https://api.github.com/repos/$RepoPath/releases?per_page=50"
    $tags += @($releases | ForEach-Object { $_.tag_name })
  } catch {
    Write-Warning "Could not read releases from GitHub, falling back to local tags: $($_.Exception.Message)"
  }

  $tags += @(& git tag --list)
  $latest = $tags |
    Where-Object { -not [string]::IsNullOrWhiteSpace($_) } |
    ForEach-Object {
      $parsed = Convert-VersionParts $_
      if ($null -ne $parsed) {
        [pscustomobject]@{ Tag = $_; Version = $parsed }
      }
    } |
    Sort-Object Version -Descending |
    Select-Object -First 1

  if ($null -eq $latest) {
    return "1.0.0"
  }

  return "$($latest.Version.Major).$($latest.Version.Minor).$($latest.Version.Build + 1)"
}

function Test-ReleaseAssets {
  param([string]$RepoPath, [string]$Tag)
  $release = Invoke-GitHubApi "https://api.github.com/repos/$RepoPath/releases/tags/$Tag"
  $required = @(
    "docker-compose-v4.yml",
    "docker-compose-v6.yml",
    "gost-amd64",
    "gost-arm64",
    "gost-amd64.sha256",
    "gost-arm64.sha256",
    "install.sh",
    "panel_install.sh",
    "offline-amd64.zip",
    "offline-arm64.zip"
  )
  $assetNames = @($release.assets | ForEach-Object { $_.name })
  $missing = @($required | Where-Object { $assetNames -notcontains $_ })
  if ($missing.Count -gt 0) {
    throw "Release $Tag is missing assets: $($missing -join ', ')"
  }
  return $release
}

function Wait-ReleaseWorkflow {
  param([string]$RepoPath, [string]$Tag, [string]$HeadSha, [int]$TimeoutMinutes)
  $deadline = (Get-Date).AddMinutes($TimeoutMinutes)
  $run = $null

  while ((Get-Date) -lt $deadline) {
    $runs = Invoke-GitHubApi "https://api.github.com/repos/$RepoPath/actions/runs?per_page=20"
    $run = @($runs.workflow_runs | Where-Object {
      $_.name -eq "Build and Push Images" -and
      $_.head_branch -eq $Tag -and
      $_.head_sha -eq $HeadSha
    } | Select-Object -First 1)

    if ($run) {
      Write-Host "Workflow: $($run.status) / $($run.conclusion) - $($run.html_url)"
      if ($run.status -eq "completed") {
        if ($run.conclusion -ne "success") {
          throw "Release workflow failed with conclusion '$($run.conclusion)': $($run.html_url)"
        }
        return $run
      }
    } else {
      Write-Host "Waiting for release workflow to appear for tag $Tag..."
    }

    Start-Sleep -Seconds 20
  }

  throw "Timed out waiting for release workflow after $TimeoutMinutes minutes."
}

$repoPath = Get-RepoPath $Remote
$currentBranch = (& git rev-parse --abbrev-ref HEAD).Trim()
if ($currentBranch -ne $Branch) {
  throw "Current branch is '$currentBranch', expected '$Branch'."
}

if ($CommitAll) {
  if ([string]::IsNullOrWhiteSpace($CommitMessage)) {
    throw "-CommitMessage is required when using -CommitAll."
  }
  Run-Git add -A
  $pending = (& git status --short)
  if (-not [string]::IsNullOrWhiteSpace($pending)) {
    Run-Git commit -m $CommitMessage
  }
}

$dirty = (& git status --short)
if (-not [string]::IsNullOrWhiteSpace($dirty)) {
  throw "Working tree is not clean. Commit or stash changes before release."
}

Run-Git fetch $Remote --tags
Run-Git pull --rebase $Remote $Branch

if ([string]::IsNullOrWhiteSpace($Version)) {
  $Version = Get-NextPatchVersion $repoPath
}

if ($Version -notmatch "^\d+\.\d+\.\d+([-.][0-9A-Za-z.-]+)?$") {
  throw "Version '$Version' is not a valid release tag. Expected something like 3.0.18."
}

$existingLocal = (& git tag --list $Version).Trim()
if (-not [string]::IsNullOrWhiteSpace($existingLocal)) {
  throw "Local tag '$Version' already exists."
}

$existingRemote = (& git ls-remote --tags $Remote "refs/tags/$Version").Trim()
if (-not [string]::IsNullOrWhiteSpace($existingRemote)) {
  throw "Remote tag '$Version' already exists."
}

$headSha = (& git rev-parse HEAD).Trim()
Run-Git push $Remote $Branch
Run-Git tag -a $Version -m "Release $Version"
Run-Git push $Remote $Version

Write-Host "Release tag pushed: $Version ($headSha)"

if (-not $SkipWait) {
  $run = Wait-ReleaseWorkflow $repoPath $Version $headSha $TimeoutMinutes
  $release = Test-ReleaseAssets $repoPath $Version
  Write-Host "Release completed: $($release.html_url)"
  Write-Host "Workflow completed: $($run.html_url)"
} else {
  Write-Host "Skipped workflow wait. Check: https://github.com/$repoPath/actions"
}
