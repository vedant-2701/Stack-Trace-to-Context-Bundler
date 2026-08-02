# scripts/new-decision/run.ps1
# PowerShell equivalent of run.sh

param (
    [string]$TitleInput,
    [string]$DependsOn = "none"
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$RepoRoot = Resolve-Path (Join-Path $ScriptDir "..\..")
$DecisionsDir = Join-Path $RepoRoot "memory\decisions"
$TemplateFile = Join-Path $DecisionsDir "_template.md"
$IndexFile = Join-Path $DecisionsDir "INDEX.md"

if (-not (Test-Path $DecisionsDir)) {
    Write-Error "memory/decisions directory not found."
    exit 1
}

if ([string]::IsNullOrWhiteSpace($TitleInput)) {
    Write-Host "=== New Architecture Decision Record ===" -ForegroundColor Cyan
    $TitleInput = Read-Host "Decision title (e.g. 'Use PostgreSQL for persistent storage')"
    if ([string]::IsNullOrWhiteSpace($TitleInput)) {
        Write-Error "Title cannot be empty."
        exit 1
    }

    $dep = Read-Host "Depends on (decision ID or none) [none]"
    if (-not [string]::IsNullOrWhiteSpace($dep)) {
        $DependsOn = $dep
    }
}

$Slug = $TitleInput.ToLower() -replace '[^a-z0-9\s_-]', '' -replace '[\s_]+', '-' -replace '^-+|-+$', ''

$MaxId = 0
Get-ChildItem -Path $DecisionsDir -Filter "*.md" | ForEach-Object {
    if ($_.Name -match '^(\d{4})-') {
        $num = [int]$Matches[1]
        if ($num -gt $MaxId) {
            $MaxId = $num
        }
    }
}

$NextNum = $MaxId + 1
$NextId = "{0:D4}" -f $NextNum
$DateNow = (Get-Date).ToString("yyyy-MM-dd")
$TargetFile = Join-Path $DecisionsDir "$NextId-$Slug.md"

if (Test-Path $TargetFile) {
    Write-Error "Decision record $TargetFile already exists."
    exit 1
}

if (Test-Path $TemplateFile) {
    $content = Get-Content $TemplateFile -Raw
    $content = $content -replace '\{\{ID\}\}', $NextId
    $content = $content -replace '\{\{DECISION_TITLE\}\}', $TitleInput
    $content = $content -replace '\{\{DATE\}\}', $DateNow
    $content = $content -replace '\{\{DEPENDS\}\}', $DependsOn
    Set-Content -Path $TargetFile -Value $content -NoNewline
} else {
    $content = "# $NextId — $TitleInput`n`n**Status:** Proposed | Accepted | Superseded by 000X`n**Date:** $DateNow`n**Depends on:** $DependsOn`n`n## Context`n`n## Decision`n`n## Alternatives considered`n`n## Consequences`n"
    Set-Content -Path $TargetFile -Value $content -NoNewline
}

Write-Host "Created decision record: memory/decisions/$NextId-$Slug.md" -ForegroundColor Green

if (Test-Path $IndexFile) {
    $NewRow = "| $NextId | $TitleInput | $DependsOn | Proposed | $DateNow |"
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
    Write-Host "Added entry to memory/decisions/INDEX.md:" -ForegroundColor Yellow
    Write-Host $NewRow
} else {
    Write-Warning "memory/decisions/INDEX.md not found - decision created but not indexed."
}