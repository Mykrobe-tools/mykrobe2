package annotation

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Mykrobe-tools/mykrobe2/internal/testutil"
)

var annotationRefData = testutil.MykrobePath("tests", "ref_data")

func TestSimpleGeneForward(t *testing.T) {
	ref, err := loadReference(filepath.Join(annotationRefData, "NC_000962.3.fasta"))
	if err != nil {
		t.Fatal(err)
	}
	g := Gene{Name: "rpoB", Region: Region{Reference: ref, Start: 759807, End: 763325, Forward: true}}
	if g.Name != "rpoB" || !g.Forward || g.Strand() != "forward" {
		t.Fatalf("unexpected gene metadata: %#v", g)
	}
	codon, err := g.GetCodon(2)
	if err != nil || codon != "GCA" {
		t.Fatalf("unexpected codon got=%q err=%v", codon, err)
	}
	pos, err := g.GetReferencePosition(1)
	if err != nil || pos != 759807 {
		t.Fatalf("unexpected reference position got=%d err=%v", pos, err)
	}
	prev, err := g.GetReferencePosition(-1)
	if err != nil || prev != 759806 {
		t.Fatalf("unexpected negative position got=%d err=%v", prev, err)
	}
}

func TestSimpleGeneReverse(t *testing.T) {
	ref, err := loadReference(filepath.Join(annotationRefData, "NC_000962.3.fasta"))
	if err != nil {
		t.Fatal(err)
	}
	g := Gene{Name: "gidB", Region: Region{Reference: ref, Start: 4407528, End: 4408202, Forward: false}}
	if g.Forward || g.Strand() != "reverse" {
		t.Fatalf("unexpected strand metadata: %#v", g)
	}
	codon, err := g.GetCodon(2)
	if err != nil || codon != "TCT" {
		t.Fatalf("unexpected codon got=%q err=%v", codon, err)
	}
	pos, err := g.GetReferencePosition(1)
	if err != nil || pos != 4408202 {
		t.Fatalf("unexpected reference position got=%d err=%v", pos, err)
	}
	upstream, err := g.GetReferencePosition(-1)
	if err != nil || upstream != 4408203 {
		t.Fatalf("unexpected upstream position got=%d err=%v", upstream, err)
	}
}

func TestGetVariantNamesForward(t *testing.T) {
	gm := mustAA2DNA(t)
	got, err := gm.GetVariantNames("rpoB", "D3A", true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"GAT759813GCA", "GAT759813GCT", "GAT759813GCC", "GAT759813GCG"}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGetVariantNamesForwardAnyAA(t *testing.T) {
	gm := mustAA2DNA(t)
	got, err := gm.GetVariantNames("rpoB", "D3X", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 59 {
		t.Fatalf("unexpected D3X expansion size: %d", len(got))
	}
}

func TestGetVariantNamesReverse(t *testing.T) {
	gm := mustAA2DNA(t)
	got, err := gm.GetVariantNames("katG", "E3A", true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"CTC2156103TGC", "CTC2156103AGC", "CTC2156103GGC", "CTC2156103CGC"}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGetVariantNamesReverseProbeModelCases(t *testing.T) {
	gm := mustAA2DNA(t)
	got, err := gm.GetVariantNames("gid", "I11N", true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"GAT4408170ATT", "GAT4408170GTT"}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGetVariantNamesDNASpace(t *testing.T) {
	gm := mustAA2DNA(t)
	got, err := gm.GetVariantNames("pncA", "C18CCA", false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"G2289224TGG"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGetVariantNamesReverseUpstreamDNA(t *testing.T) {
	gm := mustAA2DNA(t)
	got, err := gm.GetVariantNames("eis", "G-10A", false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"C2715342T"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGetVariantNamesReverseDeletionLikeDNA(t *testing.T) {
	gm := mustAA2DNA(t)
	got, err := gm.GetVariantNames("eis", "TG-1T", false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"CA2715332A"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGetVariantNamesStopCodon(t *testing.T) {
	gm := mustAA2DNA(t)
	got, err := gm.GetVariantNames("katG", "W90*", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("unexpected stop codon expansions: %v", got)
	}
	want := []string{"CCA2155842TTA", "CCA2155842CTA", "CCA2155842TCA"}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGetAltsMatchesPythonCodonOrder(t *testing.T) {
	gm := mustAA2DNA(t)
	got := gm.GetAlts("L")
	want := []string{"TTA", "TTG", "CTA", "CTT", "CTC", "CTG"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGetAltsAnyAAMatchesPythonOrder(t *testing.T) {
	gm := mustAA2DNA(t)
	got := gm.GetAlts("X")
	wantPrefix := []string{
		"AAA", "AAG",
		"AAT", "AAC",
		"ATA", "ATT", "ATC",
		"ATG",
		"ACA", "ACT", "ACC", "ACG",
		"AGA", "AGG", "CGA", "CGT", "CGC", "CGG",
		"AGT", "AGC", "TCA", "TCT", "TCC", "TCG",
		"TAT", "TAC",
		"TTA", "TTG", "CTA", "CTT", "CTC", "CTG",
	}
	if !slices.Equal(got[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("got prefix %v want %v", got[:len(wantPrefix)], wantPrefix)
	}
}

func mustAA2DNA(t *testing.T) *GeneAminoAcidChangeToDNAVariants {
	t.Helper()
	gm, err := NewGeneAminoAcidChangeToDNAVariants(
		filepath.Join(annotationRefData, "NC_000962.3.fasta"),
		filepath.Join(annotationRefData, "NC_000962.3.gb"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return gm
}
