package typescript

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeLockfile writes contents to a file named name (typically
// "package-lock.json" or "yarn.lock") in a fresh temp dir and returns
// that dir's path.
func writeLockfile(t *testing.T, name, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return dir
}

func TestReadLockfile_ValidV2(t *testing.T) {
	dir := writeLockfile(t, "package-lock.json", `{
		"lockfileVersion": 2,
		"packages": {
			"": {"name": "app"},
			"node_modules/lodash": {"version": "4.17.21"}
		}
	}`)

	packages, ok, reason := readLockfile(filepath.Join(dir, "package-lock.json"))
	if !ok {
		t.Fatalf("readLockfile() ok = false, want true (reason: %q)", reason)
	}
	want := map[string]string{
		"":                    "",
		"node_modules/lodash": "4.17.21",
	}
	if !reflect.DeepEqual(packages, want) {
		t.Errorf("readLockfile() packages = %v, want %v", packages, want)
	}
}

func TestReadLockfile_ValidV3(t *testing.T) {
	dir := writeLockfile(t, "package-lock.json", `{
		"lockfileVersion": 3,
		"packages": {
			"node_modules/@babel/core": {"version": "7.24.0"}
		}
	}`)

	packages, ok, reason := readLockfile(filepath.Join(dir, "package-lock.json"))
	if !ok {
		t.Fatalf("readLockfile() ok = false, want true (reason: %q)", reason)
	}
	want := map[string]string{"node_modules/@babel/core": "7.24.0"}
	if !reflect.DeepEqual(packages, want) {
		t.Errorf("readLockfile() packages = %v, want %v", packages, want)
	}
}

func TestReadLockfile_MalformedJSON(t *testing.T) {
	dir := writeLockfile(t, "package-lock.json", `{ this is not valid JSON`)

	_, ok, reason := readLockfile(filepath.Join(dir, "package-lock.json"))
	if ok {
		t.Fatalf("readLockfile() ok = true, want false for malformed JSON")
	}
	if reason == "" {
		t.Error("readLockfile() reason is empty, want a non-empty explanation")
	}
}

func TestReadLockfile_MissingFile(t *testing.T) {
	dir := t.TempDir()

	_, ok, reason := readLockfile(filepath.Join(dir, "package-lock.json"))
	if ok {
		t.Fatalf("readLockfile() ok = true, want false for a missing file")
	}
	if reason == "" {
		t.Error("readLockfile() reason is empty, want a non-empty explanation")
	}
}

func TestReadLockfile_LockfileVersion1(t *testing.T) {
	dir := writeLockfile(t, "package-lock.json", `{
		"lockfileVersion": 1,
		"dependencies": {
			"lodash": {"version": "4.17.21"}
		}
	}`)

	_, ok, reason := readLockfile(filepath.Join(dir, "package-lock.json"))
	if ok {
		t.Fatalf("readLockfile() ok = true, want false for lockfileVersion 1")
	}
	if reason == "" {
		t.Error("readLockfile() reason is empty, want a non-empty explanation")
	}
}

func TestReadLockfile_YarnOnly_TreatedAsMissing(t *testing.T) {
	// package-lock.json doesn't exist, but yarn.lock does, alongside it
	// -- confirms yarn.lock's mere presence changes nothing: this
	// function only ever looks for package-lock.json by exact path.
	dir := writeLockfile(t, "yarn.lock", "# yarn lockfile v1\n")

	_, ok, reason := readLockfile(filepath.Join(dir, "package-lock.json"))
	if ok {
		t.Fatalf("readLockfile() ok = true, want false when only yarn.lock is present")
	}
	if reason == "" {
		t.Error("readLockfile() reason is empty, want a non-empty explanation")
	}
}
