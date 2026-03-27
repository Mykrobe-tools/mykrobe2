package mccortex

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/martinghunt/faqt/seqio"
)

const (
	panelIndexFormat = "mykrobe2-panel-index-v1"
	invalidPanelKmer = ^uint64(0)
)

type PanelIndex struct {
	Format string
	K      int
	Probes []IndexedProbe
}

type IndexedProbe struct {
	Name       string
	KmerLength int
	Kmers      []uint64
}

func BuildPanelIndex(k int, paths []string) (*PanelIndex, error) {
	idx := &PanelIndex{Format: panelIndexFormat, K: k}
	for _, path := range paths {
		reader, err := seqio.OpenPath(path)
		if err != nil {
			return nil, err
		}
		for {
			rec, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				closeIfPossible(reader)
				return nil, err
			}
			probe, err := buildIndexedProbe(k, rec.Name, rec.Seq)
			if err != nil {
				closeIfPossible(reader)
				return nil, err
			}
			idx.Probes = append(idx.Probes, probe)
		}
		closeIfPossible(reader)
	}
	return idx, nil
}

func buildIndexedProbe(k int, name string, seq []byte) (IndexedProbe, error) {
	klen := 0
	if len(seq) >= k {
		klen = len(seq) - k + 1
	}
	probe := IndexedProbe{
		Name:       name,
		KmerLength: klen,
		Kmers:      make([]uint64, klen),
	}
	if klen <= 1 {
		return probe, nil
	}
	for i := range probe.Kmers {
		probe.Kmers[i] = invalidPanelKmer
	}
	upper := strings.ToUpper(string(seq))
	for i := 1; i < klen; i++ {
		key, ok, err := canonicalKmer([]byte(upper[i : i+k]))
		if err != nil {
			return IndexedProbe{}, err
		}
		if !ok {
			continue
		}
		probe.Kmers[i] = key
	}
	return probe, nil
}

func SavePanelIndex(path string, idx *PanelIndex) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	defer gw.Close()
	return gob.NewEncoder(gw).Encode(idx)
}

func LoadPanelIndex(path string) (*PanelIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	var idx PanelIndex
	if err := gob.NewDecoder(gr).Decode(&idx); err != nil {
		return nil, err
	}
	if idx.Format != panelIndexFormat {
		return nil, fmt.Errorf("unsupported panel index format %q", idx.Format)
	}
	return &idx, nil
}

func (c *Counter) SummarizePanelIndex(idx *PanelIndex) ([]CoverageSummary, error) {
	if idx.K != c.k {
		return nil, fmt.Errorf("panel index k=%d does not match counter k=%d", idx.K, c.k)
	}
	out := make([]CoverageSummary, 0, len(idx.Probes))
	for _, probe := range idx.Probes {
		out = append(out, c.SummarizeIndexedProbe(probe))
	}
	return out, nil
}

func (c *Counter) SummarizeIndexedProbe(probe IndexedProbe) CoverageSummary {
	summary := CoverageSummary{Name: probe.Name, Colour: 0, KmerLength: probe.KmerLength}
	if probe.KmerLength <= 1 {
		return summary
	}
	covgs := make([]uint32, probe.KmerLength)
	for i := 1; i < probe.KmerLength; i++ {
		if probe.Kmers[i] == invalidPanelKmer {
			continue
		}
		covgs[i] = c.counts[probe.Kmers[i]]
	}
	minDepth := uint32(^uint32(0))
	var nonZero float64
	for i := 1; i < probe.KmerLength; i++ {
		depth := covgs[i]
		summary.KmerCount += depth
		if depth == 0 {
			continue
		}
		nonZero++
		if depth < minDepth {
			minDepth = depth
		}
	}
	if minDepth == ^uint32(0) {
		minDepth = 0
	}
	sorted := append([]uint32(nil), covgs...)
	slices.Sort(sorted)
	summary.MedianDepth = medianUint32(sorted)
	summary.MinDepth = minDepth
	summary.PercentCoverage = nonZero / float64(probe.KmerLength-1)
	return summary
}
