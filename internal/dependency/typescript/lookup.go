package typescript

import (
	"fmt"
	"path/filepath"
	"strings"

	parsetypescript "github.com/vedant-2701/stack-trace-bundler/internal/parser/typescript"
)

// packageDirKey computes the lockfile "packages" map key for a frame's
// absolute FilePath, relative to repoRoot -- preserves the FULL nested
// node_modules/.../node_modules/... prefix chain (unlike 006a's
// bucket.go, which only wants the bare trailing package name), since
// this key must uniquely identify one specific installed copy in a
// nested dependency tree, not just a package name. Uses the shared
// FindLastNodeModulesSegment (extracted from
// internal/parser/typescript's bucket.go by this task, per plan.md's
// "Risks" section) so the normalize+find-last-occurrence logic isn't
// duplicated a third time.
//
// ok=false if FilePath isn't under repoRoot (filepath.Rel error --
// realistically unreachable on Linux between two absolute paths, since
// Rel just produces a "../"-prefixed result rather than erroring; kept
// as a defensive check, not exercised by a dedicated test), or has no
// node_modules segment at all.
func packageDirKey(filePath, repoRoot string) (key string, ok bool) {
	rel, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return "", false
	}

	segments, lastIdx := parsetypescript.FindLastNodeModulesSegment(rel)
	if lastIdx == -1 || lastIdx+1 >= len(segments) {
		return "", false
	}

	end := lastIdx + 1
	if strings.HasPrefix(segments[end], "@") && end+1 < len(segments) {
		end++
	}
	return strings.Join(segments[:end+1], "/"), true
}

// lookupFrame resolves ONE frame's own contribution to packageName's
// locked version, spec.md FR10c/FR11/FR14. This is deliberately not a
// final per-package outcome: resolve.go (T004) aggregates every
// referenced frame's lookupFrame result across a package and applies
// the multi-frame conflict rule (FR13) before deciding what actually
// goes in Locked -- a single frame here can only ever report what IT
// sees, never "the" answer for packageName as a whole.
//
//   - Exact match: packageDirKey(filePath, repoRoot) hits an entry in
//     packages. Reports that version, no note -- this frame's own
//     specific installed copy is known with certainty (FR10c).
//   - Top-level fallback: tried whenever the exact match misses, for
//     ANY reason (no node_modules segment tied to repoRoot at all, a
//     packageDirKey key not present in packages, or packageDirKey
//     failing outright) -- looks up "node_modules/" + packageName
//     (packageName already includes the "@scope/" prefix for scoped
//     packages, so no separate scoped-package branch is needed here).
//     Reports that version WITH a note flagging it's not tied to this
//     frame's exact nested copy (FR11, and the relaxed
//     LockedDependency.Note doc comment from T000).
//   - Neither: matched=false. Not itself "unresolved" -- resolve.go
//     decides the final Note text for this case (FR12), since it also
//     needs to know whether ANY other frame for the same packageName
//     matched, for the conflict-vs-plain-miss distinction (FR13).
func lookupFrame(filePath, packageName, repoRoot string, packages map[string]string) (version, note string, matched bool) {
	if key, ok := packageDirKey(filePath, repoRoot); ok {
		if v, found := packages[key]; found {
			return v, "", true
		}
	}

	if v, found := packages["node_modules/"+packageName]; found {
		return v, fmt.Sprintf("resolved via top-level node_modules/%s, not necessarily the exact nested copy this frame uses", packageName), true
	}

	return "", "", false
}
