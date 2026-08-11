#!/usr/bin/env bash
# specs/002a-cli-input-handling/verify-t008.sh
#
# Automates T008's manual run-through (tasks.md acceptance criteria) for
# cmd/java and cmd/typescript: a valid run on each binary, and --lang
# being rejected on each. Deliberately narrower than verify-t007.sh --
# T008's acceptance criteria doesn't ask for the -v/-vv, >512KB, or
# interactive-TTY cases again, since those exercise shared internal/cli
# logic already thoroughly verified end-to-end in T007. Read the output
# yourself; this script flags non-empty stdout but doesn't grade message
# content.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

WORKDIR="$(mktemp -d)"
JAVA_BIN="$WORKDIR/stba-java"
TS_BIN="$WORKDIR/stba-typescript"
trap 'rm -rf "$WORKDIR"' EXIT

section() { printf '\n\033[1m=== %s ===\033[0m\n' "$1"; }

# run_case DESCRIPTION -- CMD...
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
go build -o "$JAVA_BIN" ./cmd/java || exit 1
go build -o "$TS_BIN" ./cmd/typescript || exit 1
echo "built: $JAVA_BIN"
echo "built: $TS_BIN"

printf 'java.lang.NullPointerException\n\tat com.example.Foo.bar(Foo.java:10)\n' >"$WORKDIR/java.txt"
printf 'TypeError: x is not a function\n    at Object.<anonymous> (/app/index.js:5:1)\n' >"$WORKDIR/ts.txt"

run_case "1. cmd/java: valid run (flag after positional, per T007a)" "$JAVA_BIN" "$WORKDIR/java.txt" -vv
run_case "2. cmd/java: --lang rejected (never registered on this binary)" "$JAVA_BIN" --lang=java "$WORKDIR/java.txt"

run_case "3. cmd/typescript: valid run (flag after positional, per T007a)" "$TS_BIN" "$WORKDIR/ts.txt" -vv
run_case "4. cmd/typescript: --lang rejected (never registered on this binary)" "$TS_BIN" --lang=typescript "$WORKDIR/ts.txt"

section "Done"
cat <<'EOF'
Read everything above by eye. Specifically check:
  - case 1's dump shows "langHint": "java" (fixed, not from a flag)
  - case 3's dump shows "langHint": "typescript"
  - cases 2 and 4 exit 2 with an "unknown flag" style message mentioning --lang
  - stdout was empty in all four cases
EOF
