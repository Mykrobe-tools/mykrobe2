package mccortex

import (
	"io"

	"github.com/martinghunt/faqt/seqio"
)

func BuildGraphFromPaths(k int, paths []string) (*Graph, error) {
	g, err := NewGraph(k)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		if err := g.AddPath(path); err != nil {
			return nil, err
		}
	}
	return g, nil
}

func LoadSequences(path string) ([][]byte, error) {
	reader, err := seqio.OpenPath(path)
	if err != nil {
		return nil, err
	}
	defer closeIfPossible(reader)

	var out [][]byte
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, append([]byte(nil), rec.Seq...))
	}
}
