<#
.SYNOPSIS
  Runs the scrubber (monsters or items) against a local 5etools checkout,
  sourcing its MongoDB/Typesense credentials from either .env (dev) or
  .env.production (prod).

.DESCRIPTION
  The scrubber (cmd/scrubber) reads MONGODB_URI, TYPESENSE_URL, and
  TYPESENSE_API_KEY from the environment - it does not load an env file
  itself. This script picks which file to load based on -Environment:
    dev  -> .env             (already used for local `go run .`)
    prod -> .env.production  (gitignored; you maintain this by hand)
  and sets those values for this process only, then invokes either:
    go run ./cmd/scrubber --source <path> --edition <edition>          (monsters)
    go run ./cmd/scrubber --mode items --source <path>                 (items)

.PARAMETER Mode
  "monsters" or "items". Prompted for (default "monsters") if omitted. Items
  have no -Edition — edition is derived per entry from the 5etools source
  book, not from which checkout is scrubbed (see design.md Decision 7 in the
  add-item-search-and-linking change).

.PARAMETER Environment
  "dev" or "prod". Prompted for (default "dev") if omitted.

.PARAMETER Edition
  "5e" or "5.5e". Prompted for (default "5e") if omitted. Ignored for -Mode items.

.PARAMETER SourcePath
  Path to a local 5etools repository checkout root. If omitted, defaults to
  SCRUBBER_SOURCE_5E or SCRUBBER_SOURCE_5_5E from the env file (for monsters,
  whichever matches -Edition; for items, always SCRUBBER_SOURCE_5_5E, since
  that checkout is a strict superset for item data), and prompts only if
  that's also unset.

.EXAMPLE
  ./scripts/run-scrubber.ps1
  ./scripts/run-scrubber.ps1 -Environment prod -Edition 5e -SourcePath C:\dev\5etools-src
  ./scripts/run-scrubber.ps1 -Mode items -Environment prod -SourcePath C:\dev\5etools-src
#>

param(
    [ValidateSet('monsters', 'items')]
    [string]$Mode,

    [ValidateSet('dev', 'prod')]
    [string]$Environment,

    [ValidateSet('5e', '5.5e')]
    [string]$Edition,

    [string]$SourcePath
)

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot

function Read-EnvFile([string]$path) {
    $values = @{}
    if (Test-Path $path) {
        foreach ($line in Get-Content $path) {
            if ($line -match '^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*?)\s*$') {
                $values[$Matches[1]] = $Matches[2]
            }
        }
    }
    return $values
}

function Read-WithDefault([string]$prompt, [string]$default) {
    $label = if ($default) { "$prompt [$default]" } else { $prompt }
    $response = Read-Host -Prompt $label
    if ([string]::IsNullOrWhiteSpace($response)) { return $default }
    return $response
}

Write-Host '== Scrubber setup ==' -ForegroundColor Cyan
Write-Host 'Populates MongoDB and Typesense from a local 5etools data checkout.' -ForegroundColor DarkGray
Write-Host ''

if (-not $Mode) {
    $Mode = Read-WithDefault 'Mode (monsters or items)' 'monsters'
}
if ($Mode -ne 'monsters' -and $Mode -ne 'items') {
    throw "Mode must be 'monsters' or 'items', got '$Mode'."
}

if (-not $Environment) {
    $Environment = Read-WithDefault 'Environment (dev or prod)' 'dev'
}
if ($Environment -ne 'dev' -and $Environment -ne 'prod') {
    throw "Environment must be 'dev' or 'prod', got '$Environment'."
}

if ($Mode -eq 'monsters') {
    if (-not $Edition) {
        $Edition = Read-WithDefault 'Edition (5e or 5.5e)' '5e'
    }
    if ($Edition -ne '5e' -and $Edition -ne '5.5e') {
        throw "Edition must be '5e' or '5.5e', got '$Edition'."
    }
}

$envFileName = if ($Environment -eq 'dev') { '.env' } else { '.env.production' }
$envFilePath = Join-Path $repoRoot $envFileName
if (-not (Test-Path $envFilePath)) {
    throw "$envFileName not found at $envFilePath - create it first (see .env.example for the expected keys)."
}

$creds = Read-EnvFile $envFilePath
if ([string]::IsNullOrWhiteSpace($creds['MONGODB_URI'])) {
    throw "MONGODB_URI not set in $envFileName - the scrubber cannot write data without it."
}
if ([string]::IsNullOrWhiteSpace($creds['TYPESENSE_URL']) -or [string]::IsNullOrWhiteSpace($creds['TYPESENSE_API_KEY'])) {
    Write-Host "TYPESENSE_URL/TYPESENSE_API_KEY incomplete in $envFileName - MongoDB will still be populated, but the search index will NOT be updated." -ForegroundColor Yellow
}

Write-Host "Using credentials from $envFileName ($Environment)" -ForegroundColor DarkGray

$sourceVar = if ($Mode -eq 'items') {
    'SCRUBBER_SOURCE_5_5E'
} elseif ($Edition -eq '5e') {
    'SCRUBBER_SOURCE_5E'
} else {
    'SCRUBBER_SOURCE_5_5E'
}
if (-not $SourcePath) {
    $SourcePath = $creds[$sourceVar]
    if ($SourcePath) {
        Write-Host "Using $sourceVar from $envFileName as source path: $SourcePath" -ForegroundColor DarkGray
    } else {
        $SourcePath = Read-Host -Prompt 'Path to local 5etools repository root'
    }
}
if ([string]::IsNullOrWhiteSpace($SourcePath)) {
    throw "No source path given, and $sourceVar is not set in $envFileName."
}
if (-not (Test-Path $SourcePath)) {
    throw "Source path not found: $SourcePath"
}

$env:MONGODB_URI = $creds['MONGODB_URI']
$env:TYPESENSE_URL = $creds['TYPESENSE_URL']
$env:TYPESENSE_API_KEY = $creds['TYPESENSE_API_KEY']

Push-Location $repoRoot
try {
    if ($Mode -eq 'items') {
        Write-Host ''
        Write-Host "Running scrubber: --mode items --source `"$SourcePath`"" -ForegroundColor Cyan
        Write-Host ''
        go run ./cmd/scrubber --mode items --source $SourcePath
    } else {
        Write-Host ''
        Write-Host "Running scrubber: --source `"$SourcePath`" --edition $Edition" -ForegroundColor Cyan
        Write-Host ''
        go run ./cmd/scrubber --source $SourcePath --edition $Edition
    }
} finally {
    Pop-Location
}
