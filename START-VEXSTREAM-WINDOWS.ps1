# VexStream Music V4 — human-first Windows source launcher
# No Git, Python, or preinstalled Go is required.
[CmdletBinding()]
param(
  [string]$SourceRef = 'main'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$Repository = 'vgong24/VexstreamMusic'
$GoVersion = '1.23.2'
$GoArchive = "go$GoVersion.windows-amd64.zip"
$GoArchiveSha256 = 'bc28fe3002cd65cec65d0e4f6000584dacb8c71bfaff8801dfb532855ca42513'
$Headers = @{ 'User-Agent' = 'VexStream-Music-human-launcher' }

function Invoke-Native {
  param(
    [Parameter(Mandatory=$true)][string]$FilePath,
    [Parameter(Mandatory=$true)][string[]]$ArgumentList,
    [Parameter(Mandatory=$true)][string]$WorkingDirectory,
    [hashtable]$Environment = @{}
  )

  $priorPreference = $ErrorActionPreference
  $priorValues = @{}
  try {
    foreach ($key in $Environment.Keys) {
      $priorValues[$key] = [Environment]::GetEnvironmentVariable($key, 'Process')
      [Environment]::SetEnvironmentVariable($key, [string]$Environment[$key], 'Process')
    }
    # Windows PowerShell 5 can promote healthy native stderr to ErrorRecord.
    # Native exit code is the process truth for these source-owned commands.
    $ErrorActionPreference = 'Continue'
    Push-Location $WorkingDirectory
    try {
      & $FilePath @ArgumentList
      $code = $LASTEXITCODE
    } finally {
      Pop-Location
    }
  } finally {
    $ErrorActionPreference = $priorPreference
    foreach ($key in $Environment.Keys) {
      [Environment]::SetEnvironmentVariable($key, $priorValues[$key], 'Process')
    }
  }
  if ($null -eq $code -or $code -ne 0) {
    throw "$FilePath exited with code $code"
  }
}

function Get-ExactCommit {
  param([string]$Ref)
  $encoded = [Uri]::EscapeDataString($Ref)
  $value = Invoke-RestMethod -Headers $Headers -Uri "https://api.github.com/repos/$Repository/commits/$encoded"
  if (-not $value.sha -or $value.sha -notmatch '^[a-f0-9]{40}$') {
    throw "GitHub did not return an exact commit for '$Ref'."
  }
  return [string]$value.sha
}

function Ensure-Go {
  $toolchains = Join-Path $env:LOCALAPPDATA 'VexStreamMusic\toolchains'
  $goRoot = Join-Path $toolchains "go$GoVersion-windows-amd64"
  $goExe = Join-Path $goRoot 'go\bin\go.exe'
  if (Test-Path -LiteralPath $goExe -PathType Leaf) {
    return $goExe
  }

  New-Item -ItemType Directory -Path $toolchains -Force | Out-Null
  $temp = Join-Path $env:TEMP ("vexstream-go-$([Guid]::NewGuid().ToString('N'))")
  New-Item -ItemType Directory -Path $temp | Out-Null
  try {
    $archive = Join-Path $temp $GoArchive
    Write-Host "VexStream: obtaining private Go $GoVersion build tool..."
    Invoke-WebRequest -UseBasicParsing -Uri "https://go.dev/dl/$GoArchive" -OutFile $archive
    $observed = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($observed -ne $GoArchiveSha256) {
      throw "Go archive digest mismatch: $observed"
    }
    $expanded = Join-Path $temp 'expanded'
    Expand-Archive -LiteralPath $archive -DestinationPath $expanded -Force
    if (-not (Test-Path -LiteralPath (Join-Path $expanded 'go\bin\go.exe') -PathType Leaf)) {
      throw 'Downloaded Go archive did not contain go\bin\go.exe'
    }
    if (Test-Path -LiteralPath $goRoot) {
      Remove-Item -LiteralPath $goRoot -Recurse -Force
    }
    New-Item -ItemType Directory -Path $goRoot | Out-Null
    Move-Item -LiteralPath (Join-Path $expanded 'go') -Destination $goRoot
  } finally {
    Remove-Item -LiteralPath $temp -Recurse -Force -ErrorAction SilentlyContinue
  }
  if (-not (Test-Path -LiteralPath $goExe -PathType Leaf)) {
    throw 'Private Go toolchain formation failed.'
  }
  return $goExe
}

$commit = Get-ExactCommit -Ref $SourceRef
$installRoot = Join-Path $env:LOCALAPPDATA "VexStreamMusic\source-builds\$commit"
$exe = Join-Path $installRoot 'VexStreamMusic-4.0.0.exe'
$receipt = Join-Path $installRoot 'BUILD-INFO.json'

if ((Test-Path -LiteralPath $exe -PathType Leaf) -and (Test-Path -LiteralPath $receipt -PathType Leaf)) {
  try {
    $info = Get-Content -LiteralPath $receipt -Raw | ConvertFrom-Json
    if ($info.sourceCommit -eq $commit -and $info.exeSha256) {
      $observed = (Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash.ToLowerInvariant()
      if ($observed -eq [string]$info.exeSha256) {
        Write-Host "VexStream: launching cached V4 build from $commit"
        Start-Process -FilePath $exe
        exit 0
      }
    }
  } catch {
    Write-Host 'VexStream: cached build receipt is not reusable; rebuilding.'
  }
}

$go = Ensure-Go
$goBin = Split-Path -Parent $go
$goFmt = Join-Path $goBin 'gofmt.exe'
$work = Join-Path $env:TEMP ("vexstream-v4-source-$([Guid]::NewGuid().ToString('N'))")
New-Item -ItemType Directory -Path $work | Out-Null

try {
  $sourceZip = Join-Path $work 'source.zip'
  Write-Host "VexStream: downloading exact source $commit..."
  Invoke-WebRequest -UseBasicParsing -Headers $Headers -Uri "https://github.com/$Repository/archive/$commit.zip" -OutFile $sourceZip
  $expanded = Join-Path $work 'source'
  Expand-Archive -LiteralPath $sourceZip -DestinationPath $expanded -Force
  $sourceRoot = Get-ChildItem -LiteralPath $expanded -Directory | Select-Object -First 1
  if ($null -eq $sourceRoot) {
    throw 'VexStream source archive did not contain a repository root.'
  }
  $sourceRoot = $sourceRoot.FullName

  $goFiles = Get-ChildItem -LiteralPath (Join-Path $sourceRoot 'v2') -Filter '*.go' -Recurse -File | Sort-Object FullName
  $formatProblems = New-Object System.Collections.Generic.List[string]
  foreach ($goFile in $goFiles) {
    $priorPreference = $ErrorActionPreference
    try {
      $ErrorActionPreference = 'Continue'
      $delta = & $goFmt -d $goFile.FullName 2>&1 | Out-String
      $formatCode = $LASTEXITCODE
    } finally {
      $ErrorActionPreference = $priorPreference
    }
    if ($formatCode -ne 0) {
      throw "gofmt failed for $($goFile.FullName) with exit $formatCode"
    }
    if (-not [string]::IsNullOrWhiteSpace($delta)) {
      [void]$formatProblems.Add($delta)
    }
  }
  if ($formatProblems.Count -gt 0) {
    $formatProblems | ForEach-Object { [Console]::Error.WriteLine($_) }
    throw "V4 source is not gofmt-clean and will not be built by the human launcher."
  }

  Write-Host 'VexStream: checking V4 source...'
  Invoke-Native -FilePath $go -ArgumentList @('test','./v2/...') -WorkingDirectory $sourceRoot

  $buildEnv = @{ GOOS='windows'; GOARCH='amd64'; CGO_ENABLED='0' }
  Invoke-Native -FilePath $go -ArgumentList @('vet','./v2/...') -WorkingDirectory $sourceRoot -Environment $buildEnv

  New-Item -ItemType Directory -Path $installRoot -Force | Out-Null
  $temporaryExe = Join-Path $installRoot 'VexStreamMusic-4.0.0.forming.exe'
  Remove-Item -LiteralPath $temporaryExe -Force -ErrorAction SilentlyContinue

  Write-Host 'VexStream: building the Windows app...'
  Invoke-Native -FilePath $go -ArgumentList @(
    'build',
    '-trimpath',
    '-ldflags', "-H=windowsgui -s -w -X main.buildRef=$commit",
    '-o', $temporaryExe,
    './v2/cmd/vexstream-v4'
  ) -WorkingDirectory $sourceRoot -Environment $buildEnv

  Move-Item -LiteralPath $temporaryExe -Destination $exe -Force
  $exeSha = (Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash.ToLowerInvariant()
  [ordered]@{
    schemaVersion = 'vexstream.human-source-build/v1'
    appVersion = '4.0.0'
    sourceRef = $SourceRef
    sourceCommit = $commit
    goVersion = "go$GoVersion"
    platform = 'windows-amd64'
    exeSha256 = $exeSha
  } | ConvertTo-Json | Set-Content -LiteralPath $receipt -Encoding UTF8
} finally {
  Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host "VexStream: ready."
Write-Host "Source: $commit"
Write-Host "App:    $exe"
Start-Process -FilePath $exe
