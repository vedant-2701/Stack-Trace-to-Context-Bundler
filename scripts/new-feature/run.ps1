# scripts/new-feature/run.ps1
# PowerShell equivalent of run.sh

param (
    [string]$FeatureName,
    [string]$Description = "",
    [string]$DependsOn = "none"
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$RepoRoot = Resolve-Path (Join-Path $ScriptDir "..\..")
$SpecsDir = Join-Path $RepoRoot "specs"
$TemplatesDir = Join-Path $SpecsDir "_templates"
$IndexFile = Join-Path $SpecsDir "INDEX.md"

if (-not (Test-Path $TemplatesDir)) {
    Write-Error "specs/_templates directory not found."
    exit 1
}

if ([string]::IsNullOrWhiteSpace($FeatureName)) {
    Write-Host "=== New Feature ===" -ForegroundColor Cyan
    $FeatureName = Read-Host "Feature name (e.g. 'user-auth')"
    if ([string]::IsNullOrWhiteSpace($FeatureName)) {
        Write-Error "Feature name cannot be empty."
        exit 1
    }
    $Description = Read-Host "One-line description"
    $dep = Read-Host "Depends on (feature ID or none) [none]"
    if (-not [string]::IsNullOrWhiteSpace($dep)) {
        $DependsOn = $dep
    }
}

if ([string]::IsNullOrWhiteSpace($DependsOn)) {
    $DependsOn = "none"
}

$FeatureSlug = $FeatureName.ToLower() -replace '[^a-z0-9\s_-]', '' -replace '[\s_]+', '-' -replace '^-+|-+$', ''

$MaxId = 0
Get-ChildItem -Path $SpecsDir -Directory | ForEach-Object {
    if ($_.Name -match '^(\d{3})-') {
        $num = [int]$Matches[1]
        if ($num -gt $MaxId) {
            $MaxId = $num
        }
    }
}

$NextNum = $MaxId + 1
$NextId = "{0:D3}" -f $NextNum
$TargetDir = Join-Path $SpecsDir "$NextId-$FeatureSlug"

if (Test-Path $TargetDir) {
    Write-Error "Directory $TargetDir already exists."
    exit 1
}

Write-Host "Creating feature $NextId-$FeatureSlug..." -ForegroundColor Green
New-Item -ItemType Directory -Path $TargetDir | Out-Null

Get-ChildItem -Path $TemplatesDir -Filter "*.md" | ForEach-Object {
    $targetFile = Join-Path $TargetDir $_.Name
    $content = Get-Content $_.FullName -Raw
    $content = $content -replace '\{\{ID\}\}', $NextId
    $content = $content -replace '\{\{FEATURE_SLUG\}\}', $FeatureSlug
    $content = $content -replace '\{\{FEATURE_NAME\}\}', $FeatureName
    Set-Content -Path $targetFile -Value $content -NoNewline
}

if (Test-Path $IndexFile) {
    $NewRow = "| $NextId | $FeatureName | $Description | $DependsOn | idea |"
    $lines = Get-Content $IndexFile
    $lastRowIndex = -1
    for ($i = 0; $i -lt $lines.Count; $i++) {
        if ($lines[$i] -match '^\|') {
            $lastRowIndex = $i
        }
    }
    if ($lastRowIndex -ge 0) {
        $newLines = $lines[0..$lastRowIndex] + $NewRow + $lines[($lastRowIndex + 1)..($lines.Count - 1)]
        Set-Content -Path $IndexFile -Value $newLines
    } else {
        Add-Content -Path $IndexFile -Value "`n$NewRow"
    }
    Write-Host "Added entry to specs/INDEX.md:" -ForegroundColor Yellow
    Write-Host $NewRow
}

Write-Host "`nSuccessfully created feature specifications in $TargetDir" -ForegroundColor Green