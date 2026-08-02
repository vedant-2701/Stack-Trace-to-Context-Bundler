#!/usr/bin/env bash

# scripts/update-status/run.sh
# Updates the status of a feature in specs/INDEX.md using fuzzy search (fzf), interactive menu, or CLI parameters.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
INDEX_FILE="${REPO_ROOT}/specs/INDEX.md"

if [ ! -f "${INDEX_FILE}" ]; then
    echo "Error: specs/INDEX.md file not found at ${INDEX_FILE}."
    exit 1
fi

FEATURE_ID="$1"
STATUS_INPUT="$2"

# Interactive feature selection if FEATURE_ID is not provided
if [ -z "${FEATURE_ID}" ]; then
    echo "=== Feature Status Updater ==="
    
    # Use fzf for interactive searching and arrow-key selection if available
    if command -v fzf &>/dev/null; then
        SELECTED_ROW=$(grep -E '^\|[[:space:]]*[0-9]{3}[[:space:]]*\|' "${INDEX_FILE}" | fzf --header="Select Feature (Type to search, ↑/↓ arrow keys to navigate)" --height=40% --layout=reverse)
        if [ -n "${SELECTED_ROW}" ]; then
            FEATURE_ID=$(echo "${SELECTED_ROW}" | awk -F'|' '{print $2}' | xargs)
        fi
    else
        echo "Existing features in specs/INDEX.md:"
        grep -E '^\|[[:space:]]*[0-9]{3}[[:space:]]*\|' "${INDEX_FILE}" || true
        echo ""
        read -rp "Enter 3-digit Feature ID (e.g. 001): " FEATURE_ID
    fi
fi

if [ -z "${FEATURE_ID}" ]; then
    echo "Error: No feature selected."
    exit 1
fi

if [[ "$FEATURE_ID" =~ ^[0-9]+$ ]]; then
    FEATURE_ID=$(printf "%03d" "$((10#$FEATURE_ID))")
fi

if ! grep -q -E "^\|[[:space:]]*${FEATURE_ID}[[:space:]]*\|" "${INDEX_FILE}"; then
    echo "Error: Feature ID '${FEATURE_ID}' not found in specs/INDEX.md."
    exit 1
fi

# Interactive status selection if STATUS_INPUT is not provided
if [ -z "${STATUS_INPUT}" ]; then
    if command -v fzf &>/dev/null; then
        STATUS_CHOICES="1) idea        (listed, not discussed)\n2) specifying  (spec.md in progress)\n3) spec'd      (spec.md done, plan.md not started)\n4) planned     (plan.md + tasks.md done)\n5) in-progress (implementation underway)\n6) done        (completed)"
        SELECTED_STATUS=$(echo -e "${STATUS_CHOICES}" | fzf --header="Select new status for feature ${FEATURE_ID} (Type to search, ↑/↓ arrow keys to navigate)" --height=40% --layout=reverse)
        
        if [ -n "${SELECTED_STATUS}" ]; then
            CHOICE=$(echo "${SELECTED_STATUS}" | awk '{print $1}' | tr -d ')')
        fi
    else
        echo ""
        echo "Select new status for feature ${FEATURE_ID}:"
        echo "  1) idea        (listed, not discussed)"
        echo "  2) specifying  (spec.md in progress)"
        echo "  3) spec'd      (spec.md done, plan.md not started)"
        echo "  4) planned     (plan.md + tasks.md done)"
        echo "  5) in-progress (implementation underway)"
        echo "  6) done        (completed)"
        read -rp "Enter choice [1-6]: " CHOICE
    fi

    case "${CHOICE}" in
        1|idea)        STATUS_INPUT="idea" ;;
        2|specifying)  STATUS_INPUT="specifying" ;;
        3|spec\'d|specd) STATUS_INPUT="spec'd" ;;
        4|planned)     STATUS_INPUT="planned" ;;
        5|in-progress) STATUS_INPUT="in-progress" ;;
        6|done)        STATUS_INPUT="done" ;;
        *)
            echo "Invalid choice."
            exit 1
            ;;
    esac
fi

awk -v id="${FEATURE_ID}" -v new_status="${STATUS_INPUT}" '
BEGIN { FS="|"; OFS="|" }
{
    if ($0 ~ "^\\|[[:space:]]*" id "[[:space:]]*\\|") {
        $6 = " " new_status " "
    }
    print $0
}
' "${INDEX_FILE}" > "${INDEX_FILE}.tmp" && mv "${INDEX_FILE}.tmp" "${INDEX_FILE}"

echo "Updated feature ${FEATURE_ID} status to: ${STATUS_INPUT} in specs/INDEX.md"
