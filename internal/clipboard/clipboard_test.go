package clipboard

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// call records one invocation of fakeCmdRunner.Run, so a test can assert
// which tool(s) were actually invoked, in order, and with what
// arguments/stdin -- the WSL-exclusivity acceptance criteria (T007) and
// the "xclip never attempted" criteria (T006) depend on this, not just
// on the final returned error.
type call struct {
	name  string
	args  []string
	stdin string
}

// fakeCmdRunner is the hand-written test double for this package
// (CONVENTIONS.md: no mocking framework), mirroring
// codecontext/runner_fake_test.go's fakeGitRunner pattern but extended
// with per-tool-name configurable LookPath/Run behavior and a call log,
// per plan.md's Testing strategy. lookPath maps a tool name to whether
// it is "found on PATH"; run maps a tool name to the behavior its Run
// call should exhibit (nil entry = found tool invokes successfully;
// present entry runs the given func, which can return an error or block
// on ctx for T008's timeout case).
type fakeCmdRunner struct {
	lookPath map[string]bool
	run      map[string]func(ctx context.Context) error
	calls    []call
}

var _ cmdRunner = (*fakeCmdRunner)(nil)

// LookPath implements cmdRunner.
func (f *fakeCmdRunner) LookPath(name string) bool {
	return f.lookPath[name]
}

// Run implements cmdRunner.
func (f *fakeCmdRunner) Run(ctx context.Context, name string, args []string, stdin string) error {
	f.calls = append(f.calls, call{name: name, args: args, stdin: stdin})

	if fn, ok := f.run[name]; ok {
		return fn(ctx)
	}
	return nil
}

// calledTools returns the name of every tool actually invoked via Run,
// in call order -- a tool that was only checked via LookPath but never
// invoked (because it wasn't found, or because an earlier candidate
// already succeeded) does not appear here.
func (f *fakeCmdRunner) calledTools() []string {
	names := make([]string, len(f.calls))
	for i, c := range f.calls {
		names[i] = c.name
	}
	return names
}

func TestWrite(t *testing.T) {
	const text = "bundle contents"

	tests := []struct {
		name      string
		goos      string
		wsl       bool
		lookPath  map[string]bool
		run       map[string]func(ctx context.Context) error
		wantErr   error
		wantCalls []string
	}{
		{
			name:      "darwin: pbcopy found",
			goos:      "darwin",
			lookPath:  map[string]bool{"pbcopy": true},
			wantErr:   nil,
			wantCalls: []string{"pbcopy"},
		},
		{
			name:      "darwin: pbcopy absent",
			goos:      "darwin",
			lookPath:  map[string]bool{},
			wantErr:   ErrNoClipboardUtility,
			wantCalls: nil,
		},
		{
			name:      "windows: clip.exe found",
			goos:      "windows",
			lookPath:  map[string]bool{"clip.exe": true},
			wantErr:   nil,
			wantCalls: []string{"clip.exe"},
		},
		{
			name:      "windows: clip.exe absent",
			goos:      "windows",
			lookPath:  map[string]bool{},
			wantErr:   ErrNoClipboardUtility,
			wantCalls: nil,
		},
		{
			// Both tools are (implausibly) present on PATH, to prove the
			// unrecognized-GOOS branch returns immediately with no
			// attempt made at all (spec.md FR6), rather than merely
			// "no tool happened to be found."
			name:      "unrecognized goos: no attempt made",
			goos:      "plan9",
			lookPath:  map[string]bool{"pbcopy": true, "clip.exe": true},
			wantErr:   ErrNoClipboardUtility,
			wantCalls: nil,
		},
		{
			name:      "linux non-WSL: wl-copy found+succeeds, xclip never attempted",
			goos:      "linux",
			lookPath:  map[string]bool{"wl-copy": true, "xclip": true},
			wantErr:   nil,
			wantCalls: []string{"wl-copy"},
		},
		{
			name:     "linux non-WSL: wl-copy found+fails, xclip found+succeeds",
			goos:     "linux",
			lookPath: map[string]bool{"wl-copy": true, "xclip": true},
			run: map[string]func(ctx context.Context) error{
				"wl-copy": func(context.Context) error { return errors.New("wl-copy: connection refused") },
			},
			wantErr:   nil,
			wantCalls: []string{"wl-copy", "xclip"},
		},
		{
			name:      "linux non-WSL: wl-copy not found, xclip found+succeeds",
			goos:      "linux",
			lookPath:  map[string]bool{"xclip": true},
			wantErr:   nil,
			wantCalls: []string{"xclip"},
		},
		{
			name:     "linux non-WSL: both found, both fail",
			goos:     "linux",
			lookPath: map[string]bool{"wl-copy": true, "xclip": true},
			run: map[string]func(ctx context.Context) error{
				"wl-copy": func(context.Context) error { return errors.New("wl-copy: connection refused") },
				"xclip":   func(context.Context) error { return errors.New("xclip: cannot open display") },
			},
			wantErr:   ErrClipboardWriteFailed,
			wantCalls: []string{"wl-copy", "xclip"},
		},
		{
			name:     "linux non-WSL: wl-copy found+fails, xclip not found",
			goos:     "linux",
			lookPath: map[string]bool{"wl-copy": true},
			run: map[string]func(ctx context.Context) error{
				"wl-copy": func(context.Context) error { return errors.New("wl-copy: connection refused") },
			},
			wantErr:   ErrClipboardWriteFailed,
			wantCalls: []string{"wl-copy"},
		},
		{
			name:     "linux non-WSL: wl-copy not found, xclip found+fails",
			goos:     "linux",
			lookPath: map[string]bool{"xclip": true},
			run: map[string]func(ctx context.Context) error{
				"xclip": func(context.Context) error { return errors.New("xclip: cannot open display") },
			},
			wantErr:   ErrClipboardWriteFailed,
			wantCalls: []string{"xclip"},
		},
		{
			name:      "linux non-WSL: neither found",
			goos:      "linux",
			lookPath:  map[string]bool{},
			wantErr:   ErrNoClipboardUtility,
			wantCalls: nil,
		},
		{
			// wl-copy/xclip are also (implausibly) present on PATH, to
			// prove WSL exclusivity: clip.exe alone is ever invoked.
			name:      "linux WSL: clip.exe found+succeeds, wl-copy/xclip never attempted",
			goos:      "linux",
			wsl:       true,
			lookPath:  map[string]bool{"clip.exe": true, "wl-copy": true, "xclip": true},
			wantErr:   nil,
			wantCalls: []string{"clip.exe"},
		},
		{
			// wl-copy/xclip are present on PATH but must never be tried --
			// WSL has no fallback (spec.md FR4).
			name:      "linux WSL: clip.exe absent, no fallback attempted",
			goos:      "linux",
			wsl:       true,
			lookPath:  map[string]bool{"wl-copy": true, "xclip": true},
			wantErr:   ErrNoClipboardUtility,
			wantCalls: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeCmdRunner{lookPath: tt.lookPath, run: tt.run}

			err := write(context.Background(), text, tt.goos, tt.wsl, runner)

			switch {
			case tt.wantErr == nil && err != nil:
				t.Fatalf("write() error = %v, want nil", err)
			case tt.wantErr != nil && !errors.Is(err, tt.wantErr):
				t.Fatalf("write() error = %v, want errors.Is(err, %v)", err, tt.wantErr)
			}

			if got := runner.calledTools(); !slices.Equal(got, tt.wantCalls) {
				t.Errorf("tools invoked = %v, want %v", got, tt.wantCalls)
			}
		})
	}
}

// TestWrite_Timeout proves a tool whose Run call hangs is bounded by the
// caller's ctx (spec.md FR7) and that the resulting cancellation is
// treated as a normal failed attempt, not a crash or an indefinite
// hang: wl-copy blocks until ctx.Done(), write() must still return
// promptly, and the non-WSL fallback chain must still fall through to
// xclip afterward exactly as it would for any other failure.
func TestWrite_Timeout(t *testing.T) {
	runner := &fakeCmdRunner{
		lookPath: map[string]bool{"wl-copy": true, "xclip": true},
		run: map[string]func(ctx context.Context) error{
			"wl-copy": func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := write(ctx, "bundle contents", "linux", false, runner)
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Fatalf("write() took %v to return, want bounded by the 50ms ctx deadline", elapsed)
	}

	if err != nil {
		t.Fatalf("write() error = %v, want nil (xclip should still succeed after wl-copy's timeout)", err)
	}

	if got := runner.calledTools(); !slices.Equal(got, []string{"wl-copy", "xclip"}) {
		t.Errorf("tools invoked = %v, want [wl-copy xclip] -- timeout must fall through like any other failure", got)
	}
}

// TestWrite_ByteForByte proves the exact input string reaches the
// subprocess's stdin unmodified -- no added or stripped trailing
// newline, no re-encoding (spec.md FR1). The fixture deliberately mixes
// internal newlines, a tab, and trailing whitespace, since a naive
// implementation might trim or normalize any of those.
func TestWrite_ByteForByte(t *testing.T) {
	const text = "line one\nline two\n\ttabbed, trailing space "

	runner := &fakeCmdRunner{lookPath: map[string]bool{"pbcopy": true}}

	if err := write(context.Background(), text, "darwin", false, runner); err != nil {
		t.Fatalf("write() error = %v, want nil", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(runner.calls))
	}

	if got := runner.calls[0].stdin; got != text {
		t.Errorf("stdin = %q, want %q", got, text)
	}
}
