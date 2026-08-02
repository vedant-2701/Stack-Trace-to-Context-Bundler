#!/usr/bin/env bash

# scripts/init-repo/run.sh
# One-time bootstrap: run right after cloning/downloading this template,
# before starting the kickoff chat.
#
# Moves the SDD process guide out of the README.md slot and promotes the
# product's placeholder README into that slot, so anyone opening the repo
# on GitHub sees the actual project, not the template's own documentation.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

README="${REPO_ROOT}/README.md"
SAMPLE="${REPO_ROOT}/SAMPLE-README.md"
DOCS_DIR="${REPO_ROOT}/docs"
GUIDE="${DOCS_DIR}/SDD-GUIDE.md"

if [ -f "${GUIDE}" ]; then
    echo "Error: ${GUIDE} already exists. This repo looks already initialized."
    echo "Nothing changed — delete docs/SDD-GUIDE.md first if you really want to re-run this."
    exit 1
fi

if [ ! -f "${README}" ]; then
    echo "Error: README.md not found at ${README}."
    exit 1
fi

if [ ! -f "${SAMPLE}" ]; then
    echo "Error: SAMPLE-README.md not found at ${SAMPLE}."
    exit 1
fi

mkdir -p "${DOCS_DIR}"

echo "Moving current README.md -> docs/SDD-GUIDE.md ..."
mv "${README}" "${GUIDE}"

echo "Promoting SAMPLE-README.md -> README.md ..."
mv "${SAMPLE}" "${README}"

echo ""
echo "Done."
echo "  - The SDD process guide now lives at docs/SDD-GUIDE.md"
echo "  - README.md is now the (currently placeholder) product README,"
echo "    to be filled in during the kickoff chat"
echo ""
echo "This script has done its job — you won't need it again for this repo."