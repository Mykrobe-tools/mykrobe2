//go:build darwin || linux

package mccortex

import (
	"os"
	"syscall"
)

func LoadRuntimeIndex(path string) (*RuntimeIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	data, err := syscall.Mmap(int(f.Fd()), 0, int(info.Size()), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, err
	}
	idx, err := parseRuntimeIndex(data)
	if err != nil {
		_ = syscall.Munmap(data)
		return nil, err
	}
	return idx, nil
}

func runtimeIndexUnmap(data []byte) error {
	return syscall.Munmap(data)
}
