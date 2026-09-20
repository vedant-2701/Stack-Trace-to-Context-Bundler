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

// elidedFramesBundle isolates spec.md req. 14: a node's
// ElidedFrameCount > 0 renders a single "... N more frames (shared
// with enclosing exception)" line immediately after that node's frame
// list. Only the second (Caused-by) node carries a nonzero count here,
// so this also implicitly confirms the first node's ElidedFrameCount ==
// 0 renders no such line.
func elidedFramesBundle(t *testing.T) contract.Bundle {
	t.Helper()

	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          contract.LanguageTypeScript,
		OS:                contract.OSLinux,
		Fingerprint:       "aa22bb33cc44dd55",
		RawInputTruncated: false,
		RawInput: `RuntimeException: top-level failure
    at run (/repo/src/main.ts:10:5)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)
Caused by: IOException: disk read failed
    at readFile (/repo/src/io.ts:22:8)
    ... 6 more frames (shared with enclosing exception)`,
		Runtime: contract.Runtime{
			Name: "node", Version: "20.11.0", VersionSource: contract.VersionSourceTrace,
		},
		Chain: []contract.ExceptionNode{
			{
				ClassName: "RuntimeException",
				Message:   "top-level failure",
				Frames: []contract.Frame{
					{
						Index: 0, FilePath: "/repo/src/main.ts", MethodName: "run",
						LineNumber: 10, ColumnNumber: 5, Bucket: contract.BucketOwn,
					},
					{
						Index: 1, FilePath: "node:internal/process/task_queues", MethodName: "processTicksAndRejections",
						LineNumber: 95, ColumnNumber: 5, Bucket: contract.BucketRuntime,
					},
				},
			},
			{
				ClassName:        "IOException",
				Message:          "disk read failed",
				ElidedFrameCount: 6,
				Frames: []contract.Frame{
					{
						Index: 0, FilePath: "/repo/src/io.ts", MethodName: "readFile",
						LineNumber: 22, ColumnNumber: 8, Bucket: contract.BucketOwn,
					},
				},
			},
		},
		CodeContexts: []contract.CodeContext{
			{
				FrameRef: contract.FrameRef{ChainIndex: 0, FrameIndex: 0},
				FilePath: "/repo/src/main.ts",
				Language: contract.LanguageTypeScript,
				Status:   contract.StatusOK,
				Snippet: contract.Snippet{
					StartLine: 5, EndLine: 15, TargetLine: 10,
					Code: `import { readFile } from './io';

export async function run() {
  try {
    const data = await readFile('/tmp/input.txt');
    console.log(data);
  } catch (err) {
    throw err;
  }
}

`,
				},
				Blame: []contract.BlameEntry{
					{
						StartLine: 5, EndLine: 15,
						CommitHash: "2233445566778899001122334455667788990011",
						Author:     "vedant",
						CommitDate: "2026-04-11T14:20:00Z",
						Summary:    "wire up main entrypoint",
					},
				},
			},
			{
				FrameRef: contract.FrameRef{ChainIndex: 1, FrameIndex: 0},
				FilePath: "/repo/src/io.ts",
				Language: contract.LanguageTypeScript,
				Status:   contract.StatusOK,
				Snippet: contract.Snippet{
					StartLine: 17, EndLine: 27, TargetLine: 22,
					Code: `import * as fs from 'fs/promises';

export async function readFile(path: string): Promise<string> {
  const buffer = await fs.readFile(path);

  return buffer.toString('utf-8');
}

export async function writeFile(path: string, data: string) {
  await fs.writeFile(path, data);
}
`,
				},
				Blame: []contract.BlameEntry{
					{
						StartLine: 17, EndLine: 27,
						CommitHash: "3344556677889900112233445566778899001122",
						Author:     "vedant",
						CommitDate: "2026-04-15T09:05:00Z",
						Summary:    "add file IO helpers",
					},
				},
			},
		},
		GitMetadata: &contract.GitMetadata{
			CurrentCommit:      "3344556677889900112233445566778899001122",
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

// multilineMessageBundle isolates spec.md req. 6-7 end-to-end (T007
// already unit-tests renderChain's blockquote logic directly; this
// proves it also survives a full Markdown() render): Message contains a
// wholly empty line in its middle, which must render as a bare ">"
// rather than an actual blank line, keeping the blockquote unbroken.
func multilineMessageBundle(t *testing.T) contract.Bundle {
	t.Helper()

	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          contract.LanguageTypeScript,
		OS:                contract.OSLinux,
		Fingerprint:       "bb33cc44dd55ee66",
		RawInputTruncated: false,
		RawInput: `ValidationError: Expected values to be strictly equal:

foo !== bar
    at assertEqual (/repo/src/assert.ts:6:7)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)`,
		Runtime: contract.Runtime{
			Name: "node", Version: "20.11.0", VersionSource: contract.VersionSourceTrace,
		},
		Chain: []contract.ExceptionNode{
			{
				ClassName: "ValidationError",
				Message:   "Expected values to be strictly equal:\n\nfoo !== bar",
				Frames: []contract.Frame{
					{
						Index: 0, FilePath: "/repo/src/assert.ts", MethodName: "assertEqual",
						LineNumber: 6, ColumnNumber: 7, Bucket: contract.BucketOwn,
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
				FilePath: "/repo/src/assert.ts",
				Language: contract.LanguageTypeScript,
				Status:   contract.StatusOK,
				Snippet: contract.Snippet{
					StartLine: 3, EndLine: 13, TargetLine: 6,
					Code: `export function assertEqual(a: unknown, b: unknown) {
  if (a !== b) {
    throw new ValidationError(
      'Expected values to be strictly equal: ' + a + ' !== ' + b
    );
  }
}

export function assertNotEqual(a: unknown, b: unknown) {
  if (a === b) {
    throw new ValidationError('Expected values to differ');
`,
				},
				Blame: []contract.BlameEntry{
					{
						StartLine: 3, EndLine: 13,
						CommitHash: "4455667788990011223344556677889900112233",
						Author:     "vedant",
						CommitDate: "2026-03-20T16:40:00Z",
						Summary:    "add assertion helpers",
					},
				},
			},
		},
		GitMetadata: &contract.GitMetadata{
			CurrentCommit:      "4455667788990011223344556677889900112233",
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

// codeContextNotFoundBundle is deliberately minimal, isolating one
// CodeContext outcome (spec.md req. 10): Status == not_found renders
// only a flagged "⚠ <Note>" line -- no snippet, no blame table --
// regardless of what Snippet/Blame happen to hold (left zero-value
// here to make that explicit). GitMetadata/Dependencies are present and
// ordinary so neither is also under test in this fixture.
func codeContextNotFoundBundle(t *testing.T) contract.Bundle {
	t.Helper()

	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          contract.LanguageTypeScript,
		OS:                contract.OSLinux,
		Fingerprint:       "cc11dd22ee33ff44",
		RawInputTruncated: false,
		RawInput: `Error: legacy path removed
    at parseLegacy (/repo/src/legacy/oldParser.ts:5:3)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)`,
		Runtime: contract.Runtime{
			Name: "node", Version: "20.11.0", VersionSource: contract.VersionSourceTrace,
		},
		Chain: []contract.ExceptionNode{
			{
				ClassName: "Error",
				Message:   "legacy path removed",
				Frames: []contract.Frame{
					{
						Index: 0, FilePath: "/repo/src/legacy/oldParser.ts", MethodName: "parseLegacy",
						LineNumber: 5, ColumnNumber: 3, Bucket: contract.BucketOwn,
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
				FilePath: "/repo/src/legacy/oldParser.ts",
				Language: contract.LanguageTypeScript,
				Status:   contract.StatusNotFound,
				Note:     "file not found in current checkout (deleted or renamed since the trace was captured)",
			},
		},
		GitMetadata: &contract.GitMetadata{
			CurrentCommit:      "1234567890abcdef1234567890abcdef12345678",
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

// codeContextStaleBundle mirrors codeContextNotFoundBundle exactly,
// isolating the other non-rendering CodeContext outcome: Status ==
// stale behaves identically to not_found -- flagged Note line only.
func codeContextStaleBundle(t *testing.T) contract.Bundle {
	t.Helper()

	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          contract.LanguageTypeScript,
		OS:                contract.OSLinux,
		Fingerprint:       "dd22ee33ff44aa11",
		RawInputTruncated: false,
		RawInput: `Error: stale checkout mismatch
    at formatDate (/repo/src/utils/formatDate.ts:8:5)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)`,
		Runtime: contract.Runtime{
			Name: "node", Version: "20.11.0", VersionSource: contract.VersionSourceTrace,
		},
		Chain: []contract.ExceptionNode{
			{
				ClassName: "Error",
				Message:   "stale checkout mismatch",
				Frames: []contract.Frame{
					{
						Index: 0, FilePath: "/repo/src/utils/formatDate.ts", MethodName: "formatDate",
						LineNumber: 8, ColumnNumber: 5, Bucket: contract.BucketOwn,
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
				FilePath: "/repo/src/utils/formatDate.ts",
				Language: contract.LanguageTypeScript,
				Status:   contract.StatusStale,
				Note:     "file has uncommitted local changes; snippet may not match the trace",
			},
		},
		GitMetadata: &contract.GitMetadata{
			CurrentCommit:      "1234567890abcdef1234567890abcdef12345678",
			Branch:             "main",
			UncommittedChanges: true,
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

// codeContextOKNoBlameBundle isolates the third non-table CodeContext
// outcome: Status == ok with Blame empty still renders the snippet, but
// the blame table is replaced by a flagged "⚠ <Note>" line explaining
// why (e.g. no git repository found).
func codeContextOKNoBlameBundle(t *testing.T) contract.Bundle {
	t.Helper()

	return contract.Bundle{
		SchemaVersion:     contract.SchemaVersion,
		Language:          contract.LanguageTypeScript,
		OS:                contract.OSLinux,
		Fingerprint:       "ee33ff44aa11bb22",
		RawInputTruncated: false,
		RawInput: `Error: missing repo context
    at slugify (/repo/src/utils/slugify.ts:10:3)
    at processTicksAndRejections (node:internal/process/task_queues:95:5)`,
		Runtime: contract.Runtime{
			Name: "node", Version: "20.11.0", VersionSource: contract.VersionSourceTrace,
		},
		Chain: []contract.ExceptionNode{
			{
				ClassName: "Error",
				Message:   "missing repo context",
				Frames: []contract.Frame{
					{
						Index: 0, FilePath: "/repo/src/utils/slugify.ts", MethodName: "slugify",
						LineNumber: 10, ColumnNumber: 3, Bucket: contract.BucketOwn,
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
				FilePath: "/repo/src/utils/slugify.ts",
				Language: contract.LanguageTypeScript,
				Status:   contract.StatusOK,
				Snippet: contract.Snippet{
					StartLine: 5, EndLine: 15, TargetLine: 10,
					Code: `export function slugify(input: string): string {
  const trimmed = input.trim();
  const lower = trimmed.toLowerCase();

  return lower
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

export function unslugify(slug: string): string {
  return slug.replace(/-/g, ' ');
`,
				},
				Blame: nil,
				Note:  "no git repository found at this path",
			},
		},
		GitMetadata: &contract.GitMetadata{
			CurrentCommit:      "1234567890abcdef1234567890abcdef12345678",
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
