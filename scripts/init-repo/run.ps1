# scripts/init-repo/run.ps1
# PowerShell equivalent of run.sh
# One-time bootstrap: run right after cloning/downloading this template,
# before starting the kickoff chat.

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$RepoRoot = Resolve-Path (Join-Path $ScriptDir "..\..")

$Readme = Join-Path $RepoRoot "README.md"
$Sample = Join-Path $RepoRoot "SAMPLE-README.md"
$DocsDir = Join-Path $RepoRoot "docs"
$Guide = Join-Path $DocsDir "SDD-GUIDE.md"

if (Test-Path $Guide) {
    Write-Error "docs/SDD-GUIDE.md already exists. This repo looks already initialized. Nothing changed."
    exit 1
}

if (-not (Test-Path $Readme)) {
    Write-Error "README.md not found at $Readme."
    exit 1
}

if (-not (Test-Path $Sample)) {
    Write-Error "SAMPLE-README.md not found at $Sample."
    exit 1
}

New-Item -ItemType Directory -Force -Path $DocsDir | Out-Null

Write-Host "Moving current README.md -> docs/SDD-GUIDE.md ..." -ForegroundColor Cyan
Move-Item $Readme $Guide

Write-Host "Promoting SAMPLE-README.md -> README.md ..." -ForegroundColor Cyan
Move-Item $Sample $Readme

Write-Host ""
Write-Host "Done."
Write-Host "  - The SDD process guide now lives at docs/SDD-GUIDE.md"
Write-Host "  - README.md is now the (currently placeholder) product README,"
Write-Host "    to be filled in during the kickoff chat"
Write-Host ""
Write-Host "This script has done its job — you won't need it again for this repo."