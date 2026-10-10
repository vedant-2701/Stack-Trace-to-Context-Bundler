# Interrogation Log: Pipeline wiring (002b)

Question-and-answer record from spec interrogation, kept separately from
`spec.md` so a long chat session doesn't lose this context. Not a
replacement for `progress.md` (which logs task completion and references
this file) or `spec.md` (which states the resolved requirements) -- this
is the raw negotiation trail behind them.

---

**Q1 — What should Java do in this feature, given 005a/005b don't exist?**
Considered: (a) wire TS/JS only and leave `cmd/java`'s 002a stub
(logs at Info, nothing on stdout, exits 0 -- silent success on a Java
trace); (b) build the pipeline generically, parser list and per-language
dependency resolver supplied by each `main.go`, `cmd/java` fails fast;
(c) hardcode TS/JS inside `internal/cli`.
Risk checked on (b): Article VIII (speculative generality). Judged not to
apply, since three binaries with different parser sets exist today and
TS/JS itself already needs two parsers plus a per-language resolver.
Real cost accepted: `cmd/java` only errors until 005a lands; 010
(packaging) already depends on 005a or 006a working end to end.
**Answer:** Option (b). `cmd/typescript` and `cmd/all` wired for real
(TS/JS only). `cmd/java`, and `--lang=java` on `cmd/all`, fail fast with
an explicit not-implemented error -- never call `DetectLanguage` with
zero candidates (it panics). The exit code for that fast-fail is still
open, to be settled with the other exit codes.

---

**Q2 — Where does the rendered bundle go?**
Considered: (a) stdout and clipboard on every run, `--no-clipboard` to
skip the clipboard; (b) stdout always, clipboard only with an opt-in
`--copy`; (c) detect whether stdout is a TTY and pick stdout-or-clipboard
implicitly.
Known costs of (a): a piped run (`stba trace.txt | jq`) still overwrites
the clipboard unless `--no-clipboard` is passed; a headless machine logs a
clipboard Warn every run at the default level; on a TTY the full bundle
(up to ~512KB) is printed even when the clipboard was the point. Rejected
(c) as implicit behavior that changes with how the process is invoked.
**Answer:** Option (a). Bundle always goes to stdout (Article II) and to
the clipboard by default; new `--no-clipboard` flag skips the clipboard
write. The flag goes on all three binaries (`cmd/all`, `cmd/java`,
`cmd/typescript`), which means changing 002a's `internal/cli/parse.go`.
What a clipboard failure does (Warn + exit 0 vs. a dedicated exit code)
is deliberately NOT decided here -- separate question, still open.

**Q2 follow-up — Does normal shell redirection work as expected?**
Confirmed: `stba trace.txt` prints the bundle on the terminal (and copies
it); `stba trace.txt | jq ...` and `stba trace.txt > out.md` send stdout
(the bundle only) to the pipe/file instead. Caveats stated back: the
clipboard is still written in those cases unless `--no-clipboard` is
given; stderr (logs/warnings) is not redirected by `|` or `>`, so it
still shows on the terminal; `--format` defaults to `markdown` (002a
FR3), so piping to `jq` needs `--format json`.

**Q2 amendment — file output.** A separate saved record of Q2 said the
bundle should be written as a file in the invocation directory instead of
stdout, because a large bundle can be cut off in a terminal. Problems
raised against that: the generated file lands untracked inside the repo
being analyzed, so the next run's `GitMetadata.UncommittedChanges` would
read `true` because of the tool's own output; it breaks `| jq` and
`> out.md` (002a assumed stdout carries the bundle); and it adds spec
surface (filename, overwrite, unwritable directory).
Considered: keep stdout + clipboard only; default file in cwd; an
explicit opt-in `--output <path>` flag (Article VIII cost acknowledged --
it overlaps `> file`).
**Answer:** explicit `--output <path>` flag, no default file. Clipboard
default and `--no-clipboard` unchanged. Still open: unwritable-path exit
code.

---

**Q3 — With `--output <path>`, does the bundle also print to stdout?**
Considered: (a) the file replaces stdout (stdout stays empty; clipboard is
independent and still written unless `--no-clipboard`); (b) both, tee-style.
Reasoning for (a): `-o`/`--output` conventionally means "instead of
stdout"; it also addresses the terminal cut-off concern that motivated
the file idea; (b) would dump the full bundle to the terminal anyway, and
both-at-once is still available via `tee`.
**Answer:** Option (a). Flag is `-o`/`--output <path>` (short form
included). When set, nothing is written to stdout. Flag applies to all
three binaries (`cmd/all`, `cmd/java`, `cmd/typescript`) -- stated as the
assumption when the question was asked, not separately objected to.

---

**Q4 — What happens when the `--output` path already exists?**
Risk raised: `stba trace.txt -o trace.txt` is a plausible flag typo; the
input is fully read before any write, so the run would succeed and
silently destroy the only copy of the trace (shell `>` has the same
hazard, and worse -- it truncates before reading -- but a flag typo is
easier to make than a redirect).
Considered: (a) overwrite silently, no checks; (b) overwrite silently but
refuse when the path is the same file as the input file argument (usage
error, exit 2, specific message, `os.SameFile` after a `Stat`); (c) refuse
to overwrite any existing file unless a new `--force` flag is passed.
Reasoning for (b): closes the one destructive case cheaply, keeps repeat
runs to the same output file frictionless (the bundle is regenerable
from the trace), and avoids a `--force` flag Article VIII would need
justified.
**Answer:** Option (b). Existing files at the `--output` path are
overwritten silently, except when the path resolves to the same file as
the input file argument, which is a usage error (exit 2) with a specific
message. No `--force` flag. Not applicable when input comes from stdin.
Still open: exit code for an unwritable `--output` path (parent missing,
permission denied, etc.).

---

**Q5 — What happens when the clipboard write fails?**
Context: the bundle has already been delivered to stdout or the
`--output` file. `clipboard.Write` (009) returns `ErrNoClipboardUtility`
(no `xclip`/`wl-copy`/etc. on `PATH`) or `ErrClipboardWriteFailed`.
Considered: (a) Warn on stderr, exit 0; (b) exit 1 after delivering the
bundle; (c) new exit code 5, "bundle delivered, clipboard failed",
documented in `CONVENTIONS.md`.
Reasoning for (a): the failure is routine on headless Linux/CI and the
escape hatch is `--no-clipboard`; a non-zero exit after a successful
delivery makes scripts treat a good bundle as a failed run and forces
every headless caller to pass `--no-clipboard` just to get exit 0.
Cost accepted: a script cannot detect a failed copy from the exit code;
the clipboard is a convenience side effect, not the tool's contract.
No new exit code, `CONVENTIONS.md` unchanged. The Warn text should say
what to do (install the missing utility, or pass `--no-clipboard`).
**Answer:** Option (a). Clipboard failure is a Warn on stderr with exit 0.
This resolves 009 FR8's deferred question: no exit code is allocated for
clipboard errors.
Also settled by reading `CONVENTIONS.md` (not a new decision):
`ErrUnparseable` -> exit 3; `ErrNoMatch` and `ErrAmbiguous` -> exit 4.

---

**Q6 — How should an unwritable `--output` path fail?**
Context: unlike the clipboard, a failed file write means the bundle was
not delivered anywhere (`-o` replaces stdout), so it must be non-zero.
Hazards raised: late detection (a mistyped directory is only found after
the full pipeline -- git blame, dependency resolution -- has run) and
early truncation (opening the file at startup would wipe an existing
good output whenever the pipeline then fails with exit 3/4).
Considered: (a) write only at the end, any failure exit 1; (b) fail fast
on bad arguments, write at the end for everything else; (c) new exit
code 5 for all output-write failures.
Reasoning for (b): a mistyped directory is an argument error, same class
as Q4's same-file check, and is caught before any pipeline work; real I/O
failures are environmental and fit exit 1; a new exit code is a contract
to maintain (same reasoning as Q5). Known gap accepted: a partial write
on a full disk can corrupt an existing output file; temp-file-and-rename
would fix it but is extra code for a rare case (Article VIII).
**Answer:** Option (b).
- Before running the pipeline: if the `--output` path's parent directory
  does not exist or is not a directory, that is a usage error, exit 2,
  specific message (alongside Q4's same-file check).
- The output file is only opened/written after rendering succeeds, so a
  pipeline failure (exit 3/4/1) never touches an existing output file.
- Runtime write failures at the final write (permission denied, disk
  full, etc.) are exit 1 with a wrapped error naming the path.
- No temp-file-and-rename.

---

**Q7 — What does `--lang=typescript` do on `cmd/all`?**
Context: 002a defined `--lang` only on `cmd/all` (values `java`,
`typescript`) and said only that "omitted" defers to auto-detection.
006a split TS/JS into `javascriptParser` and `typescriptParser`, sharing
one engine and differing only in `Detect()` (TS = any `.ts`/`.tsx`
frame; JS = Node trace with none), so they never both match; compiled-
then-run TypeScript has only `.js` frames and is classified `javascript`.
Finding (from reading `internal/parser/typescript/engine.go`, not from
running it -- no shell tool): `parseTrace` accepts any first line that
contains `": "` as a valid exception header. A non-Node trace (e.g.
Java) pushed through `Parse` without `Detect` would return a one-node
chain with a junk `ClassName`, zero frames, the rest of the trace in
`Message`, and no error -- a plausible-looking wrong bundle, contrary to
Article VI. The parser is only safe behind `Detect`.
Considered: (a) hint filters candidates to the TS/JS family, `Detect`
still decides JS vs TS; (b) hint limits candidates to `typescriptParser`
only, `Detect` still runs (compiled-TS traces would exit 4 despite the
hint); (c) hint bypasses `Detect` and forces `typescriptParser`
(rejected: unsafe per the finding above).
**Answer:** Option (a).
- `--lang=typescript` on `cmd/all` -> candidates `{javascriptParser,
  typescriptParser}`; `DetectLanguage` still decides between them, so
  `Bundle.Language` may be `javascript` for a plain-JS or compiled-TS
  trace. A non-Node trace -> exit 4 (`ErrNoMatch`).
- Omitted `--lang` on `cmd/all` -> the same candidate set (Java is not
  registered, per Q1). `cmd/typescript` (no `--lang`) -> the same set.
- `--lang=java` -> Q1's fast-fail; `DetectLanguage` is never called.
- When the resolved language differs from the hint, log at Info (e.g.
  "hint typescript, trace classified javascript (no .ts frames)");
  invisible at the default Warn level, visible with `-v`.
- Invariant for the spec: the pipeline never calls `Parse` on a parser
  that did not return true from `Detect` for that trace.

---

**Q8 — Exit code and behavior for the Java fast-fail (Q1)?**
Context: applies when `cmd/java` is invoked at all, or `cmd/all
--lang=java` is (`java` currently passes 002a's flag validation as an
accepted value). CONVENTIONS codes considered: 1 (wrong -- expected,
known state, not a bug), 2 (usage error -- fits `--lang=java` on
`cmd/all`, a stretch for `cmd/java` where the flags are fine but the
binary can't work yet), 3 (wrong -- no input examined), 4 (wrong --
language is known, just unsupported), new code 5 (rejected -- documents a
state that disappears when Java support lands). Ordering risk raised: if
the fast-fail happens after input is read, `cmd/java` would block on an
open pipe before reporting it can't work.
**Answer:** Exit 2 for both cases. Error-level message is exactly "Java is
not supported yet" -- deliberately no feature IDs or internal names in
the user-facing text (stated explicitly). Checked right after flag
validation and before any input is read. When Java support lands, the
fast-fail is deleted and replaced with real wiring; no new exit code to
retire.

---

**Q9 — What does the "no parser matched" path (exit 4) print?**
Context: `memory/known-gaps.md` (006a's bare-stack row, owners 003b and
002b; 003b left the user-facing path to 002b) asks for a more helpful
message for a bare `.stack`-only line with zero frames (real example:
`TypeError: fetch failed`), which no `Detect()` can claim: point the user
at pasting the text directly into an LLM (no frames for git blame,
snippets or dependencies to act on) and mention `Error.stackTraceLimit`
being `0` as a likely cause. Problem raised: the row's trigger (message
looks like a native-binding/fetch failure) is a fuzzy JS-specific
heuristic that would break Article V if placed in `internal/cli`, and the
`Error.stackTraceLimit` hint is Node-specific noise once Java exists.
Considered: (a) generic message only, mark the known-gaps row
deliberately not done; (b) generic message plus a tip when the trimmed
input is a single line (language-agnostic trigger, no error-shape
guessing); (c) the tip on every exit 4 (noisy, wrong for e.g. Python).
**Answer:** Option (b).
- Every `ErrNoMatch` (exit 4): a generic Error-level message that the
  language of the input could not be detected, naming the checked
  languages (from the `ErrNoMatch` error text).
- When the trimmed input is a single line: additionally tell the user
  the input has no stack frames so there is nothing for this tool to
  add and they can paste it directly into their LLM, and that if the
  error came from Node.js, `Error.stackTraceLimit` may have been set to
  0. Exact wording is settled when spec.md is written.
- No feature IDs or internal names in either message (consistent with
  Q8).
- Accepted costs: multi-line inputs that are still frameless miss the
  tip; the `Error.stackTraceLimit` sentence is Node-specific and its
  wording gets revisited when Java support lands.
- Follow-up on completion of 002b: check the box in 006a's `spec.md`
  for this criterion and remove/mark done the known-gaps row.

---

**Q10 — Trailing newline when writing to stdout or the `--output` file?**
Context (verified in `internal/render/json.go` and 008's `qa-log.md` Q5):
`render.JSON` deliberately ends at `}` with no terminator, for token
efficiency when pasted into an LLM chat; `clipboard.Write` (009) writes
the string byte-for-byte. Nobody had decided what stdout/file do -- left
as is, JSON on a terminal leaves the shell prompt glued to `}` and a
saved JSON file triggers "no newline at end of file" warnings in
`diff`/`git`. (Skipped earlier by mistake; raised after the fact.)
Considered: (a) verbatim to all three destinations (byte-identical,
honors 008 literally, keeps the cosmetic costs); (b) append a single
`\n` to stdout and the `--output` file only if the rendered string does
not already end with one, clipboard stays byte-for-byte; (c) append to
all three including the clipboard (reopens 008's decision for the paste
path).
Reasoning for (b): 008's rationale concerns the paste payload, which (b)
leaves untouched; deterministic, no terminal sniffing; idempotent (no-op
if the string already ends in a newline, e.g. possibly Markdown).
Cost accepted: stdout/file and clipboard differ by one trailing byte for
JSON; piping stdout into `pbcopy` yourself carries that newline.
**Answer:** Option (b).

---

**Q11 — What goes in `Bundle.OS` on a platform outside linux/darwin/windows?**
Context: nothing sets `Bundle.OS` yet, so 002b must. `contract.OS` is
documented as populated via `runtime.GOOS`, values MUST be Go's own
constants; only three are defined (`linux`, `darwin`, `windows`); the
field is `json:"os"` without `omitempty`. Other `GOOS` values (freebsd,
openbsd, ...) can only occur in source builds.
Considered: (a) pass `runtime.GOOS` through verbatim (`"os":
"freebsd"`), the `OS` type is not enforced as closed; (b) fail with exit
1 on an unsupported OS (harsh -- the tool works there apart from the
clipboard, which already degrades via `ErrNoClipboardUtility`); (c) map
unknown to an empty string (rejected: serializes as `"os": ""`, a zero
value standing in for "not applicable", contrary to the contract header
and Article VI).
**Answer:** Option (a). `Bundle.OS = contract.OS(runtime.GOOS)` for every
value. No change to `internal/contract`, no schema bump. Plan must check
that nothing (renderers or elsewhere) validates `OS` against the three
constants -- not yet verified.

---

**Q12 — What context and cancellation model does the pipeline use?**
Context (read from `codecontext/runner.go`, `gitmeta.go`, `context.go`,
`dependency/typescript/resolve.go`): `BuildGitMetadata`,
`ResolveDependencies` and `BuildCodeContexts` never return errors; every
failure degrades inside the package (nil result, or per-frame status +
Warn), so the pipeline has no error branch or exit code for them. Every
git call has its own hard 10s timeout (`gitTimeout`). Hazard raised: a
cancelled or expired context does not fail those calls -- it collapses to
nil ("no git repo") / stale / "no git repository found" notes, so a run
with a cancellable context would still render a complete bundle carrying
false claims about the repo (Article VI).
Considered: (a) `signal.NotifyContext` in `main` plus an explicit
`ctx.Err() != nil` check before output (needs an exit code for
cancellation; 130 is not in CONVENTIONS); (b) `context.Background()`,
no signal handling, default SIGINT behavior; (c) (b) plus a global
deadline (unsafe: on expiry every remaining step degrades quietly, same
misleading-bundle hazard, unless (a)'s check is added too).
Reasoning for (b): Ctrl+C goes to the whole foreground process group, so
the tool and any in-flight `git` child die immediately, and the bundle is
only written at the very end (Q6), so there is no partial output; correct
today with zero code, no new exit code.
Cost accepted: no overall deadline; worst case is a hung git across many
own frames at up to 10s per call (one status per own frame, three in
`BuildGitMetadata`, one more in the TS resolver) -- slow but bounded, and
Ctrl+C works.
**Answer:** Option (b). `context.Background()` passed through the
pipeline; no signal handling; no global deadline. Standing invariant for
plan.md: if a cancellable or deadline context is ever introduced,
`ctx.Err()` must be checked before any output is written.

---

**Q13 — Is `-h`/`--help` in scope for 002b?**
Context: parked since 002a T004 (no acceptance criterion). Current
behavior: `ParseAll`/`ParseFixedLang` discard `pflag`'s output, so `-h`
gives no usage listing; it surfaces as a generic "help requested" error
through `slog.Error` and exits 2 (the usage-error code) for an explicit
request for help. 002b adds `-o`/`--output` and `--no-clipboard` on top
of `--lang`, `--format`, `-v`. Catch raised: `AGENTS.md`/Article II say
nothing but the bundle goes to stdout, while CLI convention sends
explicit help to stdout with exit 0 -- real help needs a decision on that
(stderr + exit 0, or a constitution amendment).
Considered: (a) out of scope, documented in spec.md; (b) in scope --
`-h`/`--help` prints `pflag` `FlagUsages()` to stderr and exits 0; (c)
only change the exit code to 0 while still printing nothing useful
(rejected).
Known costs of (b): help goes to the terminal, not the pipe, so
`--help | less` shows nothing in the pager; the usage-text requirement is
new scope touching `internal/cli` and all three `main.go` files.
**Answer:** Option (b), by explicit preference: build it as part of 002b
now rather than defer it to a follow-up feature that would have to
reopen the same code. `-h`/`--help` prints usage to stderr and exits 0.
No constitution amendment needed (stdout stays bundle-only). Each binary
lists its own flag set (`cmd/all` shows `--lang`; `cmd/java` and
`cmd/typescript` do not), which `FlagUsages()` gives per `FlagSet`.
Follow-ups: `ParseAll`/`ParseFixedLang` need a way for `main` to tell a
help request from a real error (sentinel error) and must stop routing it
through `slog.Error`; usage-text content and precedence are the next
question.

---

**Q14 — What does the usage text contain, and how does `-h` interact
with other flags?**
Context: `pflag` returns the help request as soon as it meets `-h` or
`--help`, scanning left to right; `validateLang`/`validateFormat` run
after parsing. Consequences taken as-is (no extra code to change them):
`stba --lang=cobol -h` prints help (never reaches the `--lang` error);
`stba --bogus -h` fails on the unknown flag (exit 2, `--bogus` first);
`stba -h --bogus` prints help; `cmd/java --help` prints help (Q8's Java
fast-fail runs after flag validation).
Considered: (a) synopsis line + one sentence on input (file argument, or
stdin when piped, file wins if both) + `FlagUsages()` listing; (b) (a)
plus usage examples and an exit-code table; (c) only `FlagUsages()`.
Risk raised on (b): examples and an exit-code table duplicate
`CONVENTIONS.md` and the eventual README and can go stale when a flag or
code changes.
**Answer:** Option (b), by explicit preference (reason given: examples
and exit codes are what make the tool easy for developers to use). The
usage text therefore contains: synopsis, input sentence, the flag
listing, usage examples, and an exit-code table.
Carried into the spec as assumptions -- the two defaults offered with
(a) were not separately confirmed: every flag gets a real help string
(plan must check what 002a's `--lang`, `--format`, `-v` already have);
the program name in the usage text is `filepath.Base(os.Args[0])`.
Proposed for plan.md, not yet confirmed: define the exit codes and the
examples once as Go data, render the help text from them, and add tests
that (1) every exit code appears in the help text and (2) every example's
args are accepted by `ParseAll`/`ParseFixedLang`, so neither can drift
from the real behavior.

---

**Q15 — Delivery order: stdout/`--output` first or clipboard first, and what
if the first fails?**
Context: Q5 settled a clipboard failure after a successful delivery
(Warn, exit 0); the reverse -- the output write failing -- was open.
Considered: (a) deliver to stdout or the `--output` file first; on
failure exit 1 and leave the clipboard untouched; on success write the
clipboard, whose failure is Warn + exit 0 (Q5); (b) clipboard first,
then output (a failed file write would exit 1 after already overwriting
the clipboard); (c) attempt both regardless, exit 1 if the output write
failed.
Reasoning for (a): a failed run has no side effects -- never "exit 1, but
your clipboard was overwritten" with an error saying nothing was
delivered. Accepted consequence: if a downstream reader closes stdout
(`| head -1`), Go's default is to die on SIGPIPE and the clipboard write
never happens.
**Answer:** Option (a).

---

**Q16 — What happens to 002a's stub logging?**
Context: 002a's spec note ("temporary stdout behavior") asks 002b to
replace/remove the `main.go` stub output when the real pipeline lands.
The stub, in each `main.go`: an Info `"parsed input"` line (bytes, lang
hint, format) at `-v`, and a Debug dump of the entire `Input` struct at
`-vv` (includes the full `rawText`, up to ~512KB). The Debug `"stdin
ignored"` line is a real 002a FR5 requirement and stays regardless.
Consequence for 002a: FR13 and three checked acceptance criteria (Info
summary at `-v`; Info summary + dump at `-vv`; the first criterion's
"Info summary + Debug dump available via `-v`/`-vv`" clause) stop being
true, so 002a's `spec.md` would contradict the code unless annotated.
Considered: (a) remove both stub logs, mark FR13 + those criteria
superseded in 002a's `spec.md`, add real Info lines for pipeline
outcomes; (b) remove only the Debug dump, keep `"parsed input"` as a
permanent Info line; (c) leave both stubs (contradicts 002a's note).
Reasoning for (a): `"parsed input"` describes an intermediate state that
stops mattering once the pipeline exists; what helps someone running
with `-v` is what the pipeline decided and where the bundle went; the
512KB dump has no remaining consumer.
**Answer:** Option (a).
- Remove the Info `"parsed input"` line and the Debug `Input` dump from
  all three `main.go` files. Keep the Debug `"stdin ignored"` line.
- New Info lines for pipeline outcomes: the language resolved (including
  Q7's hint-vs-resolved note) and where the bundle was delivered (stdout,
  the `--output` path, and whether the clipboard write succeeded).
  Exact wording is settled when spec.md is written. The existing
  `"dependencies resolved"` Info line from 006b is unchanged.
- On completion of 002b: annotate 002a's `spec.md` -- FR13 and the
  affected acceptance criteria marked superseded by 002b, and the
  "temporary stdout behavior" note closed.

---

**Q17 — Confirm the carried-forward assumptions so spec.md has no guesses.**
Four items not previously confirmed explicitly: (1) `-o`/`--output` and
`--no-clipboard` go on all three binaries (stated as an assumption in
Q3); (2) every flag gets a real help string and the usage text's program
name is `filepath.Base(os.Args[0])` (the two defaults from Q14's option
(a)); (3) the exit codes and usage examples are defined once as Go data,
the help is rendered from them, and tests check that every exit code
appears in the help and every example's args parse through
`ParseAll`/`ParseFixedLang` (proposed mitigation for the staleness risk of
Q14's choice); (4) the git and dependency steps use `os.Getwd()` as the
working directory, and a `Getwd` failure is exit 1 (unexpected error) --
new, not asked before.
**Answer:** Confirmed all four as stated. Plan must check what 002a's
`--lang`, `--format` and `-v` already have as help strings.

---

**Q18 — Do the code-context and dependency steps run in parallel?**
User proposal: run code-context and dependency resolution concurrently
for speed. Verified against the code: `ResolveDependencies(ctx, workDir,
chain)` needs neither `gitMeta` nor the code contexts and both steps only
read the chain; the only real dependency is `BuildCodeContexts` needing
`BuildGitMetadata`'s result, so the safe split is `BuildGitMetadata ->
BuildCodeContexts` concurrent with `ResolveDependencies`. Concurrent
read-only git calls are safe (`rev-parse` takes no lock; `status` takes
only an optional index lock).
Assessment given: gain bounded by the smaller step (dependency step =
file reads + one `git rev-parse`, estimated tens of ms; code-context
step dominates via per-own-frame `status` + `blame`); unmeasured (no
shell tool). Costs: nondeterministic log-line interleaving, tests need
`-race`, two concurrent git subprocesses (relevant on the
`\\wsl.localhost` filesystem). The bigger lever -- per-frame `status`/
`blame` inside `BuildCodeContexts` -- is 004's internals, out of 002b
scope. `sync.WaitGroup` (stdlib) would suffice; `errgroup` would be a new
third-party dependency needing an ADR.
Considered: (a) sequential, two steps written as independent calls; (b)
parallel now with `sync.WaitGroup`, joined before Bundle assembly.
**Answer:** Option (a), for MVP. Sequential, with the code-context and
dependency steps written as independent calls so parallelizing later is
a small change if measurement shows it matters.

---
