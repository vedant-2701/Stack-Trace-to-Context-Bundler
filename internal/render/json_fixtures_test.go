package render

import (
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// ampersandBundle is deliberately minimal (single exception node, one
// own-bucket frame, ordinary GitMetadata/Dependencies) -- it isolates
// one concern only: a literal `&` in a Message field, the one
// HTML-escape target character (alongside `<`/`>`, already covered by
// reusing markdownSpecialCharsBundle) that no existing 007 fixture
// contains. Lives in its own file, not 007's fixtures_test.go, since
// this fixture is JSON-renderer-specific.
func ampersandBundle(t *testing.T) contract.Bundle {
	t.Helper()

	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          contract.LanguageTypeScript,
		OS:                contract.OSLinux,
		Fingerprint:       "ff88aa99bb00cc11",
		RawInputTruncated: false,
		RawInput: `Error: fetch failed: config.json & env both missing
    at loadConfig (/repo/src/config.ts:11:6)`,
		Runtime: contract.Runtime{
			Name: "node", Version: "20.11.0", VersionSource: contract.VersionSourceTrace,
		},
		Chain: []contract.ExceptionNode{
			{
				ClassName: "Error",
				Message:   "fetch failed: config.json & env both missing",
				Frames: []contract.Frame{
					{
						Index: 0, FilePath: "/repo/src/config.ts", MethodName: "loadConfig",
						LineNumber: 11, ColumnNumber: 6, Bucket: contract.BucketOwn,
					},
				},
			},
		},
		CodeContexts: []contract.CodeContext{
			{
				FrameRef: contract.FrameRef{ChainIndex: 0, FrameIndex: 0},
				FilePath: "/repo/src/config.ts",
				Language: contract.LanguageTypeScript,
				Status:   contract.StatusOK,
				Snippet: contract.Snippet{
					StartLine: 6, EndLine: 16, TargetLine: 11,
					Code: `import * as fs from 'fs';

export function loadConfig(): Record<string, unknown> {
  const raw = fs.readFileSync('/repo/config.json', 'utf-8');
  const env = process.env.ENV;
  if (!raw || !env) throw new Error('fetch failed: config.json & env both missing');

  return { ...JSON.parse(raw), env };
}

export function reloadConfig(): void {
`,
				},
				Blame: []contract.BlameEntry{
					{
						StartLine: 6, EndLine: 16,
						CommitHash: "0011223344556677889900112233445566778899",
						Author:     "vedant",
						CommitDate: "2026-09-10T09:00:00Z",
						Summary:    "validate env before parsing config",
					},
				},
			},
		},
		GitMetadata: &contract.GitMetadata{
			CurrentCommit:      "0011223344556677889900112233445566778899",
			Branch:             "main",
			UncommittedChanges: false,
		},
		Dependencies: &contract.Dependencies{
			ManifestFile: contract.ManifestFilePackageJSON,
			Direct: map[string]string{
				"lodash": "^4.17.21",
			},
			Locked: map[string]contract.LockedDependency{
				"lodash": {Version: "4.17.21"},
			},
		},
	}
}
