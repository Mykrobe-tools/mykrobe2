package mccortex

import (
	"bufio"
	"encoding/binary"
	"os"
)

func mapUint64HeapFileRW(path string, count int) ([]uint64, []byte, func() error, error) {
	table := make([]uint64, count)
	for i := range table {
		table[i] = invalidPanelKmer
	}
	writeBack := func() error {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return err
		}
		writer := bufio.NewWriterSize(f, 1<<20)
		buf := make([]byte, 1<<20)
		valuesPerChunk := len(buf) / 8
		for start := 0; start < len(table); start += valuesPerChunk {
			end := min(start+valuesPerChunk, len(table))
			for i, value := range table[start:end] {
				binary.LittleEndian.PutUint64(buf[i*8:], value)
			}
			if _, err := writer.Write(buf[:(end-start)*8]); err != nil {
				_ = f.Close()
				return err
			}
		}
		if err := writer.Flush(); err != nil {
			_ = f.Close()
			return err
		}
		return f.Close()
	}
	return table, nil, writeBack, nil
}
