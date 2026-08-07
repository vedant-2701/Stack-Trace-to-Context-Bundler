package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Field/frame/node separators for the fingerprint's hash input. Chosen
// from the ASCII information-separator range specifically because they
// can't appear in a file path, class name, or method name, so there's no
// risk of two different frames producing the same joined string by
// accident (e.g. "foo" + "bar" colliding with "foob" + "ar" if a plain
// character like ":" were used instead).
const (
	fingerprintFieldSep = "\x1f" // unit separator, between filePath/className/methodName
	fingerprintFrameSep = "\x1e" // record separator, between frames within a node
	fingerprintNodeSep  = "\x1d" // group separator, between nodes in the chain
)

// ComputeFingerprint computes a stable identity hash for an exception
// chain, per spec.md requirement 5. For every node in chain (not only the
// outermost), it hashes the file+method/class identity (excluding line
// numbers, which are too volatile) of every own-bucket frame in that
// node, plus the single frame where that node's exception actually
// originates -- regardless of that originating frame's bucket. Frames
// beyond that one originating frame in a dependency/runtime bucket are
// excluded, so a library version bump alone does not change the
// fingerprint of a bug that hasn't actually changed.
//
// The originating frame is Frames[0] -- the frame closest to where the
// exception was actually thrown, by stack-trace convention (top of
// stack).
//
// Algorithm (SHA-256, truncated to the first 16 hex characters) is an
// implementation detail per plan.md, not a functional requirement -- the
// only functional guarantees are: same identity in -> same fingerprint
// out, and a change to any hashed field -> a different fingerprint.
func ComputeFingerprint(chain []ExceptionNode) string {
	nodeParts := make([]string, len(chain))
	for i, node := range chain {
		nodeParts[i] = fingerprintNodeIdentity(node)
	}

	sum := sha256.Sum256([]byte(strings.Join(nodeParts, fingerprintNodeSep)))
	return hex.EncodeToString(sum[:])[:16]
}

// fingerprintNodeIdentity builds the identity string for a single
// exception node: every own-bucket frame (in original order), plus the
// originating frame (Frames[0]) if it isn't already included as an
// own-bucket frame.
func fingerprintNodeIdentity(node ExceptionNode) string {
	var frameParts []string
	included := make(map[int]bool, len(node.Frames))

	for _, f := range node.Frames {
		if f.Bucket == BucketOwn {
			frameParts = append(frameParts, fingerprintFrameIdentity(f))
			included[f.Index] = true
		}
	}

	if len(node.Frames) > 0 {
		origin := node.Frames[0]
		if !included[origin.Index] {
			frameParts = append(frameParts, fingerprintFrameIdentity(origin))
		}
	}

	return strings.Join(frameParts, fingerprintFrameSep)
}

// fingerprintFrameIdentity is a frame's file+method/class identity,
// deliberately excluding LineNumber, ColumnNumber, Index, Bucket, and
// PackageName -- none of those are part of "which bug is this",
// per spec.md requirement 5.
func fingerprintFrameIdentity(f Frame) string {
	return f.FilePath + fingerprintFieldSep + f.ClassName + fingerprintFieldSep + f.MethodName
}
