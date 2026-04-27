param(
    [string]$Version = "v2.6.3"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) {
    throw "Required executable not found: go"
}

go install "github.com/sigstore/cosign/v2/cmd/cosign@$Version"

$goBin = (go env GOBIN)
if (-not $goBin) {
    $goPath = (go env GOPATH)
    if (-not $goPath) {
        throw "go env GOPATH is empty"
    }
    $goBin = Join-Path $goPath "bin"
}

$exeName = if ($env:OS -eq "Windows_NT") { "cosign.exe" } else { "cosign" }
$cosignPath = Join-Path $goBin $exeName
if (-not (Test-Path -LiteralPath $cosignPath -PathType Leaf)) {
    throw "Cosign installation completed, but executable was not found at $cosignPath"
}

Write-Host "Installed cosign $Version at $cosignPath"
& $cosignPath version
