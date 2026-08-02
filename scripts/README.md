# Automation Scripts Guide

This directory contains CLI automation tools for Spec-Driven Development (SDD). Each tool is organized in a dedicated task folder and includes both Bash (`run.sh`) and PowerShell (`run.ps1`) implementations.

---

## Directory Overview

```
scripts/
├── new-feature/       ← Creates a new feature folder & templates; registers in specs/INDEX.md
│   ├── run.sh
│   └── run.ps1
├── update-status/     ← Updates feature status in specs/INDEX.md interactively
│   ├── run.sh
│   └── run.ps1
└── new-decision/      ← Creates a new Architecture Decision Record in memory/decisions/
    ├── run.sh
    └── run.ps1
```

---

## 1. Create a New Feature (`scripts/new-feature/`)

Generates a new `specs/NNN-feature-name/` directory containing all 4 template markdown files (`spec.md`, `plan.md`, `progress.md`, `tasks.md`), auto-increments the 3-digit feature ID, and appends a row to `specs/INDEX.md`.

### Usage

**Bash / Git Bash:**
```bash
# Interactive mode (prompts for inputs):
./scripts/new-feature/run.sh

# With CLI arguments:
./scripts/new-feature/run.sh "User Authentication" "OAuth2 and JWT session handling" "none"
```

**PowerShell (Windows):**
```powershell
# Interactive mode:
.\scripts\new-feature\run.ps1

# With parameters:
.\scripts\new-feature\run.ps1 -FeatureInput "User Authentication" -Description "OAuth2 handling" -DependsOn "none"
```

---

## 2. Update Feature Status (`scripts/update-status/`)

Updates the Status column of a feature row in `specs/INDEX.md` (`idea` → `specifying` → `spec'd` → `planned` → `in-progress` → `done`).

Features interactive fuzzy search and arrow-key selection if optional tools are installed (`fzf` for Bash, `Out-ConsoleGridView` for PowerShell).

### Usage

**Bash / Git Bash:**
```bash
# Interactive mode (fuzzy search with arrow keys if fzf is installed):
./scripts/update-status/run.sh

# Direct CLI args:
./scripts/update-status/run.sh 001 in-progress
```

**PowerShell (Windows):**
```powershell
# Interactive mode (GUI/Console search grid if available):
.\scripts\update-status\run.ps1

# Direct parameters:
.\scripts\update-status\run.ps1 -FeatureId "001" -Status "in-progress"
```

---

## 3. Create a Decision Record (`scripts/new-decision/`)

Generates a numbered Architecture Decision Record (ADR) under `memory/decisions/` (`NNNN-title.md`) using `memory/decisions/_template.md`.

### Usage

**Bash / Git Bash:**
```bash
# Interactive mode:
./scripts/new-decision/run.sh

# With CLI arguments:
./scripts/new-decision/run.sh "Switch to PostgreSQL" "0001"
```

**PowerShell (Windows):**
```powershell
# Interactive mode:
.\scripts\new-decision\run.ps1

# With parameters:
.\scripts\new-decision\run.ps1 -TitleInput "Switch to PostgreSQL" -DependsOn "0001"
```

---

## Interactive UI & Optional Enhancements

- **Fuzzy Searching (`fzf`)**:
  Install `fzf` in your terminal for instant fuzzy search & arrow-key navigation when selecting features or statuses in Bash (`run.sh`).
- **Console Grid View (`Microsoft.PowerShell.ConsoleGuiTools`)**:
  Install via `Install-Module Microsoft.PowerShell.ConsoleGuiTools` in PowerShell for terminal UI selection in PowerShell (`run.ps1`).
