param(
    [Parameter(Mandatory = $true)]
    [string]$Tag,

    [string]$Repository = "ersinkoc/Mizan",

    [string]$OutputDirectory = "",

    [switch]$SkipDownload
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Resolve-Executable {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [string]$Fallback = ""
    )

    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }
    if ($Fallback -and (Test-Path -LiteralPath $Fallback)) {
        return $Fallback
    }
    throw "Required executable not found: $Name"
}

if (-not $OutputDirectory) {
    $OutputDirectory = Join-Path (Join-Path $PSScriptRoot "..\dist\release-verify") $Tag
}

$gh = Resolve-Executable -Name "gh" -Fallback "C:\Program Files\GitHub CLI\gh.exe"
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null

if (-not $SkipDownload) {
    & $gh release download $Tag --repo $Repository --dir $OutputDirectory --clobber
}

$platforms = @(
    "darwin-amd64",
    "darwin-arm64",
    "linux-amd64",
    "linux-arm64",
    "windows-amd64"
)

$verified = @()

foreach ($platform in $platforms) {
    $baseName = "mizan-$platform"
    $binaryPath = Join-Path $OutputDirectory $baseName
    $checksumPath = "$binaryPath.sha256"
    $signaturePath = "$binaryPath.sig"
    $certificatePath = "$binaryPath.pem"

    foreach ($requiredPath in @($binaryPath, $checksumPath, $signaturePath, $certificatePath)) {
        if (-not (Test-Path -LiteralPath $requiredPath -PathType Leaf)) {
            throw "Missing release asset: $requiredPath"
        }
    }

    $expectedLine = Get-Content -LiteralPath $checksumPath -TotalCount 1
    $expectedHash = (($expectedLine -split "\s+")[0]).ToLowerInvariant()
    $actualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $binaryPath).Hash.ToLowerInvariant()

    if ($actualHash -ne $expectedHash) {
        throw "SHA-256 mismatch for $baseName. expected=$expectedHash actual=$actualHash"
    }

    $verified += [pscustomobject]@{
        Asset  = $baseName
        SHA256 = $actualHash
    }
}

$assetCount = (Get-ChildItem -LiteralPath $OutputDirectory -File).Count
if ($assetCount -ne 20) {
    throw "Expected 20 release assets, found $assetCount in $OutputDirectory"
}

$verified | Format-Table -AutoSize
Write-Host "Release verification passed for ${Repository}@${Tag}: $($verified.Count) binaries, $assetCount assets."
