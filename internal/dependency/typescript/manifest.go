// Package typescript resolves concrete installed dependency versions for
// TS/JS dependency-bucket frames, by parsing package.json and
// package-lock.json directly at the repo root -- never shelling out to
// npm/yarn/pnpm (constitution Article IX's own carve-out). See
// specs/006b-ts-js-dependency-resolution for the full design.
//
// This package doc comment currently lives here (manifest.go) rather
// than on resolve.go, the package's actual orchestration entry point --
// resolve.go doesn't exist yet (T004). Move it there once resolve.go
// lands, per CONVENTIONS.md's "one file carries the doc comment, usually
// the file most central to the package" rule.
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
