package typescript

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// writeFile writes contents to name under dir, creating dir if needed.
func writeFile(t *testing.T, dir, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", name, err)
	}
}

// depFrame builds a minimal dependency-bucket contract.Frame for test
// chains -- only FilePath/PackageName/Bucket matter to this package.
func depFrame(filePath, packageName string) contract.Frame {
	return contract.Frame{FilePath: filePath, PackageName: packageName, Bucket: contract.BucketDependency}
}

// chainOf wraps frames into a single-node Chain, sufficient for every
// test below (multi-node chains don't change referencedPackages'
// behavior -- it already ranges over every node).
func chainOf(frames ...contract.Frame) []contract.ExceptionNode {
	return []contract.ExceptionNode{{Frames: frames}}
}

// alwaysFound is a findRepoRoot stub reporting root as the repo root
// unconditionally -- used by every test below that isn't specifically
// exercising the root-resolution paths themselves.
func alwaysFound(root string) func(context.Context, string) (string, bool) {
	return func(context.Context, string) (string, bool) { return root, true }
}

// neverFound is a findRepoRoot stub reporting no git repo found.
func neverFound(context.Context, string) (string, bool) {
	return "", false
}

const validLockfileV2 = `{
	"lockfileVersion": 2,
	"packages": {
		"node_modules/lodash": {"version": "4.17.21"},
		"node_modules/@babel/core": {"version": "7.24.0"},
		"node_modules/statuses": {"version": "1.0.0"},
		"node_modules/express/node_modules/statuses": {"version": "2.0.0"}
	}
}`

func TestResolveDependencies_ExactMatch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies": {"lodash": "^4.17.21"}}`)
	writeFile(t, root, "package-lock.json", validLockfileV2)

	chain := chainOf(depFrame(filepath.Join(root, "node_modules/lodash/index.js"), "lodash"))

	got := resolveDependencies(context.Background(), root, chain, alwaysFound(root))
	if got == nil {
		t.Fatalf("resolveDependencies() = nil, want a populated result")
	}
	ld, ok := got.Locked["lodash"]
	if !ok {
		t.Fatalf("Locked[%q] missing", "lodash")
	}
	if ld.Version != "4.17.21" || ld.Note != "" {
		t.Errorf("Locked[%q] = %+v, want Version 4.17.21, empty Note", "lodash", ld)
	}
}

func TestResolveDependencies_ScopedPackageExactMatch(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies": {"@babel/core": "^7.24.0"}}`)
	writeFile(t, root, "package-lock.json", validLockfileV2)

	chain := chainOf(depFrame(filepath.Join(root, "node_modules/@babel/core/lib/index.js"), "@babel/core"))

	got := resolveDependencies(context.Background(), root, chain, alwaysFound(root))
	ld := got.Locked["@babel/core"]
	if ld.Version != "7.24.0" || ld.Note != "" {
		t.Errorf("Locked[%q] = %+v, want Version 7.24.0, empty Note", "@babel/core", ld)
	}
	if _, ok := got.Direct["@babel/core"]; !ok {
		t.Errorf("Direct missing %q", "@babel/core")
	}
}

func TestResolveDependencies_TopLevelFallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies": {"lodash": "^4.17.21"}}`)
	writeFile(t, root, "package-lock.json", validLockfileV2)

	// Nested location has no lockfile entry; only the top-level lodash
	// entry exists.
	chain := chainOf(depFrame(filepath.Join(root, "node_modules/some-tool/node_modules/lodash/index.js"), "lodash"))

	got := resolveDependencies(context.Background(), root, chain, alwaysFound(root))
	ld := got.Locked["lodash"]
	if ld.Version != "4.17.21" {
		t.Errorf("Locked[%q].Version = %q, want %q", "lodash", ld.Version, "4.17.21")
	}
	if ld.Note == "" {
		t.Error("Locked[\"lodash\"].Note is empty, want a note flagging the inexact fallback match")
	}
}

func TestResolveDependencies_Conflict(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies": {"statuses": "^1.0.0"}}`)
	writeFile(t, root, "package-lock.json", validLockfileV2)

	// Two frames, same PackageName, exact matches to two different
	// nested copies at different versions (1.0.0 vs 2.0.0).
	chain := chainOf(
		depFrame(filepath.Join(root, "node_modules/statuses/index.js"), "statuses"),
		depFrame(filepath.Join(root, "node_modules/express/node_modules/statuses/index.js"), "statuses"),
	)

	got := resolveDependencies(context.Background(), root, chain, alwaysFound(root))
	ld := got.Locked["statuses"]
	if ld.Version != "" {
		t.Errorf("Locked[%q].Version = %q, want empty (conflict)", "statuses", ld.Version)
	}
	if ld.Note == "" {
		t.Error("Locked[\"statuses\"].Note is empty, want a note naming the conflicting versions")
	}
}

func TestResolveDependencies_OneResolvesOneDoesNot_NoConflictNoBlanking(t *testing.T) {
	root := t.TempDir()
	// "target-pkg" has ONLY a nested lockfile entry -- no top-level
	// node_modules/target-pkg key exists at all, so a frame with no
	// matching exact key genuinely has nothing to fall back to either.
	writeFile(t, root, "package-lock.json", `{
		"lockfileVersion": 2,
		"packages": {
			"node_modules/some-tool/node_modules/target-pkg": {"version": "1.0.0"}
		}
	}`)

	packages, ok, _ := readLockfile(filepath.Join(root, "package-lock.json"))
	if !ok {
		t.Fatalf("readLockfile() ok = false, want true")
	}

	// Two frames of the SAME PackageName ("target-pkg"): the first
	// exact-matches the nested entry; the second's own nested key isn't
	// in the lockfile, and "target-pkg" has no top-level entry to fall
	// back to either -- a genuine non-match for that frame, not a
	// second value that could conflict with the first.
	filePaths := []string{
		filepath.Join(root, "node_modules/some-tool/node_modules/target-pkg/index.js"),  // exact match -> 1.0.0
		filepath.Join(root, "node_modules/other-tool/node_modules/target-pkg/index.js"), // no exact key, no top-level fallback entry exists either -> no match at all
	}

	ld := resolvePackage("target-pkg", filePaths, root, packages, "")
	if ld.Version != "1.0.0" {
		t.Errorf("Locked[%q].Version = %q, want %q (the one frame that resolved)", "target-pkg", ld.Version, "1.0.0")
	}
	if ld.Note != "" {
		t.Errorf("Locked[%q].Note = %q, want empty -- exact match confirms it, second frame's non-match shouldn't blank or caveat it", "target-pkg", ld.Note)
	}
}

func TestResolveDependencies_ExactConfirmsDespiteAgreeingFallbackFrame(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package-lock.json", validLockfileV2) // includes a top-level node_modules/lodash entry

	packages, ok, _ := readLockfile(filepath.Join(root, "package-lock.json"))
	if !ok {
		t.Fatalf("readLockfile() ok = false, want true")
	}

	// Frame 1 exact-matches the top-level lodash entry directly. Frame 2
	// has no node_modules segment at all, so it falls back -- and
	// happens to agree on the SAME version via that same top-level
	// entry. Exercises the judgment call documented in agreedPackage's
	// doc comment: an exact match from one frame confirms the package
	// (no note), even though a different, fallback-only frame also
	// agreed with it.
	filePaths := []string{
		filepath.Join(root, "node_modules/lodash/index.js"),
		filepath.Join(root, "src/somewhere/not-a-dependency-path.js"),
	}

	ld := resolvePackage("lodash", filePaths, root, packages, "")
	if ld.Version != "4.17.21" || ld.Note != "" {
		t.Errorf("resolvePackage() = %+v, want Version 4.17.21 with no Note (exact match confirms despite an agreeing fallback-only frame)", ld)
	}
}

func TestResolveDependencies_FullyUnresolved(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies": {"left-pad": "^1.0.0"}}`)
	writeFile(t, root, "package-lock.json", validLockfileV2)

	chain := chainOf(depFrame(filepath.Join(root, "node_modules/left-pad/index.js"), "left-pad"))

	got := resolveDependencies(context.Background(), root, chain, alwaysFound(root))
	ld := got.Locked["left-pad"]
	if ld.Version != "" {
		t.Errorf("Locked[%q].Version = %q, want empty", "left-pad", ld.Version)
	}
	if ld.Note == "" {
		t.Error("Locked[\"left-pad\"].Note is empty, want an explanation")
	}
}

func TestResolveDependencies_LockfileMissing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies": {"lodash": "^4.17.21"}}`)
	// No package-lock.json written at all.

	chain := chainOf(depFrame(filepath.Join(root, "node_modules/lodash/index.js"), "lodash"))

	got := resolveDependencies(context.Background(), root, chain, alwaysFound(root))
	ld := got.Locked["lodash"]
	if ld.Version != "" || ld.Note == "" {
		t.Errorf("Locked[%q] = %+v, want empty Version and a non-empty Note", "lodash", ld)
	}
}

func TestResolveDependencies_YarnOnly_TreatedAsMissing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies": {"lodash": "^4.17.21"}}`)
	writeFile(t, root, "yarn.lock", "# yarn lockfile v1\n")

	chain := chainOf(depFrame(filepath.Join(root, "node_modules/lodash/index.js"), "lodash"))

	got := resolveDependencies(context.Background(), root, chain, alwaysFound(root))
	ld := got.Locked["lodash"]
	if ld.Version != "" || ld.Note == "" {
		t.Errorf("Locked[%q] = %+v, want empty Version and a non-empty Note, same as missing lockfile", "lodash", ld)
	}
}

func TestResolveDependencies_LockfileMalformed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies": {"lodash": "^4.17.21"}}`)
	writeFile(t, root, "package-lock.json", `{ not valid JSON`)

	chain := chainOf(depFrame(filepath.Join(root, "node_modules/lodash/index.js"), "lodash"))

	got := resolveDependencies(context.Background(), root, chain, alwaysFound(root))
	ld := got.Locked["lodash"]
	if ld.Version != "" || ld.Note == "" {
		t.Errorf("Locked[%q] = %+v, want empty Version and a non-empty Note", "lodash", ld)
	}
}

func TestResolveDependencies_LockfileVersion1(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies": {"lodash": "^4.17.21"}}`)
	writeFile(t, root, "package-lock.json", `{"lockfileVersion": 1, "dependencies": {"lodash": {"version": "4.17.21"}}}`)

	chain := chainOf(depFrame(filepath.Join(root, "node_modules/lodash/index.js"), "lodash"))

	got := resolveDependencies(context.Background(), root, chain, alwaysFound(root))
	ld := got.Locked["lodash"]
	if ld.Version != "" || ld.Note == "" {
		t.Errorf("Locked[%q] = %+v, want empty Version and a non-empty Note (unsupported lockfileVersion)", "lodash", ld)
	}
}

func TestResolveDependencies_ManifestMissing(t *testing.T) {
	root := t.TempDir()
	// No package.json written at all.

	got := resolveDependencies(context.Background(), root, nil, alwaysFound(root))
	if got != nil {
		t.Errorf("resolveDependencies() = %+v, want nil", got)
	}
}

func TestResolveDependencies_ManifestMalformed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{ not valid JSON`)

	got := resolveDependencies(context.Background(), root, nil, alwaysFound(root))
	if got != nil {
		t.Errorf("resolveDependencies() = %+v, want nil", got)
	}
}

func TestResolveDependencies_WorkDirFallback_ManifestPresent(t *testing.T) {
	workDir := t.TempDir()
	writeFile(t, workDir, "package.json", `{"dependencies": {"lodash": "^4.17.21"}}`)
	writeFile(t, workDir, "package-lock.json", validLockfileV2)

	chain := chainOf(depFrame(filepath.Join(workDir, "node_modules/lodash/index.js"), "lodash"))

	got := resolveDependencies(context.Background(), workDir, chain, neverFound)
	if got == nil {
		t.Fatalf("resolveDependencies() = nil, want populated via the workDir fallback")
	}
	if got.Locked["lodash"].Version != "4.17.21" {
		t.Errorf("Locked[%q].Version = %q, want %q", "lodash", got.Locked["lodash"].Version, "4.17.21")
	}
}

func TestResolveDependencies_WorkDirFallback_ManifestAbsent_NoPanic(t *testing.T) {
	workDir := t.TempDir()
	// No package.json in workDir either.

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("resolveDependencies() panicked: %v", r)
		}
	}()

	got := resolveDependencies(context.Background(), workDir, nil, neverFound)
	if got != nil {
		t.Errorf("resolveDependencies() = %+v, want nil", got)
	}
}

func TestResolveDependencies_GitSucceedsButNoManifest_NoWorkDirRetry(t *testing.T) {
	gitRoot := t.TempDir()
	// No package.json at gitRoot.
	workDir := t.TempDir()
	writeFile(t, workDir, "package.json", `{"dependencies": {"lodash": "^4.17.21"}}`)
	// package.json DOES exist in workDir, but must NOT be retried once
	// git successfully reports a (manifest-less) root.

	got := resolveDependencies(context.Background(), workDir, nil, alwaysFound(gitRoot))
	if got != nil {
		t.Errorf("resolveDependencies() = %+v, want nil -- workDir must not be retried after a successful git root with no manifest", got)
	}
}

func TestResolveDependencies_DirectScoping(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{
		"dependencies": {"lodash": "^4.17.21"},
		"devDependencies": {"jest": "^29.0.0"},
		"optionalDependencies": {"fsevents": "^2.3.2"},
		"peerDependencies": {"react": "^18.0.0"}
	}`)
	writeFile(t, root, "package-lock.json", `{
		"lockfileVersion": 2,
		"packages": {
			"node_modules/lodash": {"version": "4.17.21"},
			"node_modules/jest": {"version": "29.0.0"},
			"node_modules/fsevents": {"version": "2.3.2"},
			"node_modules/react": {"version": "18.0.0"},
			"node_modules/left-pad": {"version": "1.3.0"}
		}
	}`)

	chain := chainOf(
		depFrame(filepath.Join(root, "node_modules/lodash/index.js"), "lodash"),     // dependencies
		depFrame(filepath.Join(root, "node_modules/jest/index.js"), "jest"),         // devDependencies
		depFrame(filepath.Join(root, "node_modules/fsevents/index.js"), "fsevents"), // optionalDependencies
		depFrame(filepath.Join(root, "node_modules/react/index.js"), "react"),       // peerDependencies -- excluded from Direct
		depFrame(filepath.Join(root, "node_modules/left-pad/index.js"), "left-pad"), // transitive -- not declared at all
		// "unreferenced-package" is declared nowhere in this test's
		// package.json/chain at all -- confirms an undeclared,
		// unreferenced package simply never appears, which is the
		// default/absence case and needs no extra fixture.
	)

	got := resolveDependencies(context.Background(), root, chain, alwaysFound(root))

	for _, pkg := range []string{"lodash", "jest", "fsevents"} {
		if _, ok := got.Direct[pkg]; !ok {
			t.Errorf("Direct missing %q, want present (dependencies/dev/optional all populate Direct)", pkg)
		}
	}
	if _, ok := got.Direct["react"]; ok {
		t.Error("Direct contains \"react\" (peerDependencies-only), want excluded")
	}
	if _, ok := got.Direct["left-pad"]; ok {
		t.Error("Direct contains \"left-pad\" (purely transitive, not declared), want excluded")
	}

	// Locked still attempts every referenced package regardless of
	// Direct membership, including peerDependencies-only and
	// transitive ones.
	for _, pkg := range []string{"lodash", "jest", "fsevents", "react", "left-pad"} {
		if _, ok := got.Locked[pkg]; !ok {
			t.Errorf("Locked missing %q, want present (Locked attempts resolution regardless of Direct)", pkg)
		}
	}
}

func TestResolveDependencies_DeclaredButUnreferenced_ExcludedEntirely(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies": {"lodash": "^4.17.21", "never-used": "^1.0.0"}}`)
	writeFile(t, root, "package-lock.json", `{
		"lockfileVersion": 2,
		"packages": {
			"node_modules/lodash": {"version": "4.17.21"},
			"node_modules/never-used": {"version": "1.0.0"}
		}
	}`)

	// Chain never references "never-used" as a dependency-bucket frame.
	chain := chainOf(depFrame(filepath.Join(root, "node_modules/lodash/index.js"), "lodash"))

	got := resolveDependencies(context.Background(), root, chain, alwaysFound(root))
	if _, ok := got.Direct["never-used"]; ok {
		t.Error("Direct contains \"never-used\", want excluded (declared but never referenced)")
	}
	if _, ok := got.Locked["never-used"]; ok {
		t.Error("Locked contains \"never-used\", want excluded (declared but never referenced)")
	}
}
