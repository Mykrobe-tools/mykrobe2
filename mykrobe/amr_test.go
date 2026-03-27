package mykrobe

import "testing"

func TestEstimateKmerCountErrorRateAndIncorrectKmerPercentCov(t *testing.T) {
	variantCalls := map[string]Call{
		"refcall": {
			Genotype: []int{0, 0},
			Info: map[string]any{
				"coverage": map[string]any{
					"reference": map[string]any{"kmer_count": 90, "percent_coverage": 100.0},
					"alternate": map[string]any{"kmer_count": 10, "percent_coverage": 25.0},
				},
			},
		},
		"altcall": {
			Genotype: []int{1, 1},
			Info: map[string]any{
				"coverage": map[string]any{
					"reference": map[string]any{"kmer_count": 5, "percent_coverage": 10.0},
					"alternate": map[string]any{"kmer_count": 95, "percent_coverage": 100.0},
				},
			},
		},
	}
	gotRate, gotMap := EstimateKmerCountErrorRateAndIncorrectKmerPercentCov(variantCalls, DefaultErrorRate)
	if gotRate != 0.075 {
		t.Fatalf("unexpected error rate: %v", gotRate)
	}
	if gotMap[0] != 0 || gotMap[10] != 25 || gotMap[5] != 10 {
		t.Fatalf("unexpected incorrect kmer map: %#v", gotMap)
	}
}

func TestApplyONTDefaults(t *testing.T) {
	errRate, ploidy := ApplyONTDefaults(DefaultErrorRate, "diploid", true)
	if errRate != ONTErrorRate || ploidy != ONTPloidy {
		t.Fatalf("unexpected ONT defaults: %v %q", errRate, ploidy)
	}
}

func TestGuessSequenceMethod(t *testing.T) {
	errRate, ploidy, guessed := GuessSequenceMethod(DefaultErrorRate, "diploid", true, 0.002)
	if !guessed || errRate != ONTErrorRate || ploidy != ONTPloidy {
		t.Fatalf("unexpected guessed platform: %v %q %v", errRate, ploidy, guessed)
	}
}

func TestConfThresholderGetConfThreshold(t *testing.T) {
	ct := NewConfThresholder(0.01, 20, 21, map[int]float64{0: 0, 1: 5, 2: 10}, 200)
	got := ct.GetConfThreshold(95)
	if got < 0 {
		t.Fatalf("unexpected conf threshold: %d", got)
	}
}
