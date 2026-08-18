package testutil

import (
	"os"
	"path/filepath"
	"runtime"
)

// MykrobePath returns a path in the upstream Mykrobe checkout used for parity
// fixtures. MYKROBE_SOURCE_DIR can override the default sibling checkout.
func MykrobePath(parts ...string) string {
	root := os.Getenv("MYKROBE_SOURCE_DIR")
	if root == "" {
		_, thisFile, _, _ := runtime.Caller(0)
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
		root = filepath.Join(filepath.Dir(repoRoot), "mykrobe")
	}
	return filepath.Join(append([]string{root}, parts...)...)
}
