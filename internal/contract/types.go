// Package contract defines the canonical shape of a stack-trace-bundler
// bundle. internal/contract is the single source of truth for this shape
// (constitution Article IV): there is no hand-maintained JSON Schema and no
// second copy of it anywhere else in the repo. Parsers (005a Java, 006a
// TS/JS) populate these structs; renderers (007 Markdown, 008 JSON) read
// them. Neither defines or modifies the shape.
//
// Cross-cutting rule, applies throughout this file: a field that doesn't
// apply to a given language or case is omitted from the JSON entirely
// (`omitempty`) -- never `null`, never a zero value standing in for "not
// applicable." Zero/false values are reserved for cases where they're a
// real, meaningful answer (e.g. ElidedFrameCount 0, or RawInputTruncated
// false, which is intentionally NOT omitempty since it's a real,
// always-relevant status rather than a not-applicable case). Ambiguity
// here is exactly what constitution Article VI ("never guess silently")
// is meant to prevent, just at the serialization layer instead of the
// data-collection layer.
package contract

// SchemaVersion is the current version of the Bundle shape, following
// semver. Bump triggers: MAJOR for any breaking change (a field renamed,
// removed, or its type changed; an existing enum/status value's meaning
// changed); MINOR for additive-only changes (a new optional field); no
// bump for implementation changes that don't touch the shape at all.
const SchemaVersion = "2.0.0"

// Language identifies the source language of a stack trace. Closed for
// v1: matches the two parser features in specs/INDEX.md (005a Java, 006a
// TS/JS). No other languages in scope.
type Language string

const (
	// LanguageJavaScript is used for plain JavaScript stack traces.
	LanguageJavaScript Language = "javascript"
	// LanguageTypeScript is used for TypeScript stack traces.
	LanguageTypeScript Language = "typescript"
	// LanguageJava is used for Java stack traces.
	LanguageJava Language = "java"
)

// OS identifies the operating system running stack-trace-bundler,
// populated via Go's runtime.GOOS. Values MUST use Go's own constants,
// not another ecosystem's platform-naming convention (e.g. not JS's
// "win32").
type OS string

const (
	// OSLinux corresponds to Go's runtime.GOOS == "linux".
	OSLinux OS = "linux"
	// OSDarwin corresponds to Go's runtime.GOOS == "darwin".
	OSDarwin OS = "darwin"
	// OSWindows corresponds to Go's runtime.GOOS == "windows" -- NOT
	// "win32", which is JS's process.platform convention, not Go's.
	OSWindows OS = "windows"
)

// Bucket classifies a Frame's origin. Values are fixed here; the logic
// that decides which bucket a frame falls into belongs to the parsers
// (005a/006a), not this package.
type Bucket string

const (
	// BucketOwn marks a frame as belonging to the user's own code.
	BucketOwn Bucket = "own"
	// BucketDependency marks a frame as belonging to a third-party
	// dependency.
	BucketDependency Bucket = "dependency"
	// BucketRuntime marks a frame as belonging to the language runtime
	// itself.
	BucketRuntime Bucket = "runtime"
)

// CodeContextStatus describes whether a CodeContext's snippet can be
// trusted.
type CodeContextStatus string

const (
	// StatusOK means neither StatusNotFound nor StatusStale applies.
	StatusOK CodeContextStatus = "ok"
	// StatusNotFound means the referenced file doesn't exist in the
	// current checkout.
	StatusNotFound CodeContextStatus = "not_found"
	// StatusStale means that specific file has uncommitted local
	// changes -- a per-file check, distinct from the repo-wide
	// GitMetadata.UncommittedChanges flag.
	StatusStale CodeContextStatus = "stale"
)

// VersionSource explains how Runtime.Version was determined.
type VersionSource string

const (
	// VersionSourceTrace means the version was actually present in the
	// parsed trace text. Confirmed narrow case: Node's own
	// uncaught-exception crash output, which appends a trailing
	// "Node.js vX.Y.Z" line -- this does not fire when the exception is
	// caught and logged by user code, the common case. Java's
	// printStackTrace() never includes JVM version, so Java is always
	// VersionSourceLocalEnvironment or VersionSourceUnknown.
	VersionSourceTrace VersionSource = "trace"
	// VersionSourceLocalEnvironment means the version was inferred by
	// shelling out on the machine running stack-trace-bundler -- it may
	// not match the environment that actually produced the trace.
	VersionSourceLocalEnvironment VersionSource = "local-environment"
	// VersionSourceUnknown means the version could not be determined at
	// all. A real, meaningful value -- not an omission.
	VersionSourceUnknown VersionSource = "unknown"
)

// ManifestFile identifies which manifest format Dependencies was parsed
// from. Single manifest per bundle -- no monorepo/workspace support in
// v1, an accepted simplification, not a hidden gap.
type ManifestFile string

const (
	// ManifestFilePackageJSON is used for npm/yarn/pnpm projects.
	ManifestFilePackageJSON ManifestFile = "package.json"
	// ManifestFilePomXML is used for Maven projects.
	ManifestFilePomXML ManifestFile = "pom.xml"
	// ManifestFileBuildGradle is used for Gradle projects.
	ManifestFileBuildGradle ManifestFile = "build.gradle"
)

// Bundle is the full, canonical shape of a stack-trace-bundler bundle --
// the single source of truth every parser writes to and every renderer
// reads from.
type Bundle struct {
	// SchemaVersion follows semver, starting at "1.0.0". Bump triggers:
	// MAJOR for any breaking change (a field renamed, removed, or its
	// type changed; an existing enum/status value's meaning changed);
	// MINOR for additive-only changes (a new optional field); no bump
	// for implementation changes that don't touch the shape at all.
	SchemaVersion string `json:"schemaVersion"`

	Language Language `json:"language"`
	OS       OS       `json:"os"`

	// RawInput is the verbatim pasted trace -- parse fallback only, not
	// the primary payload (Chain is that). Capped at 512 KB: generous
	// enough that no real stack trace, even at pathological recursion
	// depth, gets truncated, while still catching "wrong thing pasted"
	// (e.g. an entire log file pasted by mistake). See
	// TruncateRawInput in rawinput.go.
	RawInput string `json:"rawInput"`

	// RawInputTruncated is always present, true or false -- NOT
	// omitempty, unlike most fields in this file. It's a real,
	// always-relevant status (was the cap hit or not), so an explicit
	// false is the correct signal here, not an omission that could be
	// misread as "wasn't checked."
	RawInputTruncated bool `json:"rawInputTruncated"`

	// Fingerprint is computed over EVERY exception node in Chain (not
	// only the outermost): the file+method identity (excluding line
	// numbers) of every own-bucket frame in that node, plus the single
	// frame where that node's exception originates, regardless of that
	// frame's bucket. Frames beyond that one originating frame in a
	// dependency/runtime bucket are excluded, so a library version bump
	// alone does not change the fingerprint of an unchanged bug. See
	// ComputeFingerprint in fingerprint.go.
	Fingerprint string `json:"fingerprint"`

	Runtime Runtime `json:"runtime"`

	// Chain is a strictly linear sequence of exception nodes, outermost
	// to root cause. Branching chains (AggregateError/Suppressed) are
	// out of scope for v1.
	Chain []ExceptionNode `json:"chain"`

	// CodeContexts exists only for own-bucket frames.
	CodeContexts []CodeContext `json:"codeContexts"`

	// GitMetadata is nil and omitted from the JSON entirely when no git
	// repository is found for this bundle (004-own-code-context-extraction's
	// scope). Go's omitempty has no effect on a non-pointer struct field,
	// so the pointer is required, not optional, to make omission work.
	// When non-nil, all three inner fields are always populated.
	GitMetadata *GitMetadata `json:"gitMetadata,omitempty"`

	Dependencies Dependencies `json:"dependencies"`
}

// Runtime describes the execution engine (node/bun/deno/jvm), which is
// orthogonal to Language -- e.g. TypeScript can run on node, bun, or
// deno alike.
type Runtime struct {
	// Name is NOT "java": the source language is already captured
	// separately by Bundle.Language. Runtime is the execution engine.
	// Expected values: "node" | "bun" | "deno" | "jvm".
	Name string `json:"name"`

	// Version is omitted entirely (omitempty) if unknown, per Article VI
	// and the cross-cutting omission rule -- not an empty string.
	Version string `json:"version,omitempty"`

	// VersionSource is always present, including VersionSourceUnknown --
	// NOT omitempty. "Unknown" is a real, meaningful enum value here,
	// not the absence of one; that's exactly why it's listed as one of
	// three valid values rather than being represented by omitting the
	// field.
	VersionSource VersionSource `json:"versionSource"`

	// Note explains a non-VersionSourceTrace VersionSource, e.g.
	// "inferred from local `node -v`; may not match the environment
	// that produced this trace." Omitted when VersionSource is
	// VersionSourceTrace.
	Note string `json:"note,omitempty"`
}

// ExceptionNode is one exception in the cause chain.
type ExceptionNode struct {
	ClassName string `json:"className"`
	Message   string `json:"message"`

	// ElidedFrameCount is Java's "... N more" line, which elides frames
	// this node shares with its enclosing exception. Omitted/0 for
	// JS/TS -- V8 doesn't elide shared frames the same way.
	ElidedFrameCount int `json:"elidedFrameCount,omitempty"`

	Frames []Frame `json:"frames"`
}

// Frame is a single stack frame within an ExceptionNode.
type Frame struct {
	Index int `json:"index"`

	// FilePath is a normalized absolute filesystem path, never a URI --
	// even where the source ecosystem natively emits one (e.g. Deno's
	// file:// URLs), since CodeContext needs a real path for git blame
	// and snippet extraction, so the URL form would just get parsed
	// back into a path everywhere it's consumed.
	FilePath string `json:"filePath"`

	// ClassName is absent for e.g. a bare JS function not attached to a
	// class.
	ClassName string `json:"className,omitempty"`

	MethodName string `json:"methodName"`
	LineNumber int    `json:"lineNumber"`

	// ColumnNumber is JS/TS only -- omitted entirely for Java
	// (omitempty), since Java stack traces never carry column info and
	// there is no real zero value here.
	ColumnNumber int `json:"columnNumber,omitempty"`

	Bucket Bucket `json:"bucket"`

	// PackageName is set only when Bucket == BucketDependency. Same
	// field, both languages -- not split per-language, since it's one
	// string whose convention differs by ecosystem, same pattern as
	// ElidedFrameCount/ColumnNumber above. For Java: "group:artifact"
	// (Maven/Gradle coordinate), since artifact name alone doesn't
	// disambiguate libraries that share a name under different groups;
	// must match whatever key format Dependencies.Direct/.Locked use
	// for Java entries.
	PackageName string `json:"packageName,omitempty"`
}

// FrameRef points from a CodeContext back to the Frame it describes.
type FrameRef struct {
	ChainIndex int `json:"chainIndex"`
	FrameIndex int `json:"frameIndex"`
}

// Snippet is the extracted own-code window around a frame's line.
type Snippet struct {
	StartLine  int    `json:"startLine"`
	EndLine    int    `json:"endLine"`
	TargetLine int    `json:"targetLine"`
	Code       string `json:"code"`
}

// CodeContext holds the own-code file snippet and git blame for one
// own-bucket frame. Exists only for own-bucket frames.
type CodeContext struct {
	FrameRef FrameRef `json:"frameRef"`

	// FilePath: same normalized absolute path as Frame.FilePath.
	FilePath string `json:"filePath"`

	Language Language `json:"language"`

	Status CodeContextStatus `json:"status"`

	// Note is optional, e.g. "file not found in current checkout --
	// trace may be from a different commit."
	Note string `json:"note,omitempty"`

	Snippet Snippet `json:"snippet"`

	// Blame lives here, not on GitMetadata: git blame is inherently
	// file+line scoped, not repo-level, so it belongs with the FilePath
	// and Snippet this CodeContext already has, not in a single
	// undifferentiated array on GitMetadata with no way to tell which
	// file an entry belongs to. Present only when Status == StatusOK --
	// nothing to blame for a file that doesn't exist or wasn't
	// extracted.
	Blame []BlameEntry `json:"blame,omitempty"`
}

// BlameEntry covers a contiguous range of lines within a CodeContext's
// Snippet window that share the same last-touching commit -- matches how
// `git blame -L` actually groups output, not one entry per line.
type BlameEntry struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`

	CommitHash string `json:"commitHash"`

	// Author is from `git blame --porcelain`'s "author" field.
	Author string `json:"author"`

	// CommitDate is pre-formatted ISO 8601, derived once at parse time
	// from porcelain's author-time (unix epoch) + author-tz -- not
	// stored as a raw epoch, so both renderers (007 Markdown, 008 JSON)
	// read the same value instead of each formatting it independently
	// (constitution Article V).
	CommitDate string `json:"commitDate"`

	// Summary is the first line of the commit message.
	Summary string `json:"summary"`
}

// GitMetadata holds genuinely repo-level facts -- distinct from the
// per-file Blame data on CodeContext.
type GitMetadata struct {
	CurrentCommit string `json:"currentCommit"`
	Branch        string `json:"branch"`

	// UncommittedChanges is repo-wide. Distinct from
	// CodeContext.Status == StatusStale, which is a per-file check --
	// this one says nothing about which files.
	UncommittedChanges bool `json:"uncommittedChanges"`
}

// Dependencies describes the manifest and the declared/resolved versions
// for dependency-bucket frames.
type Dependencies struct {
	ManifestFile ManifestFile `json:"manifestFile"`

	// Direct maps package name to declared version range, as written
	// directly in the manifest (e.g. "^18.2.0") -- always available
	// from parsing the manifest text alone, no external resolution
	// step, so no unresolved state is possible here (unlike Locked).
	Direct map[string]string `json:"direct"`

	// Locked maps package name to its resolved version, or an
	// explanation of why it couldn't be resolved. Value is an object,
	// not a bare string, because resolution requires actually invoking
	// mvn/gradle/npm and querying a local cache, which can fail
	// per-package rather than all-or-nothing -- same pattern as
	// Runtime.VersionSource/.Note and CodeContext.Status/.Note. Shape
	// stays generic to whichever lockfile/resolution mechanism produced
	// the value (npm's package-lock.json, yarn.lock, pnpm-lock.yaml,
	// Maven/Gradle's resolved graph) -- which lockfile formats 006b
	// actually parses is that feature's scope, not this package's.
	Locked map[string]LockedDependency `json:"locked"`
}

// LockedDependency is one package's resolved version, or an explanation
// of why it couldn't be resolved.
type LockedDependency struct {
	// Version is omitted entirely (omitempty) when unresolved, per the
	// cross-cutting omission rule -- not an empty string.
	Version string `json:"version,omitempty"`

	// Note is present only when Version is absent. E.g. "no local
	// mvn/gradle cache on this checkout (constitution Article IX,
	// decision 0001) -- expected on a fresh clone that hasn't been
	// built locally yet, not a bug."
	Note string `json:"note,omitempty"`
}
