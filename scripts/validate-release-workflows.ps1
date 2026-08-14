$ErrorActionPreference = "Stop"
$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$releaseWorkflow = Get-Content (Join-Path $repoRoot ".github/workflows/release.yml") -Raw
$ciWorkflow = Get-Content (Join-Path $repoRoot ".github/workflows/ci.yml") -Raw
$dockerfile = Get-Content (Join-Path $repoRoot "Dockerfile") -Raw
$compose = Get-Content (Join-Path $repoRoot "deploy/docker-compose.yml") -Raw

function Assert-Contains {
    param([string]$Text, [string]$Needle, [string]$Description)
    if (-not $Text.Contains($Needle)) {
        throw "Missing $Description ($Needle)"
    }
}

Assert-Contains $releaseWorkflow 'secrets.UPDATE_SIGNING_PRIVATE_KEY' "release signing secret"
Assert-Contains $releaseWorkflow '-RequireSigningKey' "fail-closed release build"
Assert-Contains $releaseWorkflow 'fronted/package-lock.json' "frontend dependency cache path"
Assert-Contains $releaseWorkflow 'branches:' "main-branch release trigger"
Assert-Contains $releaseWorkflow '- main' "main release branch"
Assert-Contains $releaseWorkflow 'paths:' "VERSION-only release trigger"
Assert-Contains $releaseWorkflow '- "VERSION"' "root VERSION release path"
Assert-Contains $releaseWorkflow 'workflow_dispatch:' "manual release retry trigger"
Assert-Contains $releaseWorkflow 'Create or verify release tag' "automatic release tag"
Assert-Contains $releaseWorkflow 'git push origin "$tag"' "release tag push"
Assert-Contains $releaseWorkflow 'tag_name: v${{ needs.release-assets.outputs.version }}' "GitHub Release tag"
Assert-Contains $releaseWorkflow '< VERSION' "root VERSION release metadata"
Assert-Contains $releaseWorkflow 'platforms: linux/amd64,linux/arm64' "multi-architecture image build"
Assert-Contains $releaseWorkflow 'Derive container tags from release version' "release-version container tag derivation"
Assert-Contains $releaseWorkflow 'type=raw,value=${{ needs.release-assets.outputs.version }}' "full release-version container tag"
Assert-Contains $releaseWorkflow 'type=raw,value=${{ steps.release-tags.outputs.major-minor }}' "minor release container tag"
Assert-Contains $releaseWorkflow "steps.release-tags.outputs.stable == 'true'" "stable latest container tag condition"
Assert-Contains $releaseWorkflow 'update-manifest.json.sig' "detached signature upload"
Assert-Contains $releaseWorkflow 'needs: [release-assets, container]' "release publication gate"
Assert-Contains $releaseWorkflow 'dist/release/submerge-linux-amd64' "amd64 release asset"
Assert-Contains $releaseWorkflow 'dist/release/submerge-linux-arm64' "arm64 release asset"
Assert-Contains $releaseWorkflow 'dist/release/docker-compose.yml' "pinned Docker Compose release asset"
Assert-Contains $releaseWorkflow 'Smoke-test published image' "published image smoke test"
Assert-Contains $releaseWorkflow 'published container did not become healthy' "release health gate"

Assert-Contains $ciWorkflow 'GOARCH=amd64' "amd64 cross-build check"
Assert-Contains $ciWorkflow 'GOARCH=arm64' "arm64 cross-build check"
Assert-Contains $ciWorkflow 'working-directory: fronted' "React frontend working directory"
Assert-Contains $ciWorkflow 'fronted/package-lock.json' "frontend dependency cache path"
Assert-Contains $ciWorkflow 'backend/version.Value=${version}' "linked binary version"
Assert-Contains $ciWorkflow 'APP_VERSION=${version}' "CI root version export"
Assert-Contains $ciWorkflow 'VERSION=${{ env.APP_VERSION }}' "CI container version build argument"
Assert-Contains $ciWorkflow '"\"version\":\"${version}\""' "CI embedded version smoke test"
Assert-Contains $ciWorkflow 'go run ./cmd/update-key-check' "unsigned build fail-closed check"
Assert-Contains $ciWorkflow 'sh docker/entrypoint_test.sh' "entrypoint behavior check"
Assert-Contains $ciWorkflow 'docker/build-push-action' "container build check"
Assert-Contains $ciWorkflow 'load: true' "locally loaded CI image"
Assert-Contains $ciWorkflow 'docker compose -f deploy/docker-compose.yml config --quiet' "Compose validation"
Assert-Contains $ciWorkflow '/api/health' "container health smoke test"

Assert-Contains $dockerfile 'COPY VERSION /src/VERSION' "root VERSION frontend build input"
Assert-Contains $dockerfile 'WORKDIR /src/fronted' "React frontend Docker workdir"
Assert-Contains $dockerfile 'COPY fronted/package.json fronted/package-lock.json ./' "React frontend Docker dependencies"
Assert-Contains $dockerfile 'test "$VERSION" = "$version"' "container metadata version matches root VERSION"
Assert-Contains $dockerfile 'backend/version.Value=${version}' "linked container binary version"
Assert-Contains $dockerfile 'COPY --from=frontend-builder /src/backend/internal/webui/dist ./internal/webui/dist' "embedded React build transfer"
Assert-Contains $dockerfile 'UPDATE_PUBLIC_KEY_BASE64' "updater public-key injection"
Assert-Contains $compose './data:/app/data' "persistent data bind mount"
Assert-Contains $compose './data/log:/app/log' "log bind mount under data"
Assert-Contains $compose './data/geo:/app/defaults/geo' "Geo bind mount under data"
Assert-Contains $compose './bin:/app/runtime' "persistent update binary bind mount"
Assert-Contains $compose 'restart: unless-stopped' "restart policy"

Assert-Contains (Get-Content (Join-Path $repoRoot "fronted/vite.config.ts") -Raw) "VERSION must contain a SemVer value" "frontend VERSION validation"
Assert-Contains (Get-Content (Join-Path $repoRoot "scripts/build-release.ps1") -Raw) "VERSION must contain a SemVer value" "release VERSION validation"
Assert-Contains (Get-Content (Join-Path $repoRoot "scripts/build-release.ps1") -Raw) 'Join-Path $repoRoot "fronted"' "release React frontend path"
Assert-Contains (Get-Content (Join-Path $repoRoot "scripts/build-release.ps1") -Raw) 'docker-compose.yml' "generated Docker Compose release asset"
Assert-Contains (Get-Content (Join-Path $repoRoot "scripts/build-release.ps1") -Raw) 'ToLowerInvariant' "lower-case GHCR release image"

Write-Output "release workflow contract passed"
