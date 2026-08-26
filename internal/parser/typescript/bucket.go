package typescript

import (
	"strings"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// normalizeFileURI strips a "file://" scheme prefix from path, per
// spec.md FR12: Node ESM mode can emit "file:///abs/path" frame paths
// (confirmed real via testdata/esm-runtime-error.txt), and
// contract.Frame.FilePath's doc comment requires "never a URI" so
// CodeContext's later git-blame/snippet-extraction step always gets a
// real filesystem path. Paths with no "file://" prefix (the common
// case) pass through unchanged. Only the scheme is stripped, not
// parsed via net/url -- every real fixture is a Linux absolute path
// (file:///abs/... -> /abs/...); this project has no Windows fixture to
// confirm file:///C:/... handling against, and stdlib-only regexp/
// strings parsing (per plan.md's Stack & versions section) doesn't need
// a URL parser for this one prefix-strip.
func normalizeFileURI(path string) string {
	return strings.TrimPrefix(path, "file://")
}

// splitAfterLastNodeModules reports whether path contains a
// "node_modules" path segment and, if so, the package name segment(s)
// immediately following the LAST such occurrence (spec.md FR10 --
// handles nested dependency trees, confirmed real via
// testdata/deep-nested-node-modules.txt's
// "node_modules/express/node_modules/finalhandler/node_modules/statuses/"
// case, where the correct package is "statuses", not "express" or
// "finalhandler"). If that segment starts with "@", the next segment is
// appended too (scoped packages, e.g. "@babel/core", confirmed real via
// testdata/scoped-package-babel.txt).
//
// Segment-equality is used, not a raw strings.Contains(path,
// "node_modules") check, so a real project directory that merely
// contains the substring "node_modules" somewhere in its name (e.g.
// "my-node_modules-cache/") is never misidentified as an actual
// dependency path -- no real fixture exercises this, but the segment
// check costs nothing extra and avoids a class of bug the substring
// check would silently invite.
func splitAfterLastNodeModules(path string) (packageName string, isDependency bool) {
	segments := strings.Split(path, "/")

	lastIdx := -1
	for i, s := range segments {
		if s == "node_modules" {
			lastIdx = i
		}
	}
	if lastIdx == -1 {
		return "", false
	}
	if lastIdx+1 >= len(segments) {
		// A "node_modules" segment with nothing after it -- no real
		// fixture produces this, but it's still a dependency-bucket
		// path, just with no package name to report.
		return "", true
	}

	packageName = segments[lastIdx+1]
	if strings.HasPrefix(packageName, "@") && lastIdx+2 < len(segments) {
		packageName += "/" + segments[lastIdx+2]
	}
	return packageName, true
}

// assignBucket implements spec.md FR10/FR11: classifies a single frame
// into exactly one of the three Bucket values and, for BucketDependency,
// sets PackageName. Runs after normalizeFileURI (FR12) so bucketing
// always sees a real filesystem path, never a "file://" URI.
//
// Per plan.md's pipeline step 4, this runs per-frame, independent of
// which ExceptionNode/chain position the frame belongs to -- bucketing
// rules don't depend on that.
//
// FR10's second BucketRuntime condition ("the frame has no file info at
// all (<anonymous>, native binding)") is checked here via
// f.FilePath == "" -- but T002's parseFrameLine can never actually
// construct a contract.Frame with an empty FilePath: locationPattern
// requires a trailing ":<line>:<col>" with real digits, so a genuine
// V8 native frame with no location at all (e.g. "at <anonymous>" alone,
// or "at native") fails to match frameLinePattern/locationPattern
// entirely and is dropped during block parsing as "not a frame line" --
// it never reaches this function as a contract.Frame in the first
// place. This branch is therefore defensive/spec-literal rather than
// reachable via the real parse pipeline today; flagged rather than
// silently assumed reachable, since bucket_test.go's "anonymous" case
// has to construct that Frame directly rather than load it from a real
// fixture.
//
// IMPORTANT, and easy to get backwards: "<anonymous>" in FR10 refers to
// a frame with NO FilePath at all -- it does NOT mean any frame whose
// MethodName happens to be "<anonymous>" (V8's own name for an
// unnamed/inline function, e.g. the very common "Object.<anonymous>"
// description on an ordinary own-code frame, confirmed real via
// testdata/bare-stack.txt's first frame). That frame has a perfectly
// real FilePath and must bucket as BucketOwn like any other own-code
// frame -- MethodName is never consulted here, only FilePath.
func assignBucket(f contract.Frame) contract.Frame {
	f.FilePath = normalizeFileURI(f.FilePath)
	f.PackageName = ""

	if f.FilePath == "" || strings.HasPrefix(f.FilePath, "node:") {
		f.Bucket = contract.BucketRuntime
		return f
	}

	if pkg, isDependency := splitAfterLastNodeModules(f.FilePath); isDependency {
		f.Bucket = contract.BucketDependency
		f.PackageName = pkg
		return f
	}

	// No node_modules segment: either genuine own code, or a
	// bundled/minified single-file build (webpack/esbuild
	// dist/bundle.js) combining own and vendored code with no
	// node_modules segment to detect -- spec.md FR11 explicitly
	// defaults that case to BucketOwn (source-map-aware bucketing is
	// out of scope for v1, a known accepted misclassification risk, not
	// a silent gap -- see memory/known-gaps.md).
	f.Bucket = contract.BucketOwn
	return f
}
