package mccortex

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/martinghunt/faqt/seqio"
)

// Counter stores canonical kmer counts from sequence inputs.
// It is intentionally scoped to the Mykrobe-relevant mccortex31 use case.
type Counter struct {
	k      int
	counts map[uint64]uint32
}

func NewCounter(k int) (*Counter, error) {
	if k < 1 || k > 31 {
		return nil, fmt.Errorf("k must be in range 1..31, got %d", k)
	}
	return &Counter{k: k, counts: make(map[uint64]uint32)}, nil
}

func (c *Counter) K() int {
	return c.k
}

func (c *Counter) AddPath(path string) error {
	reader, err := seqio.OpenPath(path)
	if err != nil {
		return err
	}
	defer closeIfPossible(reader)

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		c.AddSequence(rec.Seq)
	}
}

func (c *Counter) AddSequence(seq []byte) {
	if len(seq) < c.k {
		return
	}

	var (
		fwd   uint64
		rev   uint64
		valid int
		mask  = uint64(1<<(2*c.k)) - 1
	)

	for _, base := range seq {
		code, ok := encodeBase(base)
		if !ok {
			fwd, rev, valid = 0, 0, 0
			continue
		}

		fwd = ((fwd << 2) | uint64(code)) & mask
		rev = (rev >> 2) | (uint64(code^0x3) << (2 * (c.k - 1)))

		if valid < c.k {
			valid++
		}
		if valid < c.k {
			continue
		}

		key := fwd
		if rev < key {
			key = rev
		}
		c.counts[key]++
	}
}

func (c *Counter) CountKmerString(kmer string) (uint32, error) {
	if len(kmer) != c.k {
		return 0, fmt.Errorf("kmer length %d does not match k=%d", len(kmer), c.k)
	}
	key, err := canonicalKmerString(kmer)
	if err != nil {
		return 0, err
	}
	return c.counts[key], nil
}

type CoverageSummary struct {
	Name            string
	Colour          int
	MedianDepth     uint32
	MinDepth        uint32
	PercentCoverage float64
	KmerCount       uint32
	KmerLength      int
}

func (c *Counter) SummarizePanelPath(path string) ([]CoverageSummary, error) {
	reader, err := seqio.OpenPath(path)
	if err != nil {
		return nil, err
	}
	defer closeIfPossible(reader)

	var out []CoverageSummary
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		summary, err := c.SummarizeSequence(rec.Name, rec.Seq)
		if err != nil {
			return nil, err
		}
		out = append(out, summary)
	}
}

// SummarizeSequence intentionally mirrors the quirks of mccortex31 ctx_geno:
// coverage for the first kmer slot is left as zero, then the median is taken
// over the full kmer-length slice including that leading zero.
func (c *Counter) SummarizeSequence(name string, seq []byte) (CoverageSummary, error) {
	klen := 0
	if len(seq) >= c.k {
		klen = len(seq) - c.k + 1
	}

	summary := CoverageSummary{Name: name, Colour: 0, KmerLength: klen}
	if klen <= 1 {
		return summary, nil
	}

	covgs := make([]uint32, klen)
	for i := 1; i < klen; i++ {
		key, ok, err := canonicalKmer(seq[i : i+c.k])
		if err != nil {
			return CoverageSummary{}, err
		}
		if !ok {
			continue
		}
		covgs[i] = c.counts[key]
	}

	minDepth := uint32(^uint32(0))
	var nonZero float64
	for i := 1; i < klen; i++ {
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

	sorted := slices.Clone(covgs)
	slices.Sort(sorted)
	summary.MedianDepth = medianUint32(sorted)
	summary.MinDepth = minDepth
	summary.PercentCoverage = nonZero / float64(klen-1)
	return summary, nil
}

func WriteCoverageTSV(w io.Writer, summaries []CoverageSummary) error {
	for _, s := range summaries {
		if _, err := fmt.Fprintf(
			w,
			"%s\t%d\t%d\t%d\t%f\t%d\t%d\n",
			s.Name,
			s.Colour,
			s.MedianDepth,
			s.MinDepth,
			s.PercentCoverage,
			s.KmerCount,
			s.KmerLength,
		); err != nil {
			return err
		}
	}
	return nil
}

func canonicalKmerString(kmer string) (uint64, error) {
	key, ok, err := canonicalKmer([]byte(strings.ToUpper(kmer)))
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("invalid kmer %q", kmer)
	}
	return key, nil
}

func canonicalKmer(seq []byte) (uint64, bool, error) {
	var fwd uint64
	var rev uint64
	for i, base := range seq {
		code, ok := encodeBase(base)
		if !ok {
			return 0, false, nil
		}
		fwd = (fwd << 2) | uint64(code)
		rev |= uint64(code^0x3) << (2 * i)
	}
	if len(seq) > 31 {
		return 0, false, fmt.Errorf("kmer length %d exceeds 31", len(seq))
	}
	if rev < fwd {
		return rev, true, nil
	}
	return fwd, true, nil
}

func encodeBase(b byte) (uint8, bool) {
	switch b {
	case 'A', 'a':
		return 0, true
	case 'C', 'c':
		return 1, true
	case 'G', 'g':
		return 2, true
	case 'T', 't':
		return 3, true
	default:
		return 0, false
	}
}

func medianUint32(sorted []uint32) uint32 {
	if len(sorted) == 0 {
		return 0
	}
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid] + 1) / 2
}

func closeIfPossible(v any) {
	closer, ok := v.(io.Closer)
	if ok {
		_ = closer.Close()
	}
}
