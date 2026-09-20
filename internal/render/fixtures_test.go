package render

import (
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// tsBasicBundle is a hand-authored contract.Bundle literal -- compile-
// checked against types.go, not a JSON round-trip through another
// feature's fixture -- representing a realistic two-exception TS/JS
// chain (a lodash-mediated TypeError caused by a Postgres connection
// Error), matching how Markdown() is actually called in production:
// directly on an in-memory contract.Bundle, never via JSON.
//
// Both own-bucket snippets use the real ±5-line window convention
// (specs/INDEX.md's 011 row: "fixed at ±5/side in 004") -- 11 total
// lines, target line in the middle -- not an arbitrary short window.
func tsBasicBundle(t *testing.T) contract.Bundle {
	t.Helper()

	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          contract.LanguageTypeScript,
		OS:                contract.OSDarwin,
		Fingerprint:       "73be0bb68b5104a7",
		RawInputTruncated: false,
		RawInput: `TypeError: Cannot read properties of undefined (reading 'id')
    at handleRequest (/repo/src/handler.ts:27:14)
    at Object.get (/repo/node_modules/lodash/lodash.js:11812:3)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
Caused by: Error: ECONNREFUSED
    at queryDatabase (/repo/src/service.ts:63:9)
Node.js v20.11.0`,
		Runtime: contract.Runtime{
			Name: "node", Version: "20.11.0", VersionSource: contract.VersionSourceTrace,
		},
		Chain: []contract.ExceptionNode{
			{
				ClassName: "TypeError",
				Message:   "Cannot read properties of undefined (reading 'id')",
				Frames: []contract.Frame{
					{
						Index: 0, FilePath: "/repo/src/handler.ts", MethodName: "handleRequest",
						LineNumber: 27, ColumnNumber: 14, Bucket: contract.BucketOwn,
					},
					{
						Index: 1, FilePath: "/repo/node_modules/lodash/lodash.js", MethodName: "get",
						LineNumber: 11812, ColumnNumber: 3, Bucket: contract.BucketDependency, PackageName: "lodash",
					},
					{
						Index: 2, FilePath: "node:internal/process/task_queues", MethodName: "processTicksAndRejections",
						LineNumber: 95, ColumnNumber: 5, Bucket: contract.BucketRuntime,
					},
				},
			},
			{
				ClassName: "Error",
				Message:   "ECONNREFUSED",
				Frames: []contract.Frame{
					{
						Index: 0, FilePath: "/repo/src/service.ts", MethodName: "queryDatabase",
						LineNumber: 63, ColumnNumber: 9, Bucket: contract.BucketOwn,
					},
				},
			},
		},
		CodeContexts: []contract.CodeContext{
			{
				FrameRef: contract.FrameRef{ChainIndex: 0, FrameIndex: 0},
				FilePath: "/repo/src/handler.ts",
				Language: contract.LanguageTypeScript,
				Status:   contract.StatusOK,
				Snippet: contract.Snippet{
					StartLine: 22, EndLine: 32, TargetLine: 27,
					Code: `import { Request, Response } from 'express';
import * as service from './service';

export function handleRequest(req: Request) {
  const payload = req.body;
  return service.queryDatabase(payload.id);
}

export function handleError(err: Error) {
  console.error(err);
}
`,
				},
				Blame: []contract.BlameEntry{
					{
						StartLine: 22, EndLine: 32,
						CommitHash: "89abcdef0123456789abcdef0123456789abcdef",
						Author:     "vedant",
						CommitDate: "2026-07-29T10:15:00Z",
						Summary:    "validate request payload before dispatch",
					},
				},
			},
			{
				FrameRef: contract.FrameRef{ChainIndex: 1, FrameIndex: 0},
				FilePath: "/repo/src/service.ts",
				Language: contract.LanguageTypeScript,
				Status:   contract.StatusOK,
				Snippet: contract.Snippet{
					StartLine: 58, EndLine: 68, TargetLine: 63,
					Code: `import { Pool } from 'pg';

const pool = new Pool();

export async function queryDatabase(id: string) {
  const conn = await pool.connect();
  return conn.query(id);
}

export function closePool(): void {
  pool.end();
`,
				},
				Blame: []contract.BlameEntry{
					{
						StartLine: 58, EndLine: 68,
						CommitHash: "76543210fedcba9876543210fedcba9876543210",
						Author:     "vedant",
						CommitDate: "2026-08-01T09:00:00Z",
						Summary:    "add connection pooling for query path",
					},
				},
			},
		},
		GitMetadata: &contract.GitMetadata{
			CurrentCommit:      "76543210fedcba9876543210fedcba9876543210",
			Branch:             "feature/query-fix",
			UncommittedChanges: true,
		},
		Dependencies: &contract.Dependencies{
			ManifestFile: contract.ManifestFilePackageJSON,
			Direct: map[string]string{
				"express": "^4.18.2",
				"lodash":  "^4.17.21",
			},
			Locked: map[string]contract.LockedDependency{
				"lodash": {Version: "4.17.21"},
				"express": {
					Note: "no package-lock.json entry found for this package",
				},
			},
		},
	}
}

// noGitMetadataBundle is deliberately minimal (single exception node,
// one own-bucket frame, one runtime frame) -- it exists to isolate one
// concern only: GitMetadata == nil must omit the "Git: ..." metadata
// line entirely, not render it empty. Dependencies is present (normal)
// so this fixture doesn't also exercise T012's other concern.
func noGitMetadataBundle(t *testing.T) contract.Bundle {
	t.Helper()

	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          contract.LanguageTypeScript,
		OS:                contract.OSLinux,
		Fingerprint:       "aa11bb22cc33dd44",
		RawInputTruncated: false,
		RawInput: `RangeError: Invalid array length
    at buildMatrix (/repo/src/matrix.ts:14:10)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)`,
		Runtime: contract.Runtime{
			Name: "node", Version: "20.11.0", VersionSource: contract.VersionSourceTrace,
		},
		Chain: []contract.ExceptionNode{
			{
				ClassName: "RangeError",
				Message:   "Invalid array length",
				Frames: []contract.Frame{
					{
						Index: 0, FilePath: "/repo/src/matrix.ts", MethodName: "buildMatrix",
						LineNumber: 14, ColumnNumber: 10, Bucket: contract.BucketOwn,
					},
					{
						Index: 1, FilePath: "node:internal/process/task_queues", MethodName: "processTicksAndRejections",
						LineNumber: 95, ColumnNumber: 5, Bucket: contract.BucketRuntime,
					},
				},
			},
		},
		CodeContexts: []contract.CodeContext{
			{
				FrameRef: contract.FrameRef{ChainIndex: 0, FrameIndex: 0},
				FilePath: "/repo/src/matrix.ts",
				Language: contract.LanguageTypeScript,
				Status:   contract.StatusOK,
				Snippet: contract.Snippet{
					StartLine: 9, EndLine: 19, TargetLine: 14,
					Code: `export function buildMatrix(rows: number, cols: number) {
  if (rows < 0 || cols < 0) {
    throw new RangeError('Invalid array length');
  }

  const matrix = new Array(rows * cols);
  for (let i = 0; i < matrix.length; i++) {
    matrix[i] = 0;
  }
  return matrix;
}
`,
				},
				Blame: []contract.BlameEntry{
					{
						StartLine: 9, EndLine: 19,
						CommitHash: "11223344556677889900aabbccddeeff0011223",
						Author:     "vedant",
						CommitDate: "2026-06-10T12:00:00Z",
						Summary:    "add matrix builder utility",
					},
				},
			},
		},
		GitMetadata: nil,
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

// noDependenciesBundle is deliberately minimal, same reasoning as
// noGitMetadataBundle -- it isolates the other T012 concern:
// Dependencies == nil must omit the whole "## Dependencies" heading, not
// render it with no content. GitMetadata is present (normal) so this
// fixture doesn't also exercise T012's other concern.
func noDependenciesBundle(t *testing.T) contract.Bundle {
	t.Helper()

	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          contract.LanguageJava,
		OS:                contract.OSLinux,
		Fingerprint:       "ee55ff66aa77bb88",
		RawInputTruncated: false,
		RawInput: `java.lang.NullPointerException: Cannot invoke "String.length()" because "name" is null
	at com.example.Greeter.greet(Greeter.java:18)
	at java.base/java.lang.Thread.run(Thread.java:840)`,
		Runtime: contract.Runtime{
			Name: "jvm", Version: "17.0.9", VersionSource: contract.VersionSourceLocalEnvironment,
			Note: "inferred from local `java -version`",
		},
		Chain: []contract.ExceptionNode{
			{
				ClassName: "java.lang.NullPointerException",
				Message:   `Cannot invoke "String.length()" because "name" is null`,
				Frames: []contract.Frame{
					{
						Index: 0, FilePath: "src/main/java/com/example/Greeter.java",
						ClassName: "Greeter", MethodName: "greet",
						LineNumber: 18, Bucket: contract.BucketOwn,
					},
					{
						Index: 1, FilePath: "java.base/java.lang.Thread", MethodName: "run",
						LineNumber: 840, Bucket: contract.BucketRuntime,
					},
				},
			},
		},
		CodeContexts: []contract.CodeContext{
			{
				FrameRef: contract.FrameRef{ChainIndex: 0, FrameIndex: 0},
				FilePath: "src/main/java/com/example/Greeter.java",
				Language: contract.LanguageJava,
				Status:   contract.StatusOK,
				Snippet: contract.Snippet{
					StartLine: 13, EndLine: 23, TargetLine: 18,
					Code: `public class Greeter {

  private final String prefix;

  public String greet(String name) {
    return prefix + name.length();
  }

  public Greeter(String prefix) {
    this.prefix = prefix;
  }
`,
				},
				Blame: []contract.BlameEntry{
					{
						StartLine: 13, EndLine: 23,
						CommitHash: "aabbccddeeff00112233445566778899aabbccd",
						Author:     "vedant",
						CommitDate: "2026-05-02T08:30:00Z",
						Summary:    "add greeter service",
					},
				},
			},
		},
		GitMetadata: &contract.GitMetadata{
			CurrentCommit:      "aabbccddeeff00112233445566778899aabbccd",
			Branch:             "main",
			UncommittedChanges: false,
		},
		Dependencies: nil,
	}
}
