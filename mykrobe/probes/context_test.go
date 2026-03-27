package probes

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestContextIndexNearbyExcludesSelfAndFindsNeighbors(t *testing.T) {
	vars := []Variant{
		mustVariant(t, "ref", "A31T"),
		mustVariant(t, "ref", "A32T"),
		mustVariant(t, "ref", "C30G"),
		mustVariant(t, "ref", "G500A"),
	}
	idx := NewContextIndex(vars)
	got := idx.Nearby(vars[0], 0, 31)
	want := []Variant{vars[1], vars[2]}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if !sameVariantSets([][]Variant{got}, [][]Variant{want}) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestContextIndexDrivenCreateMatchesManualContext(t *testing.T) {
	ag := mustAlleleGenerator(t, filepath.Join(probeTestRefData, "BX571856.1.fasta"), 31)
	vars := []Variant{
		mustVariant(t, "ref", "A31T"),
		mustVariant(t, "ref", "A32T"),
	}
	idx := NewContextIndex(vars)
	gotPanel, err := ag.Create(vars[0], idx.Nearby(vars[0], 0, ag.Kmer))
	if err != nil {
		t.Fatal(err)
	}
	wantPanel, err := ag.Create(vars[0], []Variant{vars[1]})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(gotPanel.Refs)
	slices.Sort(gotPanel.Alts)
	slices.Sort(wantPanel.Refs)
	slices.Sort(wantPanel.Alts)
	if !slices.Equal(gotPanel.Refs, wantPanel.Refs) || !slices.Equal(gotPanel.Alts, wantPanel.Alts) {
		t.Fatalf("got refs=%v alts=%v want refs=%v alts=%v", gotPanel.Refs, gotPanel.Alts, wantPanel.Refs, wantPanel.Alts)
	}
}
