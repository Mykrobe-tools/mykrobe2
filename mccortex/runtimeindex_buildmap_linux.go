//go:build linux

package mccortex

// Linux file-backed MAP_SHARED writes are pathologically slow for the large,
// randomly accessed TB hash tables on some otherwise fast ext4 systems. Build
// in pointer-free heap memory, then stream the completed table to its temporary
// file. Finished indexes are still memory-mapped read-only when loaded.
func mapUint64FileRW(path string, count int) ([]uint64, []byte, func() error, error) {
	return mapUint64HeapFileRW(path, count)
}
