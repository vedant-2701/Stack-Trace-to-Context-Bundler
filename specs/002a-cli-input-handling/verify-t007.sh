#!/usr/bin/env bash
# specs/002a-cli-input-handling/verify-t007.sh
#
# Automates T007's manual run-through (tasks.md acceptance criteria) for
# cmd/all against the real built binary. This does NOT replace reading
# the output yourself -- the script flags obvious violations (non-empty
# stdout) but you still need to eyeball the Debug dump contents, the
# exact error message text, and that case 6a truly produced nothing.
#
# One case can't be automated at all: "no input, stdin not piped" needs a
# real interactive TTY, which a script's own stdin never is (even
# unredirected, a script invoked non-interactively may not have one) --
# see the printed instructions at the end for running that one by hand.
#
# T008 will need near-identical coverage for cmd/java and
# cmd/typescript; copy this file to verify-t008.sh and swap the build
# target / drop the --lang cases when that task starts.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

WORKDIR="$(mktemp -d)"
BIN="$WORKDIR/stba"
trap 'rm -rf "$WORKDIR"' EXIT

section() { printf '\n\033[1m=== %s ===\033[0m\n' "$1"; }

# run_case DESCRIPTION -- CMD...
# Runs CMD with stdout/stderr captured separately (stdin, if piped into
# run_case itself, passes through normally). Prints both streams labeled
# and the exit code; warns loudly if stdout was non-empty.
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
		echo "!!! WARNING: stdout was NOT empty -- spec requires nothing ever written to stdout in this feature !!!"
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

printf 'java.lang.NullPointerException\n\tat com.example.Foo.bar(Foo.java:10)\n' >"$WORKDIR/valid.txt"

run_case "1. Valid file" "$BIN" "$WORKDIR/valid.txt"

printf 'TypeError: x is not a function\n' | run_case "2. Valid piped stdin" "$BIN"

printf 'trace\n' | run_case "3. Invalid --lang" "$BIN" --lang=cobol

yes "stack trace line" | head -c 600000 >"$WORKDIR/big.txt"
run_case "4. File > 512KB (check dump for RawInputTruncated=true, ~524288-byte rawText)" "$BIN" "$WORKDIR/big.txt" -vv

run_case "5a. No -v/-vv: expect fully empty stdout AND stderr" "$BIN" "$WORKDIR/valid.txt"
run_case "5b. -v: expect one Info line on stderr, no Debug dump" "$BIN" "$WORKDIR/valid.txt" -v
run_case "5c. -vv: expect Info line + Debug dump on stderr" "$BIN" "$WORKDIR/valid.txt" -vv

printf 'from stdin\n' | run_case "6. File + piped stdin together, -vv: 'stdin ignored' Debug line should appear (this was the T006c bug -- unobservable before that fix)" "$BIN" "$WORKDIR/valid.txt" -vv

section "NOT automated -- run this one yourself"
cat <<EOF
Run, in an actual interactive terminal (not from this script, no pipe,
no redirection):

    $BIN

It should exit immediately (code 2) with a "no input" error on stderr,
without waiting for you to type anything or press Ctrl+D.
EOF

section "Done"
cat <<'EOF'
Read everything above -- this script flags non-empty stdout automatically
but does not grade message content. Specifically check by eye:
  - case 5a produced NOTHING on either stream
  - case 5c's dump is the full JSON struct with all 5 Input fields
  - case 6's stderr shows "stdin ignored", not stdin's actual content
  - every error message (cases 3) names the bad value and accepted set
EOF
