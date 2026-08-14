param(
    [string]$OutputDir = "dist/release",
    [string]$Repository = $(if ($env:GITHUB_REPOSITORY) { $env:GITHUB_REPOSITORY } else { "syueya/subMerge" }),
    [switch]$SkipChecks,
    [switch]$RequireSigningKey
)

$ErrorActionPreference = "Stop"
$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$outputPath = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $OutputDir))
if ($outputPath -eq $repoRoot -or -not $outputPath.StartsWith($repoRoot + [System.IO.Path]::DirectorySeparatorChar)) {
    throw "OutputDir must resolve to a child of the repository: $outputPath"
}

$version = (Get-Content (Join-Path $repoRoot "VERSION") -Raw).Trim()
if ([string]::IsNullOrWhiteSpace($version)) {
    throw "VERSION must not be empty"
}
$semVer = '^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(?:0|[1-9]\d*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9]\d*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$'
if ($version -notmatch $semVer) {
    throw "VERSION must contain a SemVer value, got $version"
}
if ($env:GITHUB_REF_TYPE -eq "tag" -and $env:GITHUB_REF_NAME -ne "v$version") {
    throw "tag $($env:GITHUB_REF_NAME) does not match VERSION v$version"
}
if ($RequireSigningKey -and [string]::IsNullOrWhiteSpace($env:UPDATE_SIGNING_PRIVATE_KEY)) {
    throw "UPDATE_SIGNING_PRIVATE_KEY is required for a signed release"
}

function Invoke-Checked {
    param([scriptblock]$Command, [string]$Description)
    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "$Description failed with exit code $LASTEXITCODE"
    }
}

Push-Location (Join-Path $repoRoot "fronted")
try {
    $npmCache = Join-Path ([System.IO.Path]::GetTempPath()) "submerge-npm-cache"
    Invoke-Checked { npm ci --cache $npmCache } "npm ci"
    if (-not $SkipChecks) {
        Invoke-Checked { npm run lint } "frontend lint"
        Invoke-Checked { npm test } "frontend tests"
    }
    Invoke-Checked { npm run build } "frontend build"
}
finally {
    Pop-Location
}

$backendDir = Join-Path $repoRoot "backend"
Push-Location $backendDir
try {
    if (-not $SkipChecks) {
        Invoke-Checked { go test ./... } "Go tests"
    }

    $toolSuffix = if ($IsWindows) { ".exe" } else { "" }
    $releaseTool = Join-Path ([System.IO.Path]::GetTempPath()) "submerge-release-tool$toolSuffix"
    Invoke-Checked { go build -trimpath -buildvcs=false -o $releaseTool ./cmd/release-tool } "release tool build"

    $publicKey = ""
    if (-not [string]::IsNullOrWhiteSpace($env:UPDATE_SIGNING_PRIVATE_KEY)) {
        $publicKey = (& $releaseTool public-key).Trim()
        if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($publicKey)) {
            throw "public key derivation failed"
        }

        $keyProbe = Join-Path ([System.IO.Path]::GetTempPath()) "submerge-update-key-check$toolSuffix"
        $probeLDFlags = "-X github.com/submerge/submerge/backend/internal/updater.PublicKeyBase64=$publicKey"
        Invoke-Checked { go build -trimpath -buildvcs=false -ldflags $probeLDFlags -o $keyProbe ./cmd/update-key-check } "update key probe build"
        $linkedPublicKey = (& $keyProbe).Trim()
        if ($LASTEXITCODE -ne 0 -or $linkedPublicKey -ne $publicKey) {
            throw "linked update public key does not match the signing key"
        }
    }

    if (Test-Path -LiteralPath $outputPath) {
        Remove-Item -LiteralPath $outputPath -Recurse -Force
    }
    New-Item -ItemType Directory -Path $outputPath | Out-Null

    $composeTemplate = Get-Content -LiteralPath (Join-Path $repoRoot "deploy/docker-compose.yml") -Raw
    $defaultImage = 'image: ${SUBMERGE_IMAGE:-ghcr.io/syueya/submerge}:${SUBMERGE_TAG:-latest}'
    if (-not $composeTemplate.Contains($defaultImage)) {
        throw "deploy/docker-compose.yml does not contain the expected configurable SubMerge image"
    }
    $releaseImage = "image: ghcr.io/$($Repository.ToLowerInvariant()):$version"
    $releaseCompose = $composeTemplate.Replace($defaultImage, $releaseImage)
    Set-Content -LiteralPath (Join-Path $outputPath "docker-compose.yml") -Value $releaseCompose -NoNewline -Encoding utf8

    $oldGoOS = $env:GOOS
    $oldGoArch = $env:GOARCH
    $oldCGO = $env:CGO_ENABLED
    try {
        $env:GOOS = "linux"
        $env:CGO_ENABLED = "0"
        foreach ($arch in @("amd64", "arm64")) {
            $env:GOARCH = $arch
            $asset = Join-Path $outputPath "submerge-linux-$arch"
            $ldflags = "-s -w -buildid= -X github.com/submerge/submerge/backend/version.Value=$version"
            if ($publicKey) {
                $ldflags += " -X github.com/submerge/submerge/backend/internal/updater.PublicKeyBase64=$publicKey"
            }
            Invoke-Checked { go build -trimpath -buildvcs=false -ldflags $ldflags -o $asset . } "Go linux/$arch build"
        }
    }
    finally {
        $env:GOOS = $oldGoOS
        $env:GOARCH = $oldGoArch
        $env:CGO_ENABLED = $oldCGO
    }

    if ($publicKey) {
        Invoke-Checked { & $releaseTool manifest --assets $outputPath --repository $Repository --version $version } "signed manifest generation"
        Set-Content -LiteralPath (Join-Path $outputPath "update-public-key.txt") -Value $publicKey -Encoding ascii
    }
    elseif (-not $RequireSigningKey) {
        Write-Warning "Unsigned development build: online update installation is disabled."
    }
}
finally {
    Pop-Location
}

Write-Host "Built SubMerge v$version release assets in $outputPath"
