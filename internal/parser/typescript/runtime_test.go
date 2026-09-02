package typescript

import (
	"context"
	"errors"
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

// fakeNodeVersionRunner is the hand-written test double for this
// package's node-version fallback (CONVENTIONS.md: no mocking
// framework), mirroring internal/codecontext/runner_fake_test.go's
// fakeGitRunner. fn is invoked for every Run call; each test supplies
// whatever behavior it needs -- canned stdout, a specific error, or a
// simulated timeout (context.DeadlineExceeded).
type fakeNodeVersionRunner struct {
	fn func(ctx context.Context) (string, error)
}

var _ nodeVersionRunner = (*fakeNodeVersionRunner)(nil)

// Run implements nodeVersionRunner.
func (f *fakeNodeVersionRunner) Run(ctx context.Context) (string, error) {
	return f.fn(ctx)
}

func TestLocalNodeVersion(t *testing.T) {
	tests := []struct {
		name        string
		fn          func(ctx context.Context) (string, error)
		wantVersion string
		wantSource  contract.VersionSource
		wantNoteSet bool
	}{
		{
			name: "success sets VersionSourceLocalEnvironment and a note",
			fn: func(context.Context) (string, error) {
				return "v24.18.0\n", nil
			},
			wantVersion: "24.18.0",
			wantSource:  contract.VersionSourceLocalEnvironment,
			wantNoteSet: true,
		},
		{
			name: "node not on PATH sets VersionSourceUnknown",
			fn: func(context.Context) (string, error) {
				return "", errors.New("exec: \"node\": executable file not found in $PATH")
			},
			wantVersion: "",
			wantSource:  contract.VersionSourceUnknown,
			wantNoteSet: false,
		},
		{
			name: "timeout sets VersionSourceUnknown",
			fn: func(context.Context) (string, error) {
				return "", context.DeadlineExceeded
			},
			wantVersion: "",
			wantSource:  contract.VersionSourceUnknown,
			wantNoteSet: false,
		},
		{
			name: "unparseable stdout sets VersionSourceUnknown",
			fn: func(context.Context) (string, error) {
				return "not a version string", nil
			},
			wantVersion: "",
			wantSource:  contract.VersionSourceUnknown,
			wantNoteSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeNodeVersionRunner{fn: tt.fn}

			gotVersion, gotSource, gotNote := localNodeVersion(context.Background(), fake)

			if gotVersion != tt.wantVersion {
				t.Errorf("version = %q, want %q", gotVersion, tt.wantVersion)
			}
			if gotSource != tt.wantSource {
				t.Errorf("source = %q, want %q", gotSource, tt.wantSource)
			}
			if tt.wantNoteSet && gotNote == "" {
				t.Errorf("note = %q, want a non-empty note", gotNote)
			}
			if !tt.wantNoteSet && gotNote != "" {
				t.Errorf("note = %q, want empty", gotNote)
			}
		})
	}
}
