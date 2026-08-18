package testutil

import (
	"path/filepath"
	"runtime"
)

// MykrobePath returns a path in the vendored Mykrobe parity fixtures.
func MykrobePath(parts ...string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	root := filepath.Join(repoRoot, "testdata", "mykrobe")
	return filepath.Join(append([]string{root}, parts...)...)
}
