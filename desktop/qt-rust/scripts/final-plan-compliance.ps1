[CmdletBinding()]
param(
    [string]$Plan,
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

function New-Check {
    param(
        [string]$Name,
        [string[]]$RequiredFiles,
        [string]$BlockerPattern
    )

    [pscustomobject]@{
        Name = $Name
        RequiredFiles = $RequiredFiles
        BlockerPattern = $BlockerPattern
    }
}

if ([string]::IsNullOrWhiteSpace($Plan)) {
    Write-Report -Lines @("status: invalid-input", "details:", "  - Missing required -Plan parameter.")
    exit 64
}

if ([string]::IsNullOrWhiteSpace($EvidenceRoot)) {
    Write-Report -Lines @("status: invalid-input", "details:", "  - Missing required -EvidenceRoot parameter.")
    exit 64
}

if (-not (Test-Path -LiteralPath $Plan -PathType Leaf)) {
    Write-Report -Lines @("status: invalid-input", "details:", "  - Plan file not found: $Plan")
    exit 64
}

$planText = Get-Content -LiteralPath $Plan -Raw
$requiredPlanMarkers = @(
    "T10. Add shared and final verification helper scripts"
    "F1. Plan compliance audit"
    "F4. Scope fidelity"
)

$missingPlanMarkers = @($requiredPlanMarkers | Where-Object { $planText -notlike "*$_*" })
if ($missingPlanMarkers.Count -gt 0) {
    Write-Report -Lines @(
        "status: failed"
        "details:"
    ) + @($missingPlanMarkers | ForEach-Object { "  - Missing plan marker: $_" })
    exit 1
}

$checks = @(
    New-Check -Name "T1 baseline/toolchain evidence" -RequiredFiles @("t1-toolchain.json") -BlockerPattern "t1*blocker*"
    New-Check -Name "T2 preflight positive and negative evidence" -RequiredFiles @("t2-preflight-positive.txt", "t2-preflight-negative.txt") -BlockerPattern "t2*blocker*"
    New-Check -Name "T3 migration matrix positive and negative evidence" -RequiredFiles @("t3-matrix-positive.txt", "t3-matrix-negative.txt") -BlockerPattern "t3*blocker*"
    New-Check -Name "T4 Rust test and boundary evidence" -RequiredFiles @("t4-cargo-test.txt", "t4-rust-boundary.txt") -BlockerPattern "t4*blocker*"
    New-Check -Name "T5 CTest positive and negative evidence" -RequiredFiles @("t5-ctest.txt", "t5-ctest-negative.txt") -BlockerPattern "t5*blocker*"
    New-Check -Name "T6 smoke-exit and QML boundary evidence" -RequiredFiles @("t6-smoke-exit.txt", "t6-qml-boundary.txt") -BlockerPattern "t6*blocker*"
    New-Check -Name "T7 GUI smoke positive and negative evidence" -RequiredFiles @("t7-gui-smoke.md", "t7-gui-smoke-negative.txt") -BlockerPattern "t7*blocker*"
    New-Check -Name "T8 CI local-equivalent and path scan evidence" -RequiredFiles @("t8-ci-local-equivalent.txt", "t8-ci-path-scan.txt") -BlockerPattern "t8*blocker*"
    New-Check -Name "T9 stale-doc and path-leak evidence" -RequiredFiles @("t9-stale-doc-scan.txt", "t9-path-leak-scan.txt") -BlockerPattern "t9*blocker*"
    New-Check -Name "T10 helper smoke evidence" -RequiredFiles @("t10-no-rg-match-positive.txt", "t10-final-plan-compliance-smoke.txt", "t10-final-scope-fidelity-smoke.txt", "t10-final-helper-positive.txt", "t10-no-rg-match-negative.txt") -BlockerPattern "t10*blocker*"
    New-Check -Name "T11 final status and staging guard evidence" -RequiredFiles @("t11-final-status.txt", "t11-staging-guard.txt") -BlockerPattern "t11*blocker*"
)

$results = New-Object System.Collections.Generic.List[string]
$missing = New-Object System.Collections.Generic.List[string]
$blockers = New-Object System.Collections.Generic.List[string]

foreach ($check in $checks) {
    $blockerFiles = @()
    if (Test-Path -LiteralPath $EvidenceRoot -PathType Container) {
        $blockerFiles = @(Get-ChildItem -LiteralPath $EvidenceRoot -File -Filter $check.BlockerPattern -ErrorAction SilentlyContinue | ForEach-Object { $_.Name })
    }

    $missingFiles = @($check.RequiredFiles | Where-Object { -not (Test-Path -LiteralPath (Join-Path $EvidenceRoot $_) -PathType Leaf) })
    if ($missingFiles.Count -eq 0) {
        $results.Add("PASS: $($check.Name)")
        continue
    }

    if ($blockerFiles.Count -gt 0) {
        $blockers.Add("$($check.Name): explicit blocker file(s): $($blockerFiles -join ', ')")
        continue
    }

    $missing.Add("$($check.Name): missing $($missingFiles -join ', ')")
}

$status = "pass"
$exitCode = 0
if ($missing.Count -gt 0) {
    if ($AllowBlockers) {
        $status = "blocked-smoke"
        $results.Add("BLOCKER: Missing downstream evidence recorded because -AllowBlockers was supplied for T10 smoke mode.")
    } else {
        $status = "failed"
        $exitCode = 1
    }
} elseif ($blockers.Count -gt 0) {
    $status = "blocked-explicit"
}

Write-Report -Lines @(
    "status: $status"
    "allow_blockers: $($AllowBlockers.IsPresent)"
    "timestamp_utc: $((Get-Date).ToUniversalTime().ToString("o"))"
    "plan: $Plan"
    "evidence_root: $EvidenceRoot"
    "checks:"
) + @($results | ForEach-Object { "  - $_" }) + @(
    "explicit_blockers:"
) + @($blockers | ForEach-Object { "  - $_" }) + @(
    "missing_evidence:"
) + @($missing | ForEach-Object { "  - $_" })

exit $exitCode
