param(
    [string]$RepoUrl = $env:PAGES_REPO_URL,
    [string]$Branch = $env:PAGES_BRANCH,
    [string]$WorkDir = ".pages",
    [string]$DataDir = "data",
    [string]$CommitMessage = "Update data"
)

if ([string]::IsNullOrWhiteSpace($RepoUrl)) {
    Write-Error "Set PAGES_REPO_URL or pass -RepoUrl."
    exit 1
}
if ([string]::IsNullOrWhiteSpace($Branch)) {
    $Branch = "gh-pages"
}
if (-not (Test-Path -Path $DataDir)) {
    Write-Error "Data directory not found: $DataDir"
    exit 1
}

if (-not (Test-Path -Path $WorkDir)) {
    git clone $RepoUrl $WorkDir | Out-Null
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

Push-Location $WorkDir
try {
    git fetch origin | Out-Null

    $branchExists = git ls-remote --heads origin $Branch
    if ([string]::IsNullOrWhiteSpace($branchExists)) {
        git checkout --orphan $Branch | Out-Null
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        Get-ChildItem -Force | Where-Object { $_.Name -ne ".git" } | Remove-Item -Recurse -Force
    } else {
        git checkout $Branch | Out-Null
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        git reset --hard origin/$Branch | Out-Null
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }

    if (Test-Path -Path "data") {
        Remove-Item -Recurse -Force "data"
    }
    New-Item -ItemType Directory -Force -Path "data" | Out-Null
    Copy-Item -Recurse -Force (Join-Path $PSScriptRoot ".." $DataDir "*") "data"

    git add -A | Out-Null
    $status = git status --porcelain
    if ([string]::IsNullOrWhiteSpace($status)) {
        Write-Host "No changes to deploy."
        exit 0
    }

    git commit -m $CommitMessage | Out-Null
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    git push origin $Branch | Out-Null
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Write-Host "Deployed to $Branch."
}
finally {
    Pop-Location
}
