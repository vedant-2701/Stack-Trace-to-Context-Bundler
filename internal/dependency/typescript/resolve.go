// Package typescript resolves concrete installed dependency versions for
// TS/JS dependency-bucket frames, by parsing package.json and
// package-lock.json directly at the repo root -- never shelling out to
// npm/yarn/pnpm (constitution Article IX's own carve-out). See
// specs/006b-ts-js-dependency-resolution for the full design.
package typescript

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vedant-2701/stack-trace-bundler/internal/codecontext"
	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// ResolveDependencies is this package's production entry point. Mirrors
// codecontext.BuildGitMetadata's shape -- never returns a Go error,
// never panics: every failure mode here (no repo root AND no manifest
// in workDir either, malformed JSON, a missing/malformed/unsupported-
// version lockfile) is spec.md FR7's own "nil is a valid, representable
// outcome" framing, not an exceptional one. Wraps the root-finder-
// injectable resolveDependencies with codecontext.FindRepoRoot as the
// real implementation.
func ResolveDependencies(ctx context.Context, workDir string, chain []contract.ExceptionNode) *contract.Dependencies {
	return resolveDependencies(ctx, workDir, chain, codecontext.FindRepoRoot)
}

// resolveDependencies takes findRepoRoot as a parameter so
// resolve_test.go can exercise every root-resolution path (workDir
// fallback success, workDir fallback failure, git-succeeds-but-no-
// manifest) with a stub instead of a real git repository or the git
// binary -- CONVENTIONS.md's rule that anything shelling out must be
// tested behind a fake.
func resolveDependencies(ctx context.Context, workDir string, chain []contract.ExceptionNode, findRepoRoot func(ctx context.Context, workDir string) (string, bool)) *contract.Dependencies {
	root, ok := findRepoRoot(ctx, workDir)
	if !ok {
		// No git repo, or the rev-parse call itself timed out -- both
		// treated identically (spec.md FR6). Fall back to workDir
		// directly, no upward walk to a parent directory.
		root = workDir
	}

	manifest, ok := readManifest(filepath.Join(root, "package.json"))
	if !ok {
		// Covers both "not found" and "found but invalid JSON" --
		// spec.md FR7 treats them identically, and applies uniformly
		// whether root came from git or the workDir fallback: there is
		// no third, harder failure mode when both come up empty.
		slog.Warn("dependencies: package.json not found or invalid, Dependencies omitted", "root", root)
		return nil
	}

	referenced := referencedPackages(chain)
	direct := buildDirect(manifest, referenced)
	locked := buildLocked(root, referenced)

	resolved := 0
	for _, ld := range locked {
		if ld.Version != "" {
			resolved++
		}
	}
	slog.Info("dependencies resolved", "count", resolved)

	return &contract.Dependencies{
		ManifestFile: contract.ManifestFilePackageJSON,
		Direct:       direct,
		Locked:       locked,
	}
}

// referencedPackages collects every dependency-bucket frame's
// PackageName across the whole chain (spec.md FR9's scoping), keyed by
// package name, with EVERY referencing frame's own FilePath preserved
// (not deduplicated, and not reduced to a count) -- resolvePackage
// needs each frame's path individually to attempt per-frame resolution
// (spec.md FR10) and detect multi-frame conflicts (spec.md FR13), not
// just the set of package names that exist.
func referencedPackages(chain []contract.ExceptionNode) map[string][]string {
	refs := make(map[string][]string)
	for _, node := range chain {
		for _, frame := range node.Frames {
			if frame.Bucket != contract.BucketDependency {
				continue
			}
			refs[frame.PackageName] = append(refs[frame.PackageName], frame.FilePath)
		}
	}
	return refs
}

// buildDirect scopes manifest's package.json-declared version ranges
// down to only referenced packages (spec.md FR9). A referenced package
// absent from manifest (a purely transitive dependency -- spec.md FR9's
// own example) is simply not added: no entry, no note -- Direct has no
// per-entry note field, so the absence itself is the correct signal.
func buildDirect(manifest map[string]string, referenced map[string][]string) map[string]string {
	direct := make(map[string]string, len(referenced))
	for pkg := range referenced {
		if v, ok := manifest[pkg]; ok {
			direct[pkg] = v
		}
	}
	return direct
}

// buildLocked resolves every referenced package's Locked entry:
// per-frame exact/fallback lookup (lookup.go's lookupFrame), then the
// multi-frame conflict rule (spec.md FR13), via resolvePackage below.
//
// package-lock.json is read exactly once here, not once per package --
// its own not-found/malformed/unsupported-lockfileVersion outcome
// (spec.md FR12) is identical for every referenced package, so
// lockReason is computed once and reused as every affected package's
// Note, rather than each package re-deriving the same explanation
// independently.
func buildLocked(root string, referenced map[string][]string) map[string]contract.LockedDependency {
	packages, lockOK, lockReason := readLockfile(filepath.Join(root, "package-lock.json"))
	if !lockOK {
		slog.Warn("dependencies: package-lock.json unusable, every referenced package's Locked entry left unresolved", "reason", lockReason)
	}

	locked := make(map[string]contract.LockedDependency, len(referenced))
	for pkg, filePaths := range referenced {
		locked[pkg] = resolvePackage(pkg, filePaths, root, packages, lockReason)
	}
	return locked
}

// frameMatch is one referencing frame's own lookupFrame result, kept
// alongside the others so resolvePackage can compare all of a
// package's frames together before deciding a final outcome.
type frameMatch struct {
	version string
	note    string
}

// resolvePackage implements spec.md FR10-FR14 for ONE referenced
// package: calls lookup.go's lookupFrame once per referencing frame,
// then decides the package's single final Locked entry from every
// frame's contribution together -- never lets whichever frame is
// processed last silently win (spec.md FR13's own framing).
func resolvePackage(pkg string, filePaths []string, root string, packages map[string]string, lockReason string) contract.LockedDependency {
	var matches []frameMatch
	for _, fp := range filePaths {
		version, note, matched := lookupFrame(fp, pkg, root, packages)
		if matched {
			matches = append(matches, frameMatch{version, note})
		}
	}

	if len(matches) == 0 {
		return unresolvedPackage(pkg, lockReason)
	}

	distinct := make(map[string]bool, len(matches))
	for _, m := range matches {
		distinct[m.version] = true
	}
	if len(distinct) > 1 {
		return conflictingPackage(pkg, distinct)
	}

	return agreedPackage(pkg, matches)
}

// unresolvedPackage handles spec.md FR12: no referencing frame matched
// at all, either because the lockfile itself was unusable (lockReason
// non-empty -- reused verbatim, since it already explains exactly why)
// or because the lockfile is fine but this specific package just isn't
// in it under any key this feature checks.
func unresolvedPackage(pkg, lockReason string) contract.LockedDependency {
	reason := lockReason
	if reason == "" {
		reason = fmt.Sprintf("no matching entry found in package-lock.json for %s", pkg)
	}
	slog.Warn("dependencies: package unresolved", "package", pkg, "reason", reason)
	return contract.LockedDependency{Note: reason}
}

// conflictingPackage handles spec.md FR13: two or more referencing
// frames resolved to different versions -- an omitted Version plus a
// Note naming every distinct version found (sorted, for deterministic
// output), not which frame each came from. Spec.md marks per-frame
// attribution as "where useful," not required, and a plain sorted
// version list is already unambiguous on its own.
func conflictingPackage(pkg string, distinct map[string]bool) contract.LockedDependency {
	versions := make([]string, 0, len(distinct))
	for v := range distinct {
		versions = append(versions, v)
	}
	sort.Strings(versions)

	slog.Warn("dependencies: version conflict across referenced frames", "package", pkg, "versions", versions)
	return contract.LockedDependency{
		Note: fmt.Sprintf("conflicting versions found across referenced frames: %s", strings.Join(versions, ", ")),
	}
}

// agreedPackage handles the case every referencing frame that matched
// agrees on one version. Judgment call (flagged, not in spec.md's own
// wording): if ANY contributing frame reached that version via an
// exact path match (frameMatch.note == ""), the package is reported as
// confirmed -- no note -- even if a different frame only reached the
// same value through the top-level fallback; an exact match is ground
// truth regardless of what a fallback-only frame also happened to
// agree with. Only when EVERY contributing frame's match was itself a
// fallback does the final entry carry a fallback note, reusing the
// first such frame's own note text (deterministic: filePaths, and
// therefore matches, are processed in referencedPackages' own frame
// order) -- there's nothing more specific to add beyond what lookup.go
// already said for that frame.
func agreedPackage(pkg string, matches []frameMatch) contract.LockedDependency {
	version := matches[0].version

	for _, m := range matches {
		if m.note == "" {
			return contract.LockedDependency{Version: version}
		}
	}

	slog.Warn("dependencies: resolved via inexact fallback match only", "package", pkg)
	return contract.LockedDependency{Version: version, Note: matches[0].note}
}
