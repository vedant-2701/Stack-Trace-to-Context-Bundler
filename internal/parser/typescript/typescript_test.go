package typescript

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFixturesLoad is a T001 placeholder: it only confirms every fixture
// referenced in testdata/README.md is present on disk and non-empty. It
// does not parse or assert on trace content -- that begins in T002+ once
// engine.go's frame-line matcher exists. Table entries mirror
// testdata/README.md's two tables (real captures, then synthetic) so the
// two stay in sync; a fixture added to one without the other should fail
// review, not just this test.
func TestFixturesLoad(t *testing.T) {
	fixtures := []string{
		// Real captures
		"crash-with-cause.txt",
		"crash-typeerror-no-cause.txt",
		"logged-object-with-cause.txt",
		"bare-stack.txt",
		"crash-3level-cause.txt",
		"crash-async-rejection-preamble.txt",
		"logged-object-fetch-cause.txt",
		"bare-stack-fetch-cause.txt",
		"nested-deps-flat.txt",
		"zero-stack-trace-limit.txt",
		"esm-runtime-error.txt",
		"import-outside-module.txt",
		"ts-native-execution.txt",
		"ts-compiled-then-run.txt",
		"ts-tsx-transformer-path.txt",
		"esbuild-minified-bundle.txt",
		"scoped-package-swc-false-caused-by.txt",
		"aggregate-error-uncaught.txt",
		"assert-multiline-diff.txt",
		// Synthetic (no real capture exists for these shapes)
		"scoped-package-babel.txt",
		"deep-nested-node-modules.txt",
		"browser-trace-false-positive.txt",
		"truncated-mid-frame.txt",
		"cutoff-cause-chain.txt",
		"unparseable-input.txt",
	}

	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("testdata", name)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", path, err)
			}
			if len(content) == 0 {
				t.Fatalf("%s is empty", path)
			}
		})
	}
}
