package mccortex

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestGraphBuildMatchesMccortexCoverageExpectation(t *testing.T) {
	g, err := NewGraph(19)
	if err != nil {
		t.Fatal(err)
	}

	// Ported from mccortex src/tests/build_graph_tests.c.
	g.AddSequence([]byte("CTACGATGTATGCTTAGCTGTTCCG"))
	g.AddSequence([]byte("TAGAACGTTCCCTACACGTCCTATG"))

	if got := g.KmerCount("CTACGATGTATGCTTAGCT"); got != 1 {
		t.Fatalf("count=%d, want 1", got)
	}
	if got := g.KmerCount("TAGAACGTTCCCTACACGT"); got != 1 {
		t.Fatalf("count=%d, want 1", got)
	}
}

func TestGraphUnitigsMatchesMccortexSupernodeCases(t *testing.T) {
	g, err := NewGraph(19)
	if err != nil {
		t.Fatal(err)
	}

	seqs := []string{
		"AGAGAGAGAGAGAGAGAGAGAGAG",
		"AAAAAAAAAAAAAAAAAAAAAAAAAA",
		"ATATATATATATATATATATATATATAT",
		"CGTTCGCGCATGGCCCACG",
		"GAACCAATCGGTCGACTGT",
		"CCCCGCAAAGTCCACTTAGTGTAAGGTACAAATTCTGCAGAGTTGCTGGATCAGCGATAC",
		"TCAATCCGATAGCAACCCGGTCCAATCAATCCGATAGCAACCCGGTCCAA",
	}
	want := []string{
		"AAAAAAAAAAAAAAAAAAA",
		"AACCCGGTCCAATCAATCCGATAGCAACCCGGTCCAATCAATC",
		"ACAGTCGACCGATTGGTTC",
		"AGAGAGAGAGAGAGAGAGAG",
		"ATATATATATATATATATA",
		"CCCCGCAAAGTCCACTTAGTGTAAGGTACAAATTCTGCAGAGTTGCTGGATCAGCGATAC",
		"CGTGGGCCATGCGCGAACG",
	}

	for _, seq := range seqs {
		g.AddSequence([]byte(seq))
	}

	got := g.Unitigs()
	if !slices.Equal(got, want) {
		t.Fatalf("unitigs=%q, want %q", got, want)
	}
}

func TestGraphSubgraphDistanceFromMccortexSubgraphCase(t *testing.T) {
	g, err := NewGraph(19)
	if err != nil {
		t.Fatal(err)
	}

	graphseq := "" +
		"GGCTACCTAACCAGATATCTCTGTATACAGCTGCATTGTGTTTAGTCTACAACGACAGAAATCCCCTTCGACGCCCGC" +
		"GACCTCTCTTAACGGACGACGCCTTCCGGTTGCGATATCGATGGATCGACAGAACAAGCCGCTTCCCTAACAACTGCG" +
		"CATGAAATCCAAAGTGCGCCGATGCTTGCTTGACGATTCCAAATCCCCATGTGACCTGTGAAGACGACTACCGTAAGA" +
		"TGTGTCACGGGTCAGTCGCTTTTACCACCTACGGAAGGTAGACGGTTATACTCAATTATTGGCACTTTAGCTGGGCAG" +
		"GTCAAAGGGAACAAGTCTGAAGTAGATATAACCTCAGTCCTTTATACGCACGTGACCCGCGTATAATCTTGCCGGTGC" +
		"GCAACGAGGGGCTTGGATAAAACAGCTTGGGACTTATACGTTCACCCACGACCCGCCTTAGCTCAACGCTCGTAACGA" +
		"CTGAATATGAGTAACGTACCTGAGGTGGGTCCGCCTTGCGGAGGTGGTGGTTCTTACTTCTATCCTCTTGTAGAGAAA" +
		"AGAATAGGTCGTCACTAACACTCTTGTGGGGACAAACGTGTATCGATTCCCAAACGTCCGTTAGTGAATATCCTACGT" +
		"GTTCCATTCGATCACACTGGAATATGGCCTTAGTTGGCCCATCTTAGTGCGCCAAGTGTTCGCAGTGGTCGTAGGCAA" +
		"CAGGCATCGGCGGTCTAGAGTTCACGCCAAGTCGGCCGTGTGAAGTTAAGCGTAAGTGCGGGACAACAAACCGAATGT" +
		"TCCGTGGCACACATGTTCGCTTATTATCAGGTAACCCTCATCTCCAGGGAGAACGCCTCAGCAGGCTTGCACCGCTTG" +
		"TAATCCCTCCTTATCAGAAGTAATCGTCGTTGCCGAGTTAGATCATGTCGGGACGTTGCCCTCAAGACGCCCAACGGA" +
		"AAAATTCACGATAGTGGCGCTCGGGAGGAGTACGCAACTCAGCACCCCGGTGAGTAGCTCCCTT"
	g.AddSequence([]byte(graphseq))

	seed := []byte("GAGGTGGGTCCGCCTTGCGGt")
	sub, err := g.SubgraphFromSequences([][]byte{seed}, 10, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := sub.NumKmers(); got != 22 {
		t.Fatalf("num kmers=%d, want 22", got)
	}
}

func TestGraphSubgraphWholeUnitigsMatchesMccortexSupernodeSubgraphCase(t *testing.T) {
	g, err := NewGraph(11)
	if err != nil {
		t.Fatal(err)
	}
	g.AddSequence([]byte("ATGGTGCCTAGAAGGTA"))
	g.AddSequence([]byte("cTGGTGCCTAGAAGGTg"))

	sub, err := g.SubgraphFromSequences([][]byte{[]byte("TGCCTAGAAGG")}, 0, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := sub.NumKmers(); got != 5 {
		t.Fatalf("num kmers=%d, want 5", got)
	}
}

func TestGraphJoinIntersectAndRoundTrip(t *testing.T) {
	g1, _ := NewGraph(5)
	g2, _ := NewGraph(5)
	g3, _ := NewGraph(5)

	g1.AddSequence([]byte("ACGTGCACTA"))
	g2.AddSequence([]byte("TGCACTATTA"))
	g3.AddSequence([]byte("GCACT"))

	joined, err := g1.Join(g2)
	if err != nil {
		t.Fatal(err)
	}
	if joined.NumKmers() <= g1.NumKmers() {
		t.Fatalf("joined graph did not grow: %d vs %d", joined.NumKmers(), g1.NumKmers())
	}

	inter, err := joined.Intersect(g3)
	if err != nil {
		t.Fatal(err)
	}
	if got := inter.NumKmers(); got != 1 {
		t.Fatalf("intersect num kmers=%d, want 1", got)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "graph.ctx")
	if err := inter.Save(path); err != nil {
		t.Fatal(err)
	}
	round, err := LoadGraph(path)
	if err != nil {
		t.Fatal(err)
	}
	if round.K != inter.K || round.NumKmers() != inter.NumKmers() {
		t.Fatalf("round trip mismatch: got k=%d n=%d want k=%d n=%d", round.K, round.NumKmers(), inter.K, inter.NumKmers())
	}
}

func TestGraphAddPathUsesFaqtReader(t *testing.T) {
	g, err := NewGraph(5)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "seq.fa")
	if err := os.WriteFile(path, []byte(">r1\nACGTGCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPath(path); err != nil {
		t.Fatal(err)
	}
	if got := g.NumKmers(); got == 0 {
		t.Fatal("graph is empty after AddPath")
	}
}
