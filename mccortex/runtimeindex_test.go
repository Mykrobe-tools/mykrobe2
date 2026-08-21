package mccortex

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRuntimeIndexRoundTripAndSummariesMatchPanelPath(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	reads := filepath.Join(dir, "reads.fa")
	if err := os.WriteFile(panel, []byte(">probe1\nAACCGGTT\n>probe2\nAACCGGTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reads, []byte(">read1\nAACCGGTT\n>read2\nAACCGGTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rtIdx, err := BuildRuntimeIndex(5, []string{panel})
	if err != nil {
		t.Fatal(err)
	}
	rtPath := filepath.Join(dir, "panel.panelindex")
	if err := SaveRuntimeIndex(rtPath, rtIdx); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadRuntimeIndex(rtPath)
	if err != nil {
		t.Fatal(err)
	}
	defer loaded.Close()
	rtCounter, err := NewRuntimeCounter(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if err := rtCounter.AddPath(reads); err != nil {
		t.Fatal(err)
	}
	got := rtCounter.Summaries()

	counter, err := NewCounter(5)
	if err != nil {
		t.Fatal(err)
	}
	if err := counter.AddPath(reads); err != nil {
		t.Fatal(err)
	}
	want, err := counter.SummarizePanelPath(panel)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runtime summaries mismatch\nwant: %#v\ngot: %#v", want, got)
	}
}

func TestBuildRuntimeIndexFileReportsMonotonicProgress(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	if err := os.WriteFile(panel, []byte(">probe1\nAACCGGTT\n>probe2\nAACCGGTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var fractions []float64
	indexPath := filepath.Join(dir, "panel.panelindex")
	if err := BuildRuntimeIndexFileWithProgress(indexPath, 5, []string{panel}, func(fraction float64) {
		fractions = append(fractions, fraction)
	}); err != nil {
		t.Fatal(err)
	}
	if len(fractions) < 2 || fractions[0] != 0 || fractions[len(fractions)-1] != 1 {
		t.Fatalf("progress endpoints = %#v, want 0..1", fractions)
	}
	for i := 1; i < len(fractions); i++ {
		if fractions[i] < fractions[i-1] {
			t.Fatalf("progress decreased at %d: %#v", i, fractions)
		}
	}
}
