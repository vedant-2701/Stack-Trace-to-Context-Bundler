package clipboard

import (
	"os"
	"strings"
)

// getenv and readProcVersion are indirected through package-level vars so
// wsl_test.go can inject fake environment/file-read behavior without a
// real WSL environment (CONVENTIONS.md's testing section) -- the same
// small-seam, hand-written-fake shape as cmdRunner (runner.go), just for
// os.Getenv/os.ReadFile instead of a subprocess call.
var (
	getenv          = os.Getenv
	readProcVersion = func() ([]byte, error) { return os.ReadFile("/proc/version") }
)

// isWSL reports whether the current process is running inside Windows
// Subsystem for Linux (spec.md FR3). Checks the WSL_DISTRO_NAME and
// WSL_INTEROP environment variables first (no I/O), then falls back to a
// case-insensitive "microsoft" substring check against /proc/version's
// contents -- the combination of checks WSL-detection utilities commonly
// use. Called by Write (clipboard.go) only when runtime.GOOS == "linux".
func isWSL() bool {
	if getenv("WSL_DISTRO_NAME") != "" || getenv("WSL_INTEROP") != "" {
		return true
	}

	contents, err := readProcVersion()
	if err != nil {
		return false
	}

	return strings.Contains(strings.ToLower(string(contents)), "microsoft")
}
