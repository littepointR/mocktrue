[CmdletBinding()]
param(
    [string]$Pattern,
    [string[]]$Path,
    [string]$OutFile
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Write-Receipt {
    param(
        [string[]]$Lines
    )

    if ([string]::IsNullOrWhiteSpace($OutFile)) {
        Write-Error "Missing required -OutFile parameter."
        exit 64
    }

    $parent = Split-Path -Parent -Path $OutFile
    if (-not [string]::IsNullOrWhiteSpace($parent)) {
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
    }

    $Lines | Set-Content -LiteralPath $OutFile -Encoding UTF8
}

function Stop-WithReceipt {
    param(
        [int]$Code,
        [string]$Status,
        [string[]]$Details
    )

    Write-Receipt -Lines @(
        "status: $Status"
        "exit_code: $Code"
        "pattern: $Pattern"
        "paths:"
    ) + @($Path | ForEach-Object { "  - $_" }) + @(
        "timestamp_utc: $((Get-Date).ToUniversalTime().ToString("o"))"
        "details:"
    ) + @($Details | ForEach-Object { "  - $_" })

    if ($Code -eq 0) {
        exit 0
    }

    if ($Code -eq 1) {
        exit 1
    }

    exit 69
}

if ([string]::IsNullOrWhiteSpace($Pattern)) {
    Stop-WithReceipt -Code 64 -Status "invalid-input" -Details @("Missing required -Pattern parameter.")
}

if ($null -eq $Path -or $Path.Count -eq 0) {
    Stop-WithReceipt -Code 64 -Status "invalid-input" -Details @("Missing required -Path parameter.")
}

$missingPaths = @($Path | Where-Object { [string]::IsNullOrWhiteSpace($_) -or -not (Test-Path -LiteralPath $_) })
if ($missingPaths.Count -gt 0) {
    Stop-WithReceipt -Code 64 -Status "invalid-input" -Details @("One or more scan paths do not exist: $($missingPaths -join ', ')")
}

$rg = Get-Command rg -ErrorAction SilentlyContinue | Select-Object -First 1
if ($null -eq $rg) {
    Stop-WithReceipt -Code 69 -Status "tool-error" -Details @("Required tool 'rg' was not found on PATH.")
}

$rgArgs = @(
    "--line-number"
    "--column"
    "--with-filename"
    "--color"
    "never"
    "-e"
    $Pattern
    "--"
) + $Path

$rgOutput = @(& $rg.Source @rgArgs 2>&1)
$rgExitCode = $LASTEXITCODE
$outputLines = @($rgOutput | ForEach-Object { $_.ToString() })

if ($rgExitCode -eq 1 -and $outputLines.Count -eq 0) {
    Stop-WithReceipt -Code 0 -Status "pass" -Details @("No forbidden matches found.", "rg_exit_code: 1")
}

if ($rgExitCode -eq 0) {
    Stop-WithReceipt -Code 1 -Status "forbidden-match" -Details @("Forbidden matches found.", "rg_exit_code: 0", "matches:") + $outputLines
}

if ($rgExitCode -eq 1) {
    Stop-WithReceipt -Code 69 -Status "tool-error" -Details @("rg returned exit code 1 with output; expected empty output for no-match pass.", "rg_exit_code: 1", "output:") + $outputLines
}

Stop-WithReceipt -Code 69 -Status "tool-error" -Details @("rg returned a tool error.", "rg_exit_code: $rgExitCode", "output:") + $outputLines
