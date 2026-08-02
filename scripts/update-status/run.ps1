# scripts/update-status/run.ps1
# Interactive feature status updater for PowerShell supporting console/GUI search & selection

param (
    [string]$FeatureId,
    [string]$Status
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$RepoRoot = Resolve-Path (Join-Path $ScriptDir "..\..")
$IndexFile = Join-Path $RepoRoot "specs\INDEX.md"

if (-not (Test-Path $IndexFile)) {
    Write-Error "specs/INDEX.md file not found at $IndexFile."
    exit 1
}

$lines = Get-Content $IndexFile
$featureRows = $lines | Where-Object { $_ -match '^\|\s*\d{3}\s*\|' }

# Interactive Feature Selection if FeatureId is missing
if ([string]::IsNullOrWhiteSpace($FeatureId)) {
    Write-Host "=== Feature Status Updater ===" -ForegroundColor Cyan
    
    $hasConsoleGui = Get-Command Out-ConsoleGridView -ErrorAction SilentlyContinue
    $hasGridView = Get-Command Out-GridView -ErrorAction SilentlyContinue
    
    if ($hasConsoleGui) {
        $selected = $featureRows | Out-ConsoleGridView -Title "Select Feature (Type to search, ↑/↓ to navigate)"
        if ($selected -match '^\|\s*(\d{3})\s*\|') {
            $FeatureId = $Matches[1]
        }
    } elseif ($hasGridView) {
        $selected = $featureRows | Out-GridView -Title "Select Feature (Type to search, select and click OK)" -OutputMode Single
        if ($selected -match '^\|\s*(\d{3})\s*\|') {
            $FeatureId = $Matches[1]
        }
    } else {
        Write-Host "Existing features in specs/INDEX.md:" -ForegroundColor Yellow
        $featureRows | Write-Host
        Write-Host ""
        $FeatureId = Read-Host "Enter 3-digit Feature ID (e.g. 001)"
    }
}

if ([string]::IsNullOrWhiteSpace($FeatureId)) {
    Write-Error "No feature selected."
    exit 1
}

if ($FeatureId -match '^\d+$') {
    $FeatureId = "{0:D3}" -f [int]$FeatureId
}

$matchingLine = $lines | Where-Object { $_ -match "^\|\s*$FeatureId\s*\|" }

if (-not $matchingLine) {
    Write-Error "Feature ID '$FeatureId' not found in specs/INDEX.md."
    exit 1
}

$statusOptions = @(
    "1) idea        (listed, not discussed)",
    "2) specifying  (spec.md in progress)",
    "3) spec'd      (spec.md done, plan.md not started)",
    "4) planned     (plan.md + tasks.md done)",
    "5) in-progress (implementation underway)",
    "6) done        (completed)"
)

$statusMap = @{
    "1" = "idea"
    "2" = "specifying"
    "3" = "spec'd"
    "4" = "planned"
    "5" = "in-progress"
    "6" = "done"
}

if ([string]::IsNullOrWhiteSpace($Status) -or -not ($statusMap.ContainsValue($Status))) {
    $hasConsoleGui = Get-Command Out-ConsoleGridView -ErrorAction SilentlyContinue
    $hasGridView = Get-Command Out-GridView -ErrorAction SilentlyContinue
    
    if ($hasConsoleGui) {
        $selectedStatus = $statusOptions | Out-ConsoleGridView -Title "Select new status for feature $FeatureId"
        if ($selectedStatus -match '^(\d)\)') {
            $Status = $statusMap[$Matches[1]]
        }
    } elseif ($hasGridView) {
        $selectedStatus = $statusOptions | Out-GridView -Title "Select new status for feature $FeatureId" -OutputMode Single
        if ($selectedStatus -match '^(\d)\)') {
            $Status = $statusMap[$Matches[1]]
        }
    } else {
        Write-Host "`nSelect new status for feature $FeatureId:" -ForegroundColor Cyan
        $statusOptions | ForEach-Object { Write-Host "  $_" }
        
        $choice = Read-Host "Enter choice [1-6]"
        if ($statusMap.ContainsKey($choice)) {
            $Status = $statusMap[$choice]
        } else {
            Write-Error "Invalid choice."
            exit 1
        }
    }
}

$updatedLines = foreach ($line in $lines) {
    if ($line -match "^\|\s*$FeatureId\s*\|") {
        $parts = $line.Split('|')
        if ($parts.Length -ge 6) {
            $parts[5] = " $Status "
            $line = $parts -join '|'
        }
    }
    $line
}

Set-Content -Path $IndexFile -Value $updatedLines
Write-Host "Updated feature $FeatureId status to: $Status in specs/INDEX.md" -ForegroundColor Green
