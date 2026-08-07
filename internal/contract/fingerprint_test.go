package contract

import "testing"

// frame builders keep the table below readable -- only the fields that
// matter to fingerprinting are parameterized; everything else uses a
// zero value that's irrelevant to the identity computation.

func ownFrame(index int, filePath, className, methodName string) Frame {
	return Frame{
		Index:      index,
		FilePath:   filePath,
		ClassName:  className,
		MethodName: methodName,
		LineNumber: index + 1, // deliberately varies per case; must not affect the hash
		Bucket:     BucketOwn,
	}
}

func depFrame(index int, filePath, methodName string) Frame {
	return Frame{
		Index:      index,
		FilePath:   filePath,
		MethodName: methodName,
		LineNumber: index + 1,
		Bucket:     BucketDependency,
	}
}

func TestComputeFingerprint(t *testing.T) {
	tests := []struct {
		name      string
		chainA    []ExceptionNode
		chainB    []ExceptionNode
		wantEqual bool
	}{
		{
			// Acceptance: same bug + dependency version bump -> same
			// fingerprint. The originating frame (index 0) is own-bucket
			// in both chains and identical; only a non-originating
			// dependency frame's path changes (simulating a version
			// bump, e.g. a vendored/cached copy path embedding the
			// version), which must be excluded from the hash.
			name: "dependency version bump does not change fingerprint",
			chainA: []ExceptionNode{
				{
					ClassName: "app.MyError",
					Message:   "boom",
					Frames: []Frame{
						ownFrame(0, "/src/app.go", "", "DoThing"),
						depFrame(1, "/deps/lib@1.2.0/foo.go", "Foo"),
					},
				},
			},
			chainB: []ExceptionNode{
				{
					ClassName: "app.MyError",
					Message:   "boom",
					Frames: []Frame{
						ownFrame(0, "/src/app.go", "", "DoThing"),
						depFrame(1, "/deps/lib@1.3.0/foo.go", "Foo"),
					},
				},
			},
			wantEqual: true,
		},
		{
			// Acceptance: genuinely different bugs -> different
			// fingerprints. Same file, different method at the
			// originating own-bucket frame.
			name: "different own-bucket frame identity changes fingerprint",
			chainA: []ExceptionNode{
				{
					ClassName: "app.MyError",
					Message:   "boom",
					Frames: []Frame{
						ownFrame(0, "/src/app.go", "", "DoThing"),
					},
				},
			},
			chainB: []ExceptionNode{
				{
					ClassName: "app.MyError",
					Message:   "boom",
					Frames: []Frame{
						ownFrame(0, "/src/app.go", "", "DoOtherThing"),
					},
				},
			},
			wantEqual: false,
		},
		{
			// Acceptance: a chain with an identical outer wrapper but a
			// different inner cause -> different fingerprint, proving
			// every node is hashed, not just the outermost.
			name: "identical outer wrapper, different inner cause",
			chainA: []ExceptionNode{
				{
					ClassName: "app.WrapperError",
					Message:   "wrapped",
					Frames: []Frame{
						ownFrame(0, "/src/wrap.go", "", "Wrap"),
					},
				},
				{
					ClassName: "app.InnerErrorA",
					Message:   "inner a",
					Frames: []Frame{
						ownFrame(0, "/src/inner_a.go", "", "InnerA"),
					},
				},
			},
			chainB: []ExceptionNode{
				{
					ClassName: "app.WrapperError",
					Message:   "wrapped",
					Frames: []Frame{
						ownFrame(0, "/src/wrap.go", "", "Wrap"),
					},
				},
				{
					ClassName: "app.InnerErrorB",
					Message:   "inner b",
					Frames: []Frame{
						ownFrame(0, "/src/inner_b.go", "", "InnerB"),
					},
				},
			},
			wantEqual: false,
		},
		{
			// LineNumber must be excluded from the hash -- two chains
			// identical except for line numbers must produce the same
			// fingerprint.
			name: "line number differences do not change fingerprint",
			chainA: []ExceptionNode{
				{
					ClassName: "app.MyError",
					Message:   "boom",
					Frames: []Frame{
						{Index: 0, FilePath: "/src/app.go", MethodName: "DoThing", LineNumber: 10, Bucket: BucketOwn},
					},
				},
			},
			chainB: []ExceptionNode{
				{
					ClassName: "app.MyError",
					Message:   "boom",
					Frames: []Frame{
						{Index: 0, FilePath: "/src/app.go", MethodName: "DoThing", LineNumber: 999, Bucket: BucketOwn},
					},
				},
			},
			wantEqual: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotA := ComputeFingerprint(tt.chainA)
			gotB := ComputeFingerprint(tt.chainB)

			if (gotA == gotB) != tt.wantEqual {
				t.Errorf("ComputeFingerprint(chainA) = %q, ComputeFingerprint(chainB) = %q, wantEqual = %v",
					gotA, gotB, tt.wantEqual)
			}
		})
	}
}

// TestComputeFingerprint_Deterministic guards against any accidental
// nondeterminism (e.g. iterating a map) sneaking into the implementation.
func TestComputeFingerprint_Deterministic(t *testing.T) {
	chain := []ExceptionNode{
		{
			ClassName: "app.MyError",
			Message:   "boom",
			Frames: []Frame{
				ownFrame(0, "/src/app.go", "", "DoThing"),
				depFrame(1, "/deps/lib@1.2.0/foo.go", "Foo"),
			},
		},
	}

	first := ComputeFingerprint(chain)
	for range 10 {
		if got := ComputeFingerprint(chain); got != first {
			t.Fatalf("ComputeFingerprint is nondeterministic: got %q, want %q", got, first)
		}
	}
}
