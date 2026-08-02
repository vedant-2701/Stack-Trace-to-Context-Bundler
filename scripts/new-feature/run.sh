#!/usr/bin/env bash

# scripts/new-feature/run.sh
# Creates a new feature folder under specs/ from templates and registers it
# in specs/INDEX.md.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SPECS_DIR="${REPO_ROOT}/specs"
TEMPLATES_DIR="${SPECS_DIR}/_templates"
INDEX_FILE="${SPECS_DIR}/INDEX.md"

if [ ! -d "${TEMPLATES_DIR}" ]; then
    echo "Error: specs/_templates directory not found at ${TEMPLATES_DIR}."
    exit 1
fi

FEATURE_NAME="$1"
FEATURE_DESC="$2"
FEATURE_DEPENDS="$3"

if [ -z "${FEATURE_NAME}" ]; then
    echo "=== New Feature ==="
    read -rp "Feature name (e.g. 'user-auth'): " FEATURE_NAME
    if [ -z "${FEATURE_NAME}" ]; then
        echo "Error: Feature name cannot be empty."
        exit 1
    fi
    read -rp "One-line description: " FEATURE_DESC
    read -rp "Depends on (feature ID or none) [none]: " FEATURE_DEPENDS
    if [ -z "${FEATURE_DEPENDS}" ]; then
        FEATURE_DEPENDS="none"
    fi
fi

if [ -z "${FEATURE_DEPENDS}" ]; then
    FEATURE_DEPENDS="none"
fi

FEATURE_SLUG=$(echo "${FEATURE_NAME}" | tr '[:upper:]' '[:lower:]' | sed -E 's/[[:space:]_]+/-/g' | sed -E 's/[^a-z0-9-]//g' | sed -E 's/^-+|-+$//g')

MAX_ID=0
for d in "${SPECS_DIR}"/*/; do
    dirname=$(basename "$d")
    if [[ "$dirname" =~ ^([0-9]{3})- ]]; then
        num=$((10#${BASH_REMATCH[1]}))
        if [ "$num" -gt "$MAX_ID" ]; then
            MAX_ID=$num
        fi
    fi
done

NEXT_NUM=$((MAX_ID + 1))
NEXT_ID=$(printf "%03d" "$NEXT_NUM")
TARGET_DIR="${SPECS_DIR}/${NEXT_ID}-${FEATURE_SLUG}"

if [ -d "${TARGET_DIR}" ]; then
    echo "Error: Directory ${TARGET_DIR} already exists."
    exit 1
fi

echo "Creating feature ${NEXT_ID}-${FEATURE_SLUG}..."
mkdir -p "${TARGET_DIR}"

for template_file in "${TEMPLATES_DIR}"/*.md; do
    if [ -f "${template_file}" ]; then
        filename=$(basename "${template_file}")
        target_file="${TARGET_DIR}/${filename}"

        sed -e "s/{{ID}}/${NEXT_ID}/g" \
            -e "s/{{FEATURE_SLUG}}/${FEATURE_SLUG}/g" \
            -e "s/{{FEATURE_NAME}}/${FEATURE_NAME}/g" \
            "${template_file}" > "${target_file}"
    fi
done

if [ -f "${INDEX_FILE}" ]; then
    NEW_ROW="| ${NEXT_ID} | ${FEATURE_NAME} | ${FEATURE_DESC} | ${FEATURE_DEPENDS} | idea |"
    LAST_ROW_LINE=$(grep -n '^|' "${INDEX_FILE}" | tail -1 | cut -d: -f1)
    if [ -n "${LAST_ROW_LINE}" ]; then
        awk -v line="${LAST_ROW_LINE}" -v row="${NEW_ROW}" \
            'NR==line{print; print row; next} {print}' \
            "${INDEX_FILE}" > "${INDEX_FILE}.tmp" && mv "${INDEX_FILE}.tmp" "${INDEX_FILE}"
    else
        echo "${NEW_ROW}" >> "${INDEX_FILE}"
    fi
    echo "Added entry to specs/INDEX.md:"
    echo "${NEW_ROW}"
fi

echo ""
echo "Successfully created feature specifications in ${TARGET_DIR}:"
echo "  - spec.md"
echo "  - plan.md"
echo "  - progress.md"
echo "  - tasks.md"