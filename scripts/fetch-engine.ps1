<#
.SYNOPSIS
    Copies the Chrome Screen AI runtime out of a local browser profile into
    engine-runtime\ so Tank-OCR can run without depending on that profile.

.DESCRIPTION
    Tank-OCR does not ship the OCR runtime. The runtime is a Chrome component:
    Google distributes it through Chrome's component updater, it lands in the
    browser profile, and Tank-OCR reads it from there.

    You only need this script when the profile copy is not usable directly:

      * deploying to a machine that has no Chrome installed
      * pinning a known runtime version so a browser update cannot change it
      * running as a service account that cannot read the user's profile

    Otherwise just run tank-ocr.exe — it finds the profile copy by itself.

.PARAMETER Destination
    Where to put the runtime. Defaults to engine-runtime\ beside the repository
    root, which tank-ocr.exe searches automatically.

.PARAMETER Source
    Copy from this directory instead of auto-detecting. Point it at a
    screen_ai\<version>\ folder.

.EXAMPLE
    .\scripts\fetch-engine.ps1

.EXAMPLE
    .\scripts\fetch-engine.ps1 -Destination D:\deploy\tank-ocr\engine-runtime
#>
[CmdletBinding()]
param(
    [string]$Destination,
    [string]$Source
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
if (-not $Destination) {
    $Destination = Join-Path $repoRoot 'engine-runtime'
}

# Library file names, newest naming first. Older private builds renamed it.
$libNames = @('chrome_screen_ai.dll', 'kichonai.dll')

function Test-RuntimeDir {
    param([string]$Dir)
    if (-not $Dir -or -not (Test-Path -LiteralPath $Dir)) { return $false }
    foreach ($n in $libNames) {
        if (Test-Path -LiteralPath (Join-Path $Dir $n)) { return $true }
    }
    return $false
}

# Version directories are named "148.12" — compare numerically, because a
# string sort would rank "9.5" above "148.12".
function Get-NewestVersionDir {
    param([string]$Root)
    if (-not (Test-Path -LiteralPath $Root)) { return $null }
    $dirs = Get-ChildItem -LiteralPath $Root -Directory -ErrorAction SilentlyContinue
    if (-not $dirs) { return $null }
    $best = $dirs | Sort-Object -Property @{ Expression = {
        $parts = $_.Name -split '\.'
        $v = 0
        foreach ($p in $parts) {
            $n = 0
            [void][int]::TryParse($p, [ref]$n)
            $v = $v * 100000 + $n
        }
        $v
    }} -Descending | Select-Object -First 1
    return $best.FullName
}

function Find-BrowserRuntime {
    $la = $env:LOCALAPPDATA
    $roots = @(
        (Join-Path $la 'Google\Chrome\User Data\screen_ai'),
        (Join-Path $la 'Google\Chrome Beta\User Data\screen_ai'),
        (Join-Path $la 'Google\Chrome Dev\User Data\screen_ai'),
        (Join-Path $la 'Microsoft\Edge\User Data\screen_ai'),
        (Join-Path $la 'Chromium\User Data\screen_ai')
    )
    foreach ($r in $roots) {
        $d = Get-NewestVersionDir -Root $r
        if (Test-RuntimeDir -Dir $d) { return $d }
    }
    return $null
}

Write-Host ''
Write-Host '=== Tank-OCR: fetch OCR runtime ===' -ForegroundColor Cyan
Write-Host ''

if ($Source) {
    if (-not (Test-RuntimeDir -Dir $Source)) {
        throw "No OCR runtime in '$Source' (expected one of: $($libNames -join ', '))."
    }
    $found = $Source
} else {
    $found = Find-BrowserRuntime
}

if (-not $found) {
    Write-Host 'No Screen AI runtime found in any browser profile.' -ForegroundColor Yellow
    Write-Host ''
    Write-Host 'Chrome downloads this component on demand, so a fresh install may'
    Write-Host 'not have it yet. To make Chrome fetch it:'
    Write-Host ''
    Write-Host '  1. Open Chrome and go to  chrome://components'
    Write-Host '  2. Find "Screen AI" (also listed as "Optical character recognition")'
    Write-Host '  3. Click "Check for update" and wait until a version number appears'
    Write-Host ''
    Write-Host 'Then run this script again. Alternatively, copy the folder from a'
    Write-Host 'machine that has it and pass -Source:'
    Write-Host ''
    Write-Host '  .\scripts\fetch-engine.ps1 -Source "D:\screen_ai\148.12"'
    Write-Host ''
    exit 1
}

$version = Split-Path -Leaf $found
$size = (Get-ChildItem -LiteralPath $found -Recurse -File | Measure-Object -Property Length -Sum).Sum
$count = (Get-ChildItem -LiteralPath $found -Recurse -File | Measure-Object).Count

Write-Host ("Source : {0}" -f $found)
Write-Host ("Version: {0}" -f $version)
Write-Host ("Size   : {0:N0} MB in {1} files" -f ($size / 1MB), $count)
Write-Host ("Target : {0}" -f $Destination)
Write-Host ''

if (Test-Path -LiteralPath $Destination) {
    Write-Host 'Target exists. Replacing its contents.' -ForegroundColor Yellow
    Remove-Item -LiteralPath $Destination -Recurse -Force
}
New-Item -ItemType Directory -Path $Destination -Force | Out-Null

Write-Host 'Copying...'
Copy-Item -Path (Join-Path $found '*') -Destination $Destination -Recurse -Force

if (-not (Test-RuntimeDir -Dir $Destination)) {
    throw "Copy finished but no library is present in '$Destination'."
}

# Record where this came from, so the copy can be traced and refreshed later.
$stamp = [ordered]@{
    source           = $found
    version          = $version
    copied_utc       = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
    note             = 'Chrome Screen AI component, copied from a local browser profile. Not covered by the Tank-OCR MIT license; see NOTICE.'
}
$stamp | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $Destination 'tank-ocr-source.json') -Encoding utf8

Write-Host ''
Write-Host ("Done. Runtime {0} is now in {1}" -f $version, $Destination) -ForegroundColor Green
Write-Host 'Start the server with run_ocr.bat.'
Write-Host ''
Write-Host 'Note: this runtime is Google-licensed and is NOT covered by the MIT' -ForegroundColor Yellow
Write-Host 'license of Tank-OCR. Do not commit it or redistribute it.' -ForegroundColor Yellow
Write-Host ''
