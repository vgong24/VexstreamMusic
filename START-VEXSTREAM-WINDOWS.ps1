# VexStream Music V4 - human-first Windows source launcher
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
$GoArchiveBytes = 81923109
$Headers = @{ 'User-Agent' = 'VexStream-Music-human-launcher' }

trap {
  Write-Progress -Activity 'VexStream setup' -Completed -ErrorAction SilentlyContinue
  Write-Host ''
  Write-Host 'VexStream setup stopped safely.' -ForegroundColor Red
  Write-Host "Reason: $($_.Exception.Message)"
  Write-Host ''
  Write-Host 'Your existing VexStream library and any previously verified app build were left in place.'
  Write-Host 'No setup step intentionally uploads your music or personal files.'
  exit 1
}

function Format-ByteSize {
  param([long]$Bytes)
  if ($Bytes -ge 1GB) { return ('{0:N1} GiB' -f ($Bytes / 1GB)) }
  if ($Bytes -ge 1MB) { return ('{0:N1} MiB' -f ($Bytes / 1MB)) }
  if ($Bytes -ge 1KB) { return ('{0:N1} KiB' -f ($Bytes / 1KB)) }
  return "$Bytes bytes"
}

function Invoke-ProgressDownload {
  param(
    [Parameter(Mandatory=$true)][string]$Uri,
    [Parameter(Mandatory=$true)][string]$OutFile,
    [Parameter(Mandatory=$true)][string]$Label,
    [long]$ExpectedBytes = 0,
    [hashtable]$RequestHeaders = @{}
  )

  Add-Type -AssemblyName System.Net.Http
  $handler = [System.Net.Http.HttpClientHandler]::new()
  $handler.AllowAutoRedirect = $true
  $client = [System.Net.Http.HttpClient]::new($handler)
  $request = $null
  $response = $null
  $stream = $null
  $file = $null
  [long]$written = 0
  try {
    $request = [System.Net.Http.HttpRequestMessage]::new([System.Net.Http.HttpMethod]::Get, $Uri)
    [void]$request.Headers.TryAddWithoutValidation('User-Agent', 'VexStream-Music-human-launcher')
    foreach ($key in $RequestHeaders.Keys) {
      if ($key -ieq 'User-Agent') { continue }
      [void]$request.Headers.TryAddWithoutValidation($key, [string]$RequestHeaders[$key])
    }

    $response = $client.SendAsync(
      $request,
      [System.Net.Http.HttpCompletionOption]::ResponseHeadersRead
    ).GetAwaiter().GetResult()
    [void]$response.EnsureSuccessStatusCode()

    [long]$reportedBytes = 0
    $contentLength = $response.Content.Headers.ContentLength
    if ($null -ne $contentLength) {
      $reportedBytes = [long]$contentLength
    }
    [long]$totalBytes = if ($ExpectedBytes -gt 0) { $ExpectedBytes } else { $reportedBytes }
    if ($totalBytes -gt 0) {
      Write-Host "      Expected transfer: $(Format-ByteSize $totalBytes) ($totalBytes bytes)"
    } else {
      Write-Host '      Expected transfer: server did not provide a total size; downloaded bytes will still be shown.'
    }

    $stream = $response.Content.ReadAsStreamAsync().GetAwaiter().GetResult()
    $file = [System.IO.File]::Open(
      $OutFile,
      [System.IO.FileMode]::Create,
      [System.IO.FileAccess]::Write,
      [System.IO.FileShare]::None
    )
    $buffer = New-Object byte[] (1024 * 1024)

    while (($read = $stream.Read($buffer, 0, $buffer.Length)) -gt 0) {
      $file.Write($buffer, 0, $read)
      $written += $read

      if ($totalBytes -gt 0) {
        $percent = [Math]::Min(100, [Math]::Floor(($written * 100.0) / $totalBytes))
        $status = "$(Format-ByteSize $written) / $(Format-ByteSize $totalBytes)  ($percent%)"
        Write-Progress -Activity "Downloading $Label" -Status $status -PercentComplete $percent
      } else {
        Write-Progress -Activity "Downloading $Label" -Status "$(Format-ByteSize $written) downloaded" -PercentComplete -1
      }
    }
    Write-Progress -Activity "Downloading $Label" -Completed
  } finally {
    if ($file) { $file.Dispose() }
    if ($stream) { $stream.Dispose() }
    if ($response) { $response.Dispose() }
    if ($request) { $request.Dispose() }
    if ($client) { $client.Dispose() }
    if ($handler) { $handler.Dispose() }
  }

  if ($ExpectedBytes -gt 0 -and $written -ne $ExpectedBytes) {
    throw "$Label download size mismatch: expected $ExpectedBytes bytes, received $written bytes"
  }
  return $written
}

function Invoke-Native {
  param(
    [Parameter(Mandatory=$true)][string]$FilePath,
    [Parameter(Mandatory=$true)][string[]]$ArgumentList,
    [Parameter(Mandatory=$true)][string]$WorkingDirectory,
    [hashtable]$Environment = @{},
    [switch]$Quiet
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
      if ($Quiet) {
        $nativeOutput = (& $FilePath @ArgumentList 2>&1 | Out-String)
      } else {
        & $FilePath @ArgumentList
        $nativeOutput = ''
      }
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
    $detail = if ($Quiet -and -not [string]::IsNullOrWhiteSpace($nativeOutput)) {
      $text = $nativeOutput.Trim()
      if ($text.Length -gt 4000) { $text.Substring($text.Length - 4000) } else { $text }
    } else {
      ''
    }
    if ([string]::IsNullOrWhiteSpace($detail)) {
      throw "$FilePath exited with code $code"
    }
    throw ("$FilePath exited with code $code." + [Environment]::NewLine + $detail)
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

function Get-RunningV4 {
  $currentUrlPath = Join-Path $env:LOCALAPPDATA 'VexStreamMusic\v4\current-url.txt'
  if (-not (Test-Path -LiteralPath $currentUrlPath -PathType Leaf)) {
    return $null
  }
  try {
    $url = (Get-Content -LiteralPath $currentUrlPath -Raw).Trim()
    if ([string]::IsNullOrWhiteSpace($url)) { return $null }
    $health = Invoke-RestMethod -UseBasicParsing -Uri ($url.TrimEnd('/') + '/health') -TimeoutSec 2
    if ($health.ok -ne $true -or [string]$health.version -ne '4.0.0') { return $null }
    return [pscustomobject]@{
      Url = $url
      Version = [string]$health.version
      BuildRef = [string]$health.buildRef
    }
  } catch {
    return $null
  }
}

function Ensure-Go {
  $toolchains = Join-Path $env:LOCALAPPDATA 'VexStreamMusic\toolchains'
  $goRoot = Join-Path $toolchains "go$GoVersion-windows-amd64"
  $goExe = Join-Path $goRoot 'go\bin\go.exe'
  $toolchainReceipt = Join-Path $goRoot 'TOOLCHAIN-INFO.json'
  if ((Test-Path -LiteralPath $goExe -PathType Leaf) -and (Test-Path -LiteralPath $toolchainReceipt -PathType Leaf)) {
    try {
      $toolInfo = Get-Content -LiteralPath $toolchainReceipt -Raw | ConvertFrom-Json
      $goExeSha = (Get-FileHash -LiteralPath $goExe -Algorithm SHA256).Hash.ToLowerInvariant()
      $priorPreference = $ErrorActionPreference
      try {
        $ErrorActionPreference = 'Continue'
        $goVersionOutput = (& $goExe version 2>&1 | Out-String).Trim()
        $goVersionCode = $LASTEXITCODE
      } finally {
        $ErrorActionPreference = $priorPreference
      }
      $receiptIdentityMatches = (
        ([string]$toolInfo.goArchiveSha256 -eq $GoArchiveSha256) -or
        ([string]$toolInfo.acquisition -eq 'EXISTING_MANAGED_PATH_VERSION_QUALIFIED')
      )
      if (
        $goVersionCode -eq 0 -and
        $goVersionOutput -match 'go1\.23\.2\s+windows/amd64' -and
        $receiptIdentityMatches -and
        [string]$toolInfo.goExeSha256 -eq $goExeSha
      ) {
        Write-Host ''
        Write-Host '[2/5] Local build tool is already prepared'
        Write-Host "      Go ${GoVersion}: verified cached copy"
        Write-Host "      Storage: $goRoot"
        return $goExe
      }
    } catch {
      # Fall through to a checksum-bound refresh below.
    }
    Write-Host ''
    Write-Host '[2/5] Refreshing the local build tool'
    Write-Host '      The cached build tool could not be re-verified, so VexStream will replace only that tool.'
  } elseif (Test-Path -LiteralPath $goExe -PathType Leaf) {
    $priorPreference = $ErrorActionPreference
    try {
      $ErrorActionPreference = 'Continue'
      $legacyVersionOutput = (& $goExe version 2>&1 | Out-String).Trim()
      $legacyVersionCode = $LASTEXITCODE
    } finally {
      $ErrorActionPreference = $priorPreference
    }
    if ($legacyVersionCode -eq 0 -and $legacyVersionOutput -match 'go1\.23\.2\s+windows/amd64') {
      $goExeSha = (Get-FileHash -LiteralPath $goExe -Algorithm SHA256).Hash.ToLowerInvariant()
      $sizeMeasure = Get-ChildItem -LiteralPath (Join-Path $goRoot 'go') -File -Recurse | Measure-Object -Property Length -Sum
      [long]$installedBytes = $sizeMeasure.Sum
      [ordered]@{
        schemaVersion = 'vexstream.local-toolchain/v1'
        goVersion = $GoVersion
        platform = 'windows-amd64'
        acquisition = 'EXISTING_MANAGED_PATH_VERSION_QUALIFIED'
        goArchiveSha256 = $null
        goArchiveBytes = $null
        goExeSha256 = $goExeSha
        installedBytes = $installedBytes
      } | ConvertTo-Json | Set-Content -LiteralPath $toolchainReceipt -Encoding UTF8
      Write-Host ''
      Write-Host '[2/5] Existing local build tool found'
      Write-Host "      Go $GoVersion is already present in VexStream's managed toolchain folder."
      Write-Host '      Version/platform check passed; no repeat download is needed.'
      Write-Host "      Local size: $(Format-ByteSize $installedBytes)"
      return $goExe
    }
    Write-Host ''
    Write-Host '[2/5] Refreshing the local build tool'
    Write-Host '      An older cached tool could not prove the expected Go version/platform.'
  }

  New-Item -ItemType Directory -Path $toolchains -Force | Out-Null
  $temp = Join-Path $env:TEMP ("vexstream-go-$([Guid]::NewGuid().ToString('N'))")
  New-Item -ItemType Directory -Path $temp | Out-Null
  try {
    $archive = Join-Path $temp $GoArchive
    Write-Host ''
    Write-Host '[2/5] Preparing the local build tool'
    Write-Host "      Download: Go $GoVersion for Windows"
    Write-Host '      Source:   https://go.dev (official Go project)'
    Write-Host "      Size:     $(Format-ByteSize $GoArchiveBytes) ($GoArchiveBytes bytes)"
    Write-Host "      Storage:  $goRoot"
    Write-Host '      Why:      VexStream is open source. This tool compiles that source into your local app.'
    Write-Host '      Safety:   This download does not scan or upload your personal files.'
    Write-Host '      Frequency: one-time for this build-tool version.'
    Write-Host ''
    [void](Invoke-ProgressDownload -Uri "https://go.dev/dl/$GoArchive" -OutFile $archive -Label "Go $GoVersion build tool" -ExpectedBytes $GoArchiveBytes)
    Write-Host "      Download complete: $(Format-ByteSize $GoArchiveBytes)."
    Write-Host '      Verifying the trusted checksum...'
    $observed = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($observed -ne $GoArchiveSha256) {
      throw "Go archive digest mismatch: $observed"
    }
    Write-Host '      Verification passed.'
    Write-Host '      Unpacking the local build tool...'
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
    $sizeMeasure = Get-ChildItem -LiteralPath (Join-Path $goRoot 'go') -File -Recurse | Measure-Object -Property Length -Sum
    [long]$installedBytes = $sizeMeasure.Sum
    $goExeSha = (Get-FileHash -LiteralPath $goExe -Algorithm SHA256).Hash.ToLowerInvariant()
    [ordered]@{
      schemaVersion = 'vexstream.local-toolchain/v1'
      goVersion = $GoVersion
      platform = 'windows-amd64'
      acquisition = 'CHECKSUM_VERIFIED_GO_ARCHIVE'
      goArchiveSha256 = $GoArchiveSha256
      goArchiveBytes = $GoArchiveBytes
      goExeSha256 = $goExeSha
      installedBytes = $installedBytes
    } | ConvertTo-Json | Set-Content -LiteralPath $toolchainReceipt -Encoding UTF8
    Write-Host "      Local build tool ready: $(Format-ByteSize $installedBytes) on disk."
  } finally {
    Remove-Item -LiteralPath $temp -Recurse -Force -ErrorAction SilentlyContinue
  }
  if (-not (Test-Path -LiteralPath $goExe -PathType Leaf)) {
    throw 'Private Go toolchain formation failed.'
  }
  return $goExe
}

Write-Host ''
Write-Host 'VexStream Music 4.0 - local source launcher'
Write-Host 'This launcher uses GitHub and go.dev only for the setup files described below.'
Write-Host 'Your music library remains local to VexStream.'
Write-Host ''
Write-Host '[1/5] Checking which VexStream source you asked for'
$commit = Get-ExactCommit -Ref $SourceRef
Write-Host "      Source:  https://github.com/$Repository"
Write-Host "      Ref:     $SourceRef"
Write-Host "      Commit:  $commit"
$buildRoot = Join-Path $env:LOCALAPPDATA 'VexStreamMusic\source-builds'
$installRoot = Join-Path $buildRoot $commit
$exe = Join-Path $installRoot 'VexStreamMusic-4.0.0.exe'
$receipt = Join-Path $installRoot 'BUILD-INFO.json'

$previousBuilds = @()
if (Test-Path -LiteralPath $buildRoot -PathType Container) {
  $previousBuilds = @(
    Get-ChildItem -LiteralPath $buildRoot -Filter 'BUILD-INFO.json' -File -Recurse -ErrorAction SilentlyContinue |
      Where-Object { $_.FullName -ne $receipt }
  )
}
if ($previousBuilds.Count -gt 0) {
  Write-Host ''
  Write-Host '      Update path detected.'
  Write-Host '      A previous local VexStream build is present and will be kept as a rollback copy.'
  Write-Host '      The new build will not overwrite your music library.'
} else {
  Write-Host ''
  Write-Host '      First local VexStream source build on this computer.'
}

$runningV4 = Get-RunningV4
if ($null -ne $runningV4) {
  Write-Host ''
  Write-Host '      VexStream 4.0 is currently running.'
  Write-Host '      It can stay open while this source is prepared.'
  Write-Host '      If this produces a different build, VexStream will ask before switching.'
}

if ((Test-Path -LiteralPath $exe -PathType Leaf) -and (Test-Path -LiteralPath $receipt -PathType Leaf)) {
  try {
    $info = Get-Content -LiteralPath $receipt -Raw | ConvertFrom-Json
    if ($info.sourceCommit -eq $commit -and $info.exeSha256) {
      $observed = (Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash.ToLowerInvariant()
      if ($observed -eq [string]$info.exeSha256) {
        Write-Host ''
        Write-Host '      Already installed and verified for this exact source.'
        Write-Host '      No downloads or rebuild are needed.'
        Write-Host ''
        Write-Host '[5/5] Launching VexStream'
        Write-Host "      App: $exe"
        Start-Process -FilePath $exe
        exit 0
      }
    }
  } catch {
    Write-Host '      The cached app receipt could not be verified; rebuilding this source safely.'
  }
} elseif (Test-Path -LiteralPath $exe -PathType Leaf) {
  Write-Host '      An unverified cached app was found for this source; rebuilding it before launch.'
}

$go = Ensure-Go
$goBin = Split-Path -Parent $go
$goFmt = Join-Path $goBin 'gofmt.exe'
$work = Join-Path $env:TEMP ("vexstream-v4-source-$([Guid]::NewGuid().ToString('N'))")
New-Item -ItemType Directory -Path $work | Out-Null

try {
  $sourceZip = Join-Path $work 'source.zip'
  Write-Host ''
  Write-Host '[3/5] Downloading the exact VexStream source'
  Write-Host "      Source: https://github.com/$Repository"
  Write-Host "      Commit: $commit"
  Write-Host '      Why:    these are the exact files used to build your VexStream app.'
  Write-Host '      Storage: temporary; the source archive and extracted build copy are removed afterward.'
  Write-Host ''
  $sourceBytes = Invoke-ProgressDownload -RequestHeaders $Headers -Uri "https://github.com/$Repository/archive/$commit.zip" -OutFile $sourceZip -Label 'VexStream source'
  Write-Host "      Download complete: $(Format-ByteSize $sourceBytes)."
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

  Write-Host ''
  Write-Host '[4/5] Checking and building VexStream'
  Write-Host '      Running source safety checks...'
  Invoke-Native -FilePath $go -ArgumentList @('test','./v2/...') -WorkingDirectory $sourceRoot -Quiet

  $buildEnv = @{ GOOS='windows'; GOARCH='amd64'; CGO_ENABLED='0' }
  Invoke-Native -FilePath $go -ArgumentList @('vet','./v2/...') -WorkingDirectory $sourceRoot -Environment $buildEnv -Quiet

  New-Item -ItemType Directory -Path $installRoot -Force | Out-Null
  $temporaryExe = Join-Path $installRoot 'VexStreamMusic-4.0.0.forming.exe'
  Remove-Item -LiteralPath $temporaryExe -Force -ErrorAction SilentlyContinue

  Write-Host '      Source checks passed.'
  Write-Host '      Building VexStreamMusic-4.0.0.exe...'
  Invoke-Native -FilePath $go -ArgumentList @(
    'build',
    '-trimpath',
    '-ldflags', "-H=windowsgui -s -w -X main.buildRef=$commit",
    '-o', $temporaryExe,
    './v2/cmd/vexstream-v4'
  ) -WorkingDirectory $sourceRoot -Environment $buildEnv -Quiet

  Write-Host '      Build complete.'
  Move-Item -LiteralPath $temporaryExe -Destination $exe -Force
  $exeBytes = (Get-Item -LiteralPath $exe).Length
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

Write-Host ''
Write-Host '[5/5] VexStream is ready - launching now'
Write-Host "      Source: $commit"
Write-Host "      App:    $exe"
Write-Host "      Size:   $(Format-ByteSize $exeBytes)"
Write-Host ''
Write-Host 'The browser should open automatically. Your existing VexStream library should remain available.'
Start-Process -FilePath $exe
