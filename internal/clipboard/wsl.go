package clipboard

import (
	"os"
	"strings"
)

// lookupEnv and readProcVersion are indirected through package-level vars
// so wsl_test.go can inject fake environment/file-read behavior without a
// real WSL environment (CONVENTIONS.md's testing section) -- the same
// small-seam, hand-written-fake shape as cmdRunner (runner.go), just for
// os.LookupEnv/os.ReadFile instead of a subprocess call. LookupEnv (not
// Getenv) is deliberate: spec.md FR3 says WSL_DISTRO_NAME/WSL_INTEROP
// need only be "set," not set to a non-empty value, and Getenv can't
// distinguish "unset" from "set to empty string."
var (
	lookupEnv       = os.LookupEnv
	readProcVersion = func() ([]byte, error) { return os.ReadFile("/proc/version") }
)

// isWSL reports whether the current process is running inside Windows
// Subsystem for Linux (spec.md FR3). Checks the WSL_DISTRO_NAME and
// WSL_INTEROP environment variables first (no I/O) -- present at all,
// regardless of value -- then falls back to a case-insensitive
// "microsoft" substring check against /proc/version's contents. This is
// the combination of checks WSL-detection utilities commonly use.
// Called by Write (clipboard.go) only when runtime.GOOS == "linux".
func isWSL() bool {
	if _, ok := lookupEnv("WSL_DISTRO_NAME"); ok {
		return true
	}
	if _, ok := lookupEnv("WSL_INTEROP"); ok {
		return true
	}

	contents, err := readProcVersion()
	if err != nil {
		return false
	}

	return strings.Contains(strings.ToLower(string(contents)), "microsoft")
}
