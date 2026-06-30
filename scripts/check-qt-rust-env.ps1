param(
    [switch]$SkipCompileSmoke
)

$ErrorActionPreference = "Stop"

$envScript = "D:\Tools\portweave-env.ps1"
if (-not (Test-Path -LiteralPath $envScript)) {
    throw "PortWeave environment bootstrap not found: $envScript"
}

. $envScript

function Require-Command {
    param([string]$Name)

    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if (-not $cmd) {
        throw "Required command not found on PATH after loading ${envScript}: $Name"
    }
    return $cmd.Source
}

function Run-And-Print {
    param(
        [string]$Label,
        [scriptblock]$Command
    )

    $output = & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "$Label failed with exit code $LASTEXITCODE"
    }
    $firstLine = ($output | Select-Object -First 1)
    Write-Host "${Label}: $firstLine"
}

if (-not (Test-Path -LiteralPath $env:QT_ROOT)) {
    throw "QT_ROOT does not exist: $env:QT_ROOT"
}

if ($env:CMAKE_PREFIX_PATH -ne $env:QT_ROOT) {
    throw "CMAKE_PREFIX_PATH must point at QT_ROOT. QT_ROOT=$env:QT_ROOT CMAKE_PREFIX_PATH=$env:CMAKE_PREFIX_PATH"
}

$rustcPath = Require-Command "rustc"
$cargoPath = Require-Command "cargo"
$cmakePath = Require-Command "cmake"
$qmakePath = Require-Command "qmake"
$clPath = Require-Command "cl.exe"
$ninjaPath = Require-Command "ninja"

Write-Host "Resolved tools:"
Write-Host "  rustc=$rustcPath"
Write-Host "  cargo=$cargoPath"
Write-Host "  cmake=$cmakePath"
Write-Host "  qmake=$qmakePath"
Write-Host "  cl.exe=$clPath"
Write-Host "  ninja=$ninjaPath"

Run-And-Print "rustc" { rustc --version }
Run-And-Print "cargo" { cargo --version }
Run-And-Print "cmake" { cmake --version }

$qtVersion = (& qmake -query QT_VERSION).Trim()
if ($LASTEXITCODE -ne 0) {
    throw "qmake -query QT_VERSION failed with exit code $LASTEXITCODE"
}
if (-not $qtVersion.StartsWith("6.")) {
    throw "Qt 6 is required, got QT_VERSION=$qtVersion"
}
Write-Host "qmake QT_VERSION: $qtVersion"

if ($SkipCompileSmoke) {
    Write-Host "Compile smoke skipped"
    Write-Host "Qt/Rust environment preflight OK"
    exit 0
}

$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("portweave-qt-rust-preflight-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempRoot | Out-Null

try {
    $rustSource = Join-Path $tempRoot "rust_smoke.rs"
    $rustExe = Join-Path $tempRoot "rust_smoke.exe"
    Set-Content -LiteralPath $rustSource -Encoding ASCII -Value 'fn main() { println!("portweave-rust-smoke"); }'
    & rustc $rustSource -o $rustExe
    if ($LASTEXITCODE -ne 0) {
        throw "rustc compile smoke failed with exit code $LASTEXITCODE"
    }
    $rustOutput = (& $rustExe).Trim()
    if ($rustOutput -ne "portweave-rust-smoke") {
        throw "unexpected Rust smoke output: $rustOutput"
    }
    Write-Host "Rust compile smoke: $rustOutput"

    $cppSource = Join-Path $tempRoot "cl_smoke.cpp"
    $cppExe = Join-Path $tempRoot "cl_smoke.exe"
    $cppObj = Join-Path $tempRoot "cl_smoke.obj"
    Set-Content -LiteralPath $cppSource -Encoding ASCII -Value @(
        '#include <iostream>'
        'int main() { std::cout << "portweave-cl-smoke"; return 0; }'
    )
    & cl.exe /nologo /EHsc "/Fe:$cppExe" "/Fo:$cppObj" $cppSource | Out-Host
    if ($LASTEXITCODE -ne 0) {
        throw "cl.exe compile smoke failed with exit code $LASTEXITCODE"
    }
    $cppOutput = (& $cppExe).Trim()
    if ($cppOutput -ne "portweave-cl-smoke") {
        throw "unexpected cl.exe smoke output: $cppOutput"
    }
    Write-Host "MSVC compile smoke: $cppOutput"
}
finally {
    $resolvedTemp = [System.IO.Path]::GetFullPath($tempRoot)
    $resolvedBase = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
    if ($resolvedTemp.StartsWith($resolvedBase, [System.StringComparison]::OrdinalIgnoreCase)) {
        Remove-Item -LiteralPath $resolvedTemp -Recurse -Force
    }
}

Write-Host "Qt/Rust environment preflight OK"
