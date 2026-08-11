#!/usr/bin/env bash
# specs/002a-cli-input-handling/verify-t009.sh
#
# Closes the remaining gaps in spec.md's Acceptance Criteria that T007/
# T008's manual run-throughs didn't already exercise against a real
# binary: file-not-found, empty/whitespace-only input, and --format=yaml.
# Everything else in spec.md's checklist was already confirmed via real
# binaries in T007 (cmd/all) and T008 (cmd/java, cmd/typescript) --
# re-running those again here would pad this script without adding
# rigor. --format=yaml is checked on cmd/all only: validateFormat is the
# identical call site on every binary's parse path, already exhaustively
# unit-tested: this confirms the wiring works via one real binary, not a
# per-binary edge case.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

WORKDIR="$(mktemp -d)"
BIN="$WORKDIR/stba"
trap 'rm -rf "$WORKDIR"' EXIT

section() { printf '\n\033[1m=== %s ===\033[0m\n' "$1"; }

run_case() {
	local desc="$1"
	shift
	local out err rc
	out="$WORKDIR/out.$$"
	err="$WORKDIR/err.$$"

	"$@" >"$out" 2>"$err"
	rc=$?

	section "$desc"
	echo "--- STDOUT ---"
	cat "$out"
	if [ -s "$out" ]; then
		echo "!!! WARNING: stdout was NOT empty !!!"
	else
		echo "(empty, as expected)"
	fi
	echo "--- STDERR ---"
	cat "$err"
	echo "--- exit: $rc ---"

	rm -f "$out" "$err"
}

section "Build"
go build -o "$BIN" ./cmd/all
echo "built: $BIN"

run_case "1. Nonexistent file" "$BIN" "$WORKDIR/does-not-exist.txt"

printf '' >"$WORKDIR/empty.txt"
run_case "2. Empty file" "$BIN" "$WORKDIR/empty.txt"

printf '   \n\t\n  ' | run_case "3. Whitespace-only stdin" "$BIN"

printf 'trace\n' | run_case "4. --format=yaml" "$BIN" --format=yaml

section "Done"
cat <<'EOF'
Read everything above by eye. Specifically check:
  - case 1's message includes the nonexistent file's path
  - case 2 and 3 both say "input is empty" (same message, both sources)
  - case 4 names "yaml" and lists "json"/"markdown" as accepted
  - all four exit 2, all four have empty stdout
EOF
