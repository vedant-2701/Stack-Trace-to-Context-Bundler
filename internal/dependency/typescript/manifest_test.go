package typescript

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeManifest writes contents to package.json in a fresh temp dir and
// returns its path.
func writeManifest(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "package.json")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestReadManifest_AllThreeSectionsMerged(t *testing.T) {
	path := writeManifest(t, `{
		"dependencies": {"lodash": "^4.17.21"},
		"devDependencies": {"jest": "^29.0.0"},
		"optionalDependencies": {"fsevents": "^2.3.2"}
	}`)

	direct, ok := readManifest(path)
	if !ok {
		t.Fatalf("readManifest() ok = false, want true")
	}
	want := map[string]string{
		"lodash":   "^4.17.21",
		"jest":     "^29.0.0",
		"fsevents": "^2.3.2",
	}
	if !reflect.DeepEqual(direct, want) {
		t.Errorf("readManifest() direct = %v, want %v", direct, want)
	}
}

func TestReadManifest_PeerDependenciesExcluded(t *testing.T) {
	path := writeManifest(t, `{
		"dependencies": {"lodash": "^4.17.21"},
		"peerDependencies": {"react": "^18.0.0"}
	}`)

	direct, ok := readManifest(path)
	if !ok {
		t.Fatalf("readManifest() ok = false, want true")
	}
	if _, present := direct["react"]; present {
		t.Errorf("readManifest() direct contains peerDependencies entry %q, want excluded", "react")
	}
	if direct["lodash"] != "^4.17.21" {
		t.Errorf(`readManifest() direct["lodash"] = %q, want "^4.17.21"`, direct["lodash"])
	}
}

func TestReadManifest_MalformedJSON(t *testing.T) {
	path := writeManifest(t, `{ this is not valid JSON`)

	_, ok := readManifest(path)
	if ok {
		t.Fatalf("readManifest() ok = true, want false for malformed JSON")
	}
}

func TestReadManifest_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist", "package.json")

	_, ok := readManifest(path)
	if ok {
		t.Fatalf("readManifest() ok = true, want false for a missing file")
	}
}
