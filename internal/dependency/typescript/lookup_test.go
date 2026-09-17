package typescript

import "testing"

func TestPackageDirKey(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		repoRoot string
		wantKey  string
		wantOK   bool
	}{
		{
			name:     "top-level package",
			filePath: "/repo/node_modules/lodash/index.js",
			repoRoot: "/repo",
			wantKey:  "node_modules/lodash",
			wantOK:   true,
		},
		{
			name:     "scoped package",
			filePath: "/repo/node_modules/@babel/core/lib/index.js",
			repoRoot: "/repo",
			wantKey:  "node_modules/@babel/core",
			wantOK:   true,
		},
		{
			name:     "nested duplicate -- full chain preserved, not just the trailing name",
			filePath: "/repo/node_modules/express/node_modules/statuses/index.js",
			repoRoot: "/repo",
			wantKey:  "node_modules/express/node_modules/statuses",
			wantOK:   true,
		},
		{
			name:     "no node_modules segment at all",
			filePath: "/repo/src/index.js",
			repoRoot: "/repo",
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, ok := packageDirKey(tt.filePath, tt.repoRoot)
			if ok != tt.wantOK {
				t.Fatalf("packageDirKey() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && key != tt.wantKey {
				t.Errorf("packageDirKey() key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestLookupFrame_ExactMatch(t *testing.T) {
	packages := map[string]string{"node_modules/lodash": "4.17.21"}

	version, note, matched := lookupFrame("/repo/node_modules/lodash/index.js", "lodash", "/repo", packages)
	if !matched {
		t.Fatalf("lookupFrame() matched = false, want true")
	}
	if version != "4.17.21" {
		t.Errorf("lookupFrame() version = %q, want %q", version, "4.17.21")
	}
	if note != "" {
		t.Errorf("lookupFrame() note = %q, want empty for an exact match", note)
	}
}

func TestLookupFrame_ScopedPackageExactMatch(t *testing.T) {
	packages := map[string]string{"node_modules/@babel/core": "7.24.0"}

	version, note, matched := lookupFrame("/repo/node_modules/@babel/core/lib/index.js", "@babel/core", "/repo", packages)
	if !matched {
		t.Fatalf("lookupFrame() matched = false, want true")
	}
	if version != "7.24.0" {
		t.Errorf("lookupFrame() version = %q, want %q", version, "7.24.0")
	}
	if note != "" {
		t.Errorf("lookupFrame() note = %q, want empty for an exact match", note)
	}
}

func TestLookupFrame_NestedDuplicateVersion_PicksExactNestedCopy(t *testing.T) {
	// Both a top-level AND a nested copy of the same package name exist
	// at different versions; the frame's own path points at the nested
	// one -- must report the nested copy's version, not silently prefer
	// (or fall back to) the top-level one.
	packages := map[string]string{
		"node_modules/statuses":                      "1.0.0",
		"node_modules/express/node_modules/statuses": "2.0.0",
	}

	version, note, matched := lookupFrame("/repo/node_modules/express/node_modules/statuses/index.js", "statuses", "/repo", packages)
	if !matched {
		t.Fatalf("lookupFrame() matched = false, want true")
	}
	if version != "2.0.0" {
		t.Errorf("lookupFrame() version = %q, want the nested copy's %q, not the top-level copy's version", version, "2.0.0")
	}
	if note != "" {
		t.Errorf("lookupFrame() note = %q, want empty -- this was an exact match, not a fallback", note)
	}
}

func TestLookupFrame_TopLevelFallback(t *testing.T) {
	// No exact key for this frame's specific nested location, but a
	// top-level bare-name entry exists.
	packages := map[string]string{"node_modules/lodash": "4.17.21"}

	version, note, matched := lookupFrame("/repo/node_modules/some-tool/node_modules/lodash/index.js", "lodash", "/repo", packages)
	if !matched {
		t.Fatalf("lookupFrame() matched = false, want true")
	}
	if version != "4.17.21" {
		t.Errorf("lookupFrame() version = %q, want %q", version, "4.17.21")
	}
	if note == "" {
		t.Error("lookupFrame() note is empty, want a non-empty note flagging the inexact match")
	}
}

func TestLookupFrame_Unresolved(t *testing.T) {
	packages := map[string]string{"node_modules/lodash": "4.17.21"}

	version, note, matched := lookupFrame("/repo/node_modules/left-pad/index.js", "left-pad", "/repo", packages)
	if matched {
		t.Fatalf("lookupFrame() matched = true, want false")
	}
	if version != "" || note != "" {
		t.Errorf("lookupFrame() = (%q, %q), want empty strings when unmatched", version, note)
	}
}
