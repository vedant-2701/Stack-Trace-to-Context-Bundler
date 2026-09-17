package typescript

import (
	"encoding/json"
	"os"
)

// npmManifest is package.json's shape, scoped to only the three sections
// spec.md FR8 cares about. peerDependencies is deliberately absent as a
// field here, not just unused -- it declares a version range without
// itself causing an install, so it must never leak into Direct even if
// a future edit naively ranges over every field on this struct.
type npmManifest struct {
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

// readManifest reads and parses path (package.json) into a single
// declared-version-range map, spec.md FR8: dependencies +
// devDependencies + optionalDependencies merged, peerDependencies
// excluded entirely. ok=false covers both "file does not exist" and
// "exists but is not valid JSON" identically -- spec.md FR7 treats both
// outcomes the same way (Bundle.Dependencies nil), so this function's
// caller never needs to distinguish them either.
//
// Returns the UNSCOPED map -- every declared package, not just ones
// referenced by a dependency-bucket frame in this trace. Filtering to
// referenced packages only (spec.md FR9) is resolve.go's job (T004),
// not this function's -- mirrors plan.md's pipeline separating manifest
// parsing (this file) from resolution scoping (resolve.go).
//
// No override-precedence logic is needed across the three sections:
// npm itself doesn't allow the same package name to appear in more than
// one of dependencies/devDependencies/optionalDependencies at once, so
// there is no real "which section wins" case to resolve here.
func readManifest(path string) (direct map[string]string, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var m npmManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false
	}

	direct = make(map[string]string, len(m.Dependencies)+len(m.DevDependencies)+len(m.OptionalDependencies))
	for name, version := range m.Dependencies {
		direct[name] = version
	}
	for name, version := range m.DevDependencies {
		direct[name] = version
	}
	for name, version := range m.OptionalDependencies {
		direct[name] = version
	}

	return direct, true
}
