package clipboard

import "errors"

// ErrNoClipboardUtility is returned by Write when no usable clipboard tool
// could be found on PATH for the current OS/environment -- e.g. neither
// wl-copy nor xclip is present on non-WSL Linux, or pbcopy/clip.exe is
// missing on darwin/windows, or runtime.GOOS is unrecognized. Distinguishes
// "nothing to even try" from ErrClipboardWriteFailed ("something was found
// but every attempt to invoke it failed"). A future caller (002b) maps this
// to its own CLI exit code via errors.Is, mirroring parser.ErrNoMatch's
// exit-code-distinguishing role.
var ErrNoClipboardUtility = errors.New("no usable clipboard utility found on PATH")

// ErrClipboardWriteFailed is returned by Write when at least one clipboard
// tool was found on PATH but every attempt to invoke it failed (non-zero
// exit or the FR7 timeout elapsed). The wrapping error's message names only
// the tool(s) that were actually found and invoked, and how each failed --
// a tool that was never found is not named as a "failure" (spec.md FR5/FR6).
// A future caller (002b) maps this to its own CLI exit code via errors.Is,
// mirroring parser.ErrAmbiguous's exit-code-distinguishing role.
var ErrClipboardWriteFailed = errors.New("clipboard utility found but write failed")
