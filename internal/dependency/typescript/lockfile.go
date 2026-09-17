package typescript

import (
	"encoding/json"
	"fmt"
	"os"
)

// npmLockfile is package-lock.json's shape, scoped to only the fields
// spec.md's lockfileVersion 2/3 resolution needs (requirement 1).
type npmLockfile struct {
	LockfileVersion int                           `json:"lockfileVersion"`
	Packages        map[string]npmLockfilePackage `json:"packages"`
}

// npmLockfilePackage is one entry in npmLockfile.Packages, scoped to
// only the field spec.md requirement 10c actually reads.
type npmLockfilePackage struct {
	Version string `json:"version"`
}

// readLockfile reads and parses path (package-lock.json) into a flat
// map from the lockfile's own "packages" key -- a node_modules-relative
// directory path, e.g. "node_modules/lodash" or, for a scoped package,
// "node_modules/@babel/core" -- to that entry's resolved version. The
// per-entry npmLockfilePackage wrapper is discarded once past this
// function; lookup.go/resolve.go only ever need the version string.
//
// ok=false covers three distinct unusable cases, spec.md requirement 12,
// each with its own reason string for the caller (T003b/T004) to fold
// into a LockedDependency.Note:
//   - file not found -- this is also what happens in the "only
//     yarn.lock exists" case (requirement 2/12): this function only
//     ever looks for package-lock.json by the exact path it's given and
//     never inspects yarn.lock at all, so yarn.lock's mere presence
//     alongside a missing package-lock.json changes nothing here.
//   - malformed JSON.
//   - lockfileVersion present but not 2 or 3 -- covers npm <=6's
//     lockfileVersion 1 (a structurally different nested-tree format,
//     requirement 1, that this function makes no attempt to parse).
func readLockfile(path string) (packages map[string]string, ok bool, reason string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, "no package-lock.json found"
	}

	var lf npmLockfile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, false, "package-lock.json is not valid JSON"
	}

	if lf.LockfileVersion != 2 && lf.LockfileVersion != 3 {
		return nil, false, fmt.Sprintf("package-lock.json lockfileVersion %d is unsupported (only 2 and 3 are)", lf.LockfileVersion)
	}

	packages = make(map[string]string, len(lf.Packages))
	for key, pkg := range lf.Packages {
		packages[key] = pkg.Version
	}
	return packages, true, ""
}
