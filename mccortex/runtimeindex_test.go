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
	rtPath := filepath.Join(dir, "panel.runtimeidx.bin")
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
