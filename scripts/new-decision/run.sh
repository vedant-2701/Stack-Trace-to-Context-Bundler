#!/usr/bin/env bash

# scripts/new-decision/run.sh
# Creates a new Architecture Decision Record (ADR) in memory/decisions/ and
# registers it in memory/decisions/INDEX.md.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DECISIONS_DIR="${REPO_ROOT}/memory/decisions"
TEMPLATE_FILE="${DECISIONS_DIR}/_template.md"
INDEX_FILE="${DECISIONS_DIR}/INDEX.md"

if [ ! -d "${DECISIONS_DIR}" ]; then
    echo "Error: memory/decisions directory not found at ${DECISIONS_DIR}."
    exit 1
fi

TITLE_INPUT="$1"
DEPENDS_INPUT="$2"

if [ -z "${TITLE_INPUT}" ]; then
    echo "=== New Architecture Decision Record ==="
    read -rp "Decision title (e.g. 'Use PostgreSQL for persistent storage'): " TITLE_INPUT
    if [ -z "${TITLE_INPUT}" ]; then
        echo "Error: Title cannot be empty."
        exit 1
    fi
fi

if [ -z "${DEPENDS_INPUT}" ]; then
    read -rp "Depends on (decision ID or none) [none]: " DEPENDS_INPUT
    if [ -z "${DEPENDS_INPUT}" ]; then
        DEPENDS_INPUT="none"
    fi
fi

SLUG=$(echo "${TITLE_INPUT}" | tr '[:upper:]' '[:lower:]' | sed -E 's/[[:space:]_]+/-/g' | sed -E 's/[^a-z0-9-]//g' | sed -E 's/^-+|-+$//g')

MAX_ID=0
for f in "${DECISIONS_DIR}"/*.md; do
    if [ -f "$f" ]; then
        filename=$(basename "$f")
        if [[ "$filename" =~ ^([0-9]{4})- ]]; then
            num=$((10#${BASH_REMATCH[1]}))
            if [ "$num" -gt "$MAX_ID" ]; then
                MAX_ID=$num
            fi
        fi
    fi
done

NEXT_NUM=$((MAX_ID + 1))
NEXT_ID=$(printf "%04d" "$NEXT_NUM")
DATE_NOW=$(date +%Y-%m-%d)
TARGET_FILE="${DECISIONS_DIR}/${NEXT_ID}-${SLUG}.md"

if [ -f "${TARGET_FILE}" ]; then
    echo "Error: Decision record ${TARGET_FILE} already exists."
    exit 1
fi

if [ -f "${TEMPLATE_FILE}" ]; then
    sed -e "s/{{ID}}/${NEXT_ID}/g" \
        -e "s/{{DECISION_TITLE}}/${TITLE_INPUT}/g" \
        -e "s/{{DATE}}/${DATE_NOW}/g" \
        -e "s/{{DEPENDS}}/${DEPENDS_INPUT}/g" \
        "${TEMPLATE_FILE}" > "${TARGET_FILE}"
else
    cat << EOF > "${TARGET_FILE}"
# ${NEXT_ID} — ${TITLE_INPUT}

**Status:** Proposed | Accepted | Superseded by 000X
**Date:** ${DATE_NOW}
**Depends on:** ${DEPENDS_INPUT}

## Context

## Decision

## Alternatives considered

## Consequences
EOF
fi

echo "Created decision record: memory/decisions/${NEXT_ID}-${SLUG}.md"

if [ -f "${INDEX_FILE}" ]; then
    NEW_ROW="| ${NEXT_ID} | ${TITLE_INPUT} | ${DEPENDS_INPUT} | Proposed | ${DATE_NOW} |"
    LAST_ROW_LINE=$(grep -n '^|' "${INDEX_FILE}" | tail -1 | cut -d: -f1)
    if [ -n "${LAST_ROW_LINE}" ]; then
        awk -v line="${LAST_ROW_LINE}" -v row="${NEW_ROW}" \
            'NR==line{print; print row; next} {print}' \
            "${INDEX_FILE}" > "${INDEX_FILE}.tmp" && mv "${INDEX_FILE}.tmp" "${INDEX_FILE}"
    else
        echo "${NEW_ROW}" >> "${INDEX_FILE}"
    fi
    echo "Added entry to memory/decisions/INDEX.md:"
    echo "${NEW_ROW}"
else
    echo "Warning: memory/decisions/INDEX.md not found — decision created but not indexed."
fi