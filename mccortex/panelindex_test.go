package mccortex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPanelIndexRoundTripAndSummarizeMatchesPath(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	reads := filepath.Join(dir, "reads.fa")
	indexPath := filepath.Join(dir, "panel.idx.gob.gz")
	if err := os.WriteFile(panel, []byte(">p1\nACGTGCACTA\n>p2\nTTTTTCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reads, []byte(">r1\nACGTGCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}

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
	idx, err := BuildPanelIndex(5, []string{panel})
	if err != nil {
		t.Fatal(err)
	}
	if err := SavePanelIndex(indexPath, idx); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPanelIndex(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	got, err := counter.SummarizePanelIndex(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("len(got)=%d len(want)=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("summary %d mismatch\ngot  %#v\nwant %#v", i, got[i], want[i])
		}
	}
}
