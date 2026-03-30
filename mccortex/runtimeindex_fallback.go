//go:build !darwin && !linux

package mccortex

import (
	"encoding/binary"
	"os"
)

func LoadRuntimeIndex(path string) (*RuntimeIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseRuntimeIndex(data)
}

func runtimeIndexUnmap(data []byte) error {
	return nil
}

func mapUint64FileRW(path string, count int) ([]uint64, []byte, func() error, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, err
	}
	table := make([]uint64, count)
	for i := range table {
		table[i] = binary.LittleEndian.Uint64(data[i*8:])
	}
	return table, nil, func() error {
		buf := make([]byte, len(table)*8)
		for i, v := range table {
			binary.LittleEndian.PutUint64(buf[i*8:], v)
		}
		return os.WriteFile(path, buf, 0o644)
	}, nil
}
