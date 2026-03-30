//go:build darwin || linux

package mccortex

import (
	"os"
	"syscall"
	"unsafe"
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

func mapUint64FileRW(path string, count int) ([]uint64, []byte, func() error, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	data, err := syscall.Mmap(int(f.Fd()), 0, count*8, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	closeErr := f.Close()
	if err != nil {
		if closeErr != nil {
			return nil, nil, nil, closeErr
		}
		return nil, nil, nil, err
	}
	if closeErr != nil {
		_ = syscall.Munmap(data)
		return nil, nil, nil, closeErr
	}
	table := unsafe.Slice((*uint64)(unsafe.Pointer(&data[0])), count)
	return table, data, func() error { return syscall.Munmap(data) }, nil
}
