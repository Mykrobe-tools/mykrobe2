package mykrobe

import (
	"math"
	"testing"
)

func TestPresenceTyperBaseCaseNoCoverage(t *testing.T) {
	pt := NewPresenceTyper([]float64{100}, nil, 1)
	pc := ProbeCoverage{MinDepth: 0, PercentCoverage: 0, MedianDepth: 0, KCount: 0, KLen: 31}
	s := SequenceProbeCoverage{Name: "A123T", ProbeCoverage: pc}
	call := pt.Type(s)
	if got := call.Genotype; got[0] != 0 || got[1] != 0 {
		t.Fatalf("genotype=%v", got)
	}
}

func TestPresenceTyperGene11(t *testing.T) {
	pt := NewPresenceTyper([]float64{100}, nil, 1)
	pc := ProbeCoverage{MinDepth: 100, PercentCoverage: 100, MedianDepth: 100, KCount: 100, KLen: 31}
	s := SequenceProbeCoverage{Name: "A123T", ProbeCoverage: pc, PercentCoverageThreshold: 80}
	call := pt.Type(s)
	if got := call.Genotype; got[0] != 1 || got[1] != 1 {
		t.Fatalf("genotype=%v", got)
	}
}

func TestPresenceTyperGene01(t *testing.T) {
	pt := NewPresenceTyper([]float64{100}, nil, 1)
	pc := ProbeCoverage{MinDepth: 100, PercentCoverage: 82, MedianDepth: 2, KCount: 82, KLen: 31}
	s := SequenceProbeCoverage{Name: "A123T", ProbeCoverage: pc, PercentCoverageThreshold: 80}
	call := pt.Type(s)
	if got := call.Genotype; got[0] != 0 || got[1] != 1 {
		t.Fatalf("genotype=%v", got)
	}
}

func TestVariantTyperSimpleCases(t *testing.T) {
	vt := NewVariantTyper([]float64{100}, nil, DefaultErrorRate, DefaultMinorFreq, false, nil, 3, "kmer_count", DefaultKmerSize, 0.3, "diploid")
	ref := ProbeCoverage{MinDepth: 100, PercentCoverage: 100, MedianDepth: 100, KCount: 100, KLen: 31}
	alt := ProbeCoverage{MinDepth: 100, PercentCoverage: 3, MedianDepth: 100, KCount: 3, KLen: 31}
	v := &VariantProbeCoverage{VarName: "A123T", ReferenceCoverages: []ProbeCoverage{ref}, AlternateCoverages: []ProbeCoverage{alt}}
	call := vt.Type(v)
	if got := call.Genotype; got[0] != 0 || got[1] != 0 {
		t.Fatalf("wt genotype=%v", got)
	}

	v = &VariantProbeCoverage{VarName: "A123T", ReferenceCoverages: []ProbeCoverage{alt}, AlternateCoverages: []ProbeCoverage{ref}}
	call = vt.Type(v)
	if got := call.Genotype; got[0] != 1 || got[1] != 1 {
		t.Fatalf("alt genotype=%v", got)
	}

	half := ProbeCoverage{MinDepth: 100, PercentCoverage: 100, MedianDepth: 50, KCount: 50, KLen: 31}
	v = &VariantProbeCoverage{VarName: "A123T", ReferenceCoverages: []ProbeCoverage{half}, AlternateCoverages: []ProbeCoverage{half}}
	call = vt.Type(v)
	if got := call.Genotype; got[0] != 0 || got[1] != 1 {
		t.Fatalf("mixed genotype=%v", got)
	}
}

func TestVariantTyperLowMinimumCases(t *testing.T) {
	vt := NewVariantTyper([]float64{1}, nil, DefaultErrorRate, DefaultMinorFreq, false, nil, 3, "kmer_count", DefaultKmerSize, 0.3, "diploid")
	ref := ProbeCoverage{MinDepth: 2, PercentCoverage: 59.52, MedianDepth: 2, KCount: 60, KLen: 31}
	alt := ProbeCoverage{MinDepth: 1, PercentCoverage: 83.33, MedianDepth: 1, KCount: 83, KLen: 31}
	v := &VariantProbeCoverage{VarName: "A123T", ReferenceCoverages: []ProbeCoverage{ref}, AlternateCoverages: []ProbeCoverage{alt}}
	call := vt.Type(v)
	if got := call.Genotype; got[0] != 1 || got[1] != 1 {
		t.Fatalf("genotype=%v", got)
	}
	if call.Info["conf"].(int) >= 150 {
		t.Fatalf("conf=%v", call.Info["conf"])
	}
}

func TestLogLikProbabilityOfNGapsMatchesPythonRounding(t *testing.T) {
	if got := pythonRoundToInt(20.5); got != 20 {
		t.Fatalf("pythonRoundToInt(20.5)=%d want 20", got)
	}
	got := LogLikProbabilityOfNGaps(4, 50, 41)
	want := -48.81511632110543
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("got=%v want=%v", got, want)
	}
}

func TestLikelihoodsToConfidenceMatchesPythonRounding(t *testing.T) {
	got := LikelihoodsToConfidence([]float64{-15.5, -99999999, -9770.1})
	if got != 9755 {
		t.Fatalf("got=%d want=9755", got)
	}
}

func TestGeneCollectionTyperGetBestVersionSortsTiesByVersionAscending(t *testing.T) {
	gt := NewGeneCollectionTyper([]float64{36}, nil, 1)
	got := gt.GetBestVersion(map[string]SequenceProbeCoverage{
		"13": {Version: "13", ProbeCoverage: ProbeCoverage{PercentCoverage: 100}},
		"1":  {Version: "1", ProbeCoverage: ProbeCoverage{PercentCoverage: 100}},
		"8":  {Version: "8", ProbeCoverage: ProbeCoverage{PercentCoverage: 100}},
		"7":  {Version: "7", ProbeCoverage: ProbeCoverage{PercentCoverage: 100}},
	}, 100)
	want := []string{"1", "7", "8", "13"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Version != w {
			t.Fatalf("at %d got=%s want=%s full=%v", i, got[i].Version, w, got)
		}
	}
}
