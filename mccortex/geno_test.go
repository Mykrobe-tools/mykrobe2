package mccortex

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCounterCountKmerStringMatchesBuildGraphExpectation(t *testing.T) {
	counter, err := NewCounter(19)
	if err != nil {
		t.Fatal(err)
	}

	// Ported from mccortex src/tests/build_graph_tests.c
	counter.AddSequence([]byte("CTACGATGTATGCTTAGCTGTTCCG"))
	counter.AddSequence([]byte("TAGAACGTTCCCTACACGTCCTATG"))

	got, err := counter.CountKmerString("CTACGATGTATGCTTAGCT")
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("count(CTACGATGTATGCTTAGCT)=%d, want 1", got)
	}

	got, err = counter.CountKmerString("TAGAACGTTCCCTACACGT")
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("count(TAGAACGTTCCCTACACGT)=%d, want 1", got)
	}
}

func TestCounterDuplicateReadIncrementsCoverageLikeBuildGraphWithoutPCRFiltering(t *testing.T) {
	counter, err := NewCounter(19)
	if err != nil {
		t.Fatal(err)
	}

	// Derived from the "filtering turned off" case in build_graph_tests.c.
	seq := []byte("CTACGATGTATGCTTAGCTAATGAT")
	counter.AddSequence(seq)
	counter.AddSequence(seq)

	got, err := counter.CountKmerString("CTACGATGTATGCTTAGCT")
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("count(CTACGATGTATGCTTAGCT)=%d, want 2", got)
	}
}

func TestCounterCanonicalizesReverseComplements(t *testing.T) {
	counter, err := NewCounter(19)
	if err != nil {
		t.Fatal(err)
	}

	// Reverse complement pair called out in build_graph_tests.c comments.
	counter.AddSequence([]byte("CTACGATGTATGCTTAGCTGTTCCG"))

	for _, kmer := range []string{
		"CTACGATGTATGCTTAGCT",
		"AGCTAAGCATACATCGTAG",
	} {
		got, err := counter.CountKmerString(kmer)
		if err != nil {
			t.Fatal(err)
		}
		if got != 1 {
			t.Fatalf("count(%s)=%d, want 1", kmer, got)
		}
	}
}

func TestSummarizeSequenceMatchesMccortexGenoStyleRow(t *testing.T) {
	counter, err := NewCounter(5)
	if err != nil {
		t.Fatal(err)
	}
	counter.AddSequence([]byte("ACGTGCACTA"))

	got, err := counter.SummarizeSequence("probe1", []byte("ACGTGCACTA"))
	if err != nil {
		t.Fatal(err)
	}

	want := CoverageSummary{
		Name:            "probe1",
		Colour:          0,
		MedianDepth:     1,
		MinDepth:        1,
		PercentCoverage: 1,
		KmerCount:       7,
		KmerLength:      6,
	}
	if got != want {
		t.Fatalf("summary=%+v, want %+v", got, want)
	}
}

func TestSummarizePanelPathAndWriteCoverageTSV(t *testing.T) {
	counter, err := NewCounter(5)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	reads := filepath.Join(dir, "reads.fa")
	panel := filepath.Join(dir, "panel.fa")

	if err := os.WriteFile(reads, []byte(">r1\nACGTGCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(panel, []byte(">probe1\nACGTGCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := counter.AddPath(reads); err != nil {
		t.Fatal(err)
	}
	summaries, err := counter.SummarizePanelPath(panel)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries)=%d, want 1", len(summaries))
	}

	var buf bytes.Buffer
	if err := WriteCoverageTSV(&buf, summaries); err != nil {
		t.Fatal(err)
	}

	got := strings.TrimSpace(buf.String())
	want := "probe1\t0\t1\t1\t1.000000\t7\t6"
	if got != want {
		t.Fatalf("tsv=%q, want %q", got, want)
	}
}
