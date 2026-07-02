[CmdletBinding()]
param(
    [string]$EvidenceRoot,
    [string]$OutFile,
    [switch]$AllowBlockers
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Write-Report {
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

function Invoke-Rg {
    param(
        [string]$Pattern,
        [string[]]$Paths
    )

    $existingPaths = @($Paths | Where-Object { Test-Path -LiteralPath $_ })
    $missingPaths = @($Paths | Where-Object { -not (Test-Path -LiteralPath $_) })
    if ($existingPaths.Count -eq 0) {
        return [pscustomobject]@{
            ExitCode = 64
            Output = @("No scan paths exist: $($Paths -join ', ')")
            MissingPaths = $missingPaths
        }
    }

    $rg = Get-Command rg -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -eq $rg) {
        return [pscustomobject]@{
            ExitCode = 69
            Output = @("Required tool 'rg' was not found on PATH.")
            MissingPaths = $missingPaths
        }
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
    ) + $existingPaths

    $rgOutput = @(& $rg.Source @rgArgs 2>&1)
    [pscustomobject]@{
        ExitCode = $LASTEXITCODE
        Output = @($rgOutput | ForEach-Object { $_.ToString() })
        MissingPaths = $missingPaths
    }
}

function Test-NoMatchScan {
    param(
        [string]$Name,
        [string]$Pattern,
        [string[]]$Paths
    )

    $scan = Invoke-Rg -Pattern $Pattern -Paths $Paths
    $details = New-Object System.Collections.Generic.List[string]
    foreach ($missingPath in $scan.MissingPaths) {
        $details.Add("missing_path: $missingPath")
    }

    if ($scan.ExitCode -eq 1 -and $scan.Output.Count -eq 0) {
        $details.Add("PASS: no forbidden matches found.")
        return [pscustomobject]@{ Name = $Name; Status = "pass"; Details = @($details) }
    }

    if ($scan.ExitCode -eq 0) {
        $details.Add("FAIL: forbidden matches found.")
        foreach ($line in $scan.Output) {
            $details.Add($line)
        }
        return [pscustomobject]@{ Name = $Name; Status = "failed"; Details = @($details) }
    }

    $details.Add("TOOL_ERROR: rg exit code $($scan.ExitCode).")
    foreach ($line in $scan.Output) {
        $details.Add($line)
    }
    return [pscustomobject]@{ Name = $Name; Status = "tool-error"; Details = @($details) }
}

function Get-ContractedFeatureIds {
    $matrixPath = "docs/development/migration-feature-matrix.csv"
    if (-not (Test-Path -LiteralPath $matrixPath -PathType Leaf)) {
        return @()
    }

    $contracted = New-Object System.Collections.Generic.HashSet[string]
    foreach ($row in (Import-Csv -LiteralPath $matrixPath)) {
        $values = @($row.PSObject.Properties | ForEach-Object { [string]$_.Value })
        $hasContractedStatus = ($values | Where-Object { $_ -match "\bcontracted\b" }).Count -gt 0
        if (-not $hasContractedStatus) {
            continue
        }

        foreach ($value in $values) {
            if ($value -match "^[A-Za-z][A-Za-z0-9_.-]+$") {
                [void]$contracted.Add($value)
            }
        }
    }

    @($contracted)
}

function Test-ProductScope {
    $name = "product-scope"
    $qmlRoot = "desktop/qt-rust/qml"
    if (-not (Test-Path -LiteralPath $qmlRoot -PathType Container)) {
        return [pscustomobject]@{ Name = $name; Status = "tool-error"; Details = @("No QML root found: $qmlRoot") }
    }

    $scan = Invoke-Rg -Pattern "featureId\s*:\s*`"[^`"]+`"" -Paths @($qmlRoot)
    $details = New-Object System.Collections.Generic.List[string]

    if ($scan.ExitCode -eq 1 -and $scan.Output.Count -eq 0) {
        $details.Add("PASS: no featureId declarations found.")
        return [pscustomobject]@{ Name = $name; Status = "pass"; Details = @($details) }
    }

    if ($scan.ExitCode -gt 1) {
        $details.Add("TOOL_ERROR: rg exit code $($scan.ExitCode).")
        foreach ($line in $scan.Output) {
            $details.Add($line)
        }
        return [pscustomobject]@{ Name = $name; Status = "tool-error"; Details = @($details) }
    }

    $contracted = Get-ContractedFeatureIds
    $violations = New-Object System.Collections.Generic.List[string]
    foreach ($line in $scan.Output) {
        if ($line -notmatch 'featureId\s*:\s*"([^"]+)"') {
            $violations.Add("Unable to parse featureId match: $line")
            continue
        }

        $featureId = $Matches[1]
        $allowedSkeleton = $featureId -like "shell.smoke.*" -or $featureId -eq "serial" -or $featureId -eq "settings"
        $allowedContracted = $contracted -contains $featureId
        if ($allowedSkeleton -or $allowedContracted) {
            $details.Add("allowed: $featureId")
            continue
        }

        $violations.Add($line)
    }

    if ($violations.Count -eq 0) {
        $details.Add("PASS: featureId declarations stay within skeleton/navigation scope or contracted rows.")
        return [pscustomobject]@{ Name = $name; Status = "pass"; Details = @($details) }
    }

    return [pscustomobject]@{ Name = $name; Status = "failed"; Details = @("FAIL: uncontracted product featureId declarations found.") + @($violations) }
}

if ([string]::IsNullOrWhiteSpace($EvidenceRoot)) {
    Write-Report -Lines @("status: invalid-input", "details:", "  - Missing required -EvidenceRoot parameter.")
    exit 64
}

$scans = @(
    (Test-ProductScope)
    (Test-NoMatchScan -Name "qml-boundary" -Pattern "portweave_rust_|extern|Library|ffi|Process" -Paths @("desktop/qt-rust/qml"))
    (Test-NoMatchScan -Name "rust-boundary" -Pattern "Qt|QML|QObject|qmetaobject|cxx_qt" -Paths @("desktop/qt-rust/rust/src", "desktop/qt-rust/rust/Cargo.toml"))
    (Test-NoMatchScan -Name "path-leak" -Pattern "[A-Za-z]:(/|\\)|PORTWEAVE_QT_RUST_ENV_SCRIPT=.*[A-Za-z]:" -Paths @("docs/development", "scripts", ".github/workflows/ci.yml"))
)

$hasFailures = @($scans | Where-Object { $_.Status -ne "pass" }).Count -gt 0
$status = "pass"
$exitCode = 0
if ($hasFailures) {
    if ($AllowBlockers) {
        $status = "blocked-smoke"
    } else {
        $status = "failed"
        $exitCode = 1
    }
}

$lines = New-Object System.Collections.Generic.List[string]
$lines.Add("status: $status")
$lines.Add("allow_blockers: $($AllowBlockers.IsPresent)")
$lines.Add("timestamp_utc: $((Get-Date).ToUniversalTime().ToString("o"))")
$lines.Add("evidence_root: $EvidenceRoot")
$lines.Add("scans:")
foreach ($scan in $scans) {
    $scanStatus = $scan.Status
    if ($AllowBlockers -and $scan.Status -ne "pass") {
        $scanStatus = "blocker"
    }

    $lines.Add("  - $($scan.Name): $scanStatus")
    foreach ($detail in $scan.Details) {
        $lines.Add("    $detail")
    }
}

Write-Report -Lines @($lines)
exit $exitCode
