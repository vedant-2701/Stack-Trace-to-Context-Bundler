package typescript

import (
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// TestAssignBucket exercises every spec.md bucketing acceptance
// criterion individually (T008). Inputs are literal FilePath strings
// taken from real fixtures already in testdata/ (cited per case, same
// style as TestParseFrameLine's literal-line table) rather than routed
// through parseChain -- assignBucket operates per-frame, independent of
// how that frame was constructed (plan.md pipeline step 4), so a direct
// table test is the right level for this function, matching
// plan.md's own description of bucket_test.go as "bucketing edge
// cases," not a fixture-loading integration test.
func TestAssignBucket(t *testing.T) {
	tests := []struct {
		name            string
		input           contract.Frame
		wantBucket      contract.Bucket
		wantPackageName string
		wantFilePath    string
	}{
		{
			// testdata/bare-stack.txt's first frame.
			name:         "own code, plain project path",
			input:        contract.Frame{FilePath: "/home/vedant/script.js"},
			wantBucket:   contract.BucketOwn,
			wantFilePath: "/home/vedant/script.js",
		},
		{
			// testdata/nested-deps-flat.txt's first frame.
			name:            "dependency, plain package name",
			input:           contract.Frame{FilePath: "/home/vedant/stack-trace-bundler/errors-test/node_modules/statuses/index.js"},
			wantBucket:      contract.BucketDependency,
			wantPackageName: "statuses",
			wantFilePath:    "/home/vedant/stack-trace-bundler/errors-test/node_modules/statuses/index.js",
		},
		{
			// testdata/scoped-package-babel.txt's 5th frame.
			name:            "dependency, scoped package name",
			input:           contract.Frame{FilePath: "/home/vedant/project/node_modules/@babel/core/lib/index.js"},
			wantBucket:      contract.BucketDependency,
			wantPackageName: "@babel/core",
			wantFilePath:    "/home/vedant/project/node_modules/@babel/core/lib/index.js",
		},
		{
			// testdata/deep-nested-node-modules.txt's 5th frame -- three
			// nested node_modules segments (express, then finalhandler,
			// then statuses); the correct package is the one after the
			// LAST occurrence, "statuses", not "express" or
			// "finalhandler".
			name:            "dependency, nested node_modules uses last occurrence",
			input:           contract.Frame{FilePath: "/home/vedant/project/node_modules/express/node_modules/finalhandler/node_modules/statuses/index.js"},
			wantBucket:      contract.BucketDependency,
			wantPackageName: "statuses",
			wantFilePath:    "/home/vedant/project/node_modules/express/node_modules/finalhandler/node_modules/statuses/index.js",
		},
		{
			// testdata/bare-stack.txt's 4th frame.
			name:         "runtime, node: internal path",
			input:        contract.Frame{FilePath: "node:internal/modules/cjs/loader"},
			wantBucket:   contract.BucketRuntime,
			wantFilePath: "node:internal/modules/cjs/loader",
		},
		{
			// Synthetic -- see assignBucket's doc comment: T002's
			// parseFrameLine can never actually produce a Frame with an
			// empty FilePath (locationPattern requires real digits), so
			// this case can't be reached via a real fixture today. Still
			// required by FR10's literal wording and this task's
			// acceptance criterion, so tested directly against the
			// function rather than skipped.
			name:         "runtime, no file info at all (unreachable via real parse, still spec'd)",
			input:        contract.Frame{FilePath: ""},
			wantBucket:   contract.BucketRuntime,
			wantFilePath: "",
		},
		{
			// testdata/bare-stack.txt's first frame again, but this
			// time asserting the MethodName == "<anonymous>" case does
			// NOT get misbucketed as BucketRuntime -- see assignBucket's
			// doc comment's "IMPORTANT" note. MethodName must never be
			// consulted, only FilePath.
			name: "own code, MethodName is <anonymous> but FilePath is real -- must stay BucketOwn",
			input: contract.Frame{
				FilePath: "/home/vedant/script.js", ClassName: "Object", MethodName: "<anonymous>",
			},
			wantBucket:   contract.BucketOwn,
			wantFilePath: "/home/vedant/script.js",
		},
		{
			// testdata/esbuild-minified-bundle.txt's first frame --
			// bundled/minified single-file output, no node_modules
			// segment present. spec.md FR11: defaults to BucketOwn.
			name:         "bundled/minified output with no node_modules segment defaults to own (FR11)",
			input:        contract.Frame{FilePath: "/home/vedant/stack-trace-bundler/errors-test/dist/bundle.js"},
			wantBucket:   contract.BucketOwn,
			wantFilePath: "/home/vedant/stack-trace-bundler/errors-test/dist/bundle.js",
		},
		{
			// testdata/esm-runtime-error.txt's first frame -- file://
			// URI must be normalized to a real filesystem path (FR12)
			// as well as correctly bucketed (own code, no node_modules
			// segment).
			name:            "file:// URI is normalized before bucketing (FR12)",
			input:           contract.Frame{FilePath: "file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js"},
			wantBucket:      contract.BucketOwn,
			wantPackageName: "",
			wantFilePath:    "/home/vedant/stack-trace-bundler/errors-test/pg-fails.js",
		},
		{
			// Synthetic -- no real Windows-generated fixture exists yet
			// (see memory/known-gaps.md); confirms
			// splitAfterLastNodeModules's backslash normalization is
			// actually wired up. FilePath itself is NOT rewritten to
			// forward slashes -- normalization is local to
			// splitAfterLastNodeModules's own segment-finding, not applied
			// to the Frame -- that's 004's concern (git blame/snippet
			// extraction), not bucketing's, so it stays exactly as given.
			name:            "dependency, Windows-style backslash path (synthetic, unverified against a real trace)",
			input:           contract.Frame{FilePath: `C:\Users\vedant\project\node_modules\lodash\index.js`},
			wantBucket:      contract.BucketDependency,
			wantPackageName: "lodash",
			wantFilePath:    `C:\Users\vedant\project\node_modules\lodash\index.js`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assignBucket(tt.input)
			if got.Bucket != tt.wantBucket {
				t.Errorf("Bucket = %q, want %q", got.Bucket, tt.wantBucket)
			}
			if got.PackageName != tt.wantPackageName {
				t.Errorf("PackageName = %q, want %q", got.PackageName, tt.wantPackageName)
			}
			if got.FilePath != tt.wantFilePath {
				t.Errorf("FilePath = %q, want %q", got.FilePath, tt.wantFilePath)
			}
		})
	}
}
