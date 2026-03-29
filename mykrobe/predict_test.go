package mykrobe

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTBPredictorCoverageGreaterThanThreshold(t *testing.T) {
	p, err := NewTBPredictor(nil, nil, "/Users/martin/git/mykrobe/tests/ref_data/tb_variant_to_resistance_drug.json")
	if err != nil {
		t.Fatal(err)
	}
	call := Call{
		Genotype:            []int{0, 1},
		GenotypeLikelihoods: []float64{0.1, 0.9, 0.12},
		Info: map[string]any{
			"contamination_depths": []float64{},
			"coverage": map[string]any{
				"alternate": map[string]any{"percent_coverage": 100.0, "median_depth": 15.0, "min_depth": 2.0},
				"reference": map[string]any{"percent_coverage": 100.0, "median_depth": 139.0, "min_depth": 128.0},
			},
			"expected_depths": []float64{152},
		},
	}
	if p.CoverageGreaterThanThreshold(call, []string{""}) {
		t.Fatal("expected threshold check to be false")
	}
}

func TestSusceptibilityResultFromJSONAndDiff(t *testing.T) {
	r1, err := SusceptibilityResultFromJSON(`{"susceptibility":{"Rifampicin":{"predict":"R"}}}`)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := SusceptibilityResultFromJSON(`{"susceptibility":{"Rifampicin":{"predict":"R"}}}`)
	if err != nil {
		t.Fatal(err)
	}
	r3, err := SusceptibilityResultFromJSON(`{"susceptibility":{"Rifampicin":{"predict":"S"}}}`)
	if err != nil {
		t.Fatal(err)
	}
	r4, err := SusceptibilityResultFromJSON(`{"susceptibility":{"Quin":{"predict":"S"}}}`)
	if err != nil {
		t.Fatal(err)
	}
	if diff := r1.Diff(r2); len(diff) != 0 {
		t.Fatalf("diff=%v", diff)
	}
	diff := r1.Diff(r3)
	if diff["Rifampicin"]["predict"] != [2]string{"R", "S"} {
		t.Fatalf("diff=%v", diff)
	}
	diff = r1.Diff(r4)
	if diff["Rifampicin"]["predict"] != [2]string{"R", "NA"} || diff["Quin"]["predict"] != [2]string{"NA", "S"} {
		t.Fatalf("diff=%v", diff)
	}
}

func TestAnalyzeCoverageSetTB(t *testing.T) {
	set, err := ParseCoverageReader(strings.NewReader(
		"katG?name=katG&panel_type=presence&version=1\t0\t100\t100\t1.0\t100\t31\n" +
			"ref-A123T?var_name=A123T&gene=katG&mut=A123T\t0\t100\t100\t1.0\t100\t31\n" +
			"alt-A123T?var_name=A123T&gene=katG&mut=A123T\t0\t3\t3\t0.03\t3\t31\n",
	))
	if err != nil {
		t.Fatal(err)
	}
	res, err := AnalyzeCoverageSetTB(set, 100, "/Users/martin/git/mykrobe/tests/ref_data/tb_variant_to_resistance_drug.json", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.GeneCalls) == 0 {
		t.Fatal("expected gene calls")
	}
}

func TestTBPredictorAddsCalledByForResistanceCalls(t *testing.T) {
	variantCalls := map[string]Call{
		"inhA_I21T-ATC1674262ACT": {
			Class:               "Call.VariantCall",
			Genotype:            []int{1, 1},
			GenotypeLikelihoods: []float64{-10, -99999999, -1},
			Info: map[string]any{
				"contamination_depths": []float64{},
				"coverage": map[string]any{
					"alternate": map[string]any{"percent_coverage": 100.0, "median_depth": 4.0, "min_non_zero_depth": 3.0, "kmer_count": 72, "klen": 20},
					"reference": map[string]any{"percent_coverage": 0.0, "median_depth": 0.0, "min_non_zero_depth": 0.0, "kmer_count": 0, "klen": 21},
				},
				"expected_depths": []float64{4},
				"filter":          []string{},
				"conf":            458,
			},
			Variant: "ref-I21T?var_name=ATC1674262ACT&gene=inhA&mut=I21T",
		},
	}
	p, err := NewTBPredictor(variantCalls, nil, "/Users/martin/git/mykrobe/tests/ref_data/tb_variant_to_resistance_drug.json")
	if err != nil {
		t.Fatal(err)
	}
	got := p.Run().Susceptibility["Isoniazid"]
	if got["predict"] != "R" {
		t.Fatalf("predict=%v", got)
	}
	calledBy, ok := got["called_by"].(map[string]Call)
	if !ok {
		t.Fatalf("missing called_by: %v", got)
	}
	call, ok := calledBy["inhA_I21T-ATC1674262ACT"]
	if !ok {
		t.Fatalf("missing resistance call: %v", calledBy)
	}
	if call.Class != "Call.VariantCall" || call.Variant != nil {
		t.Fatalf("unexpected stored call: %+v", call)
	}
	if contam, ok := call.Info["contamination_depths"].([]float64); !ok || len(contam) != 0 {
		t.Fatalf("unexpected contamination depths: %#v", call.Info["contamination_depths"])
	}
	if got := variantCalls["inhA_I21T-ATC1674262ACT"].Variant; got != nil {
		t.Fatalf("expected source variant call to be mutated to null variant, got %#v", got)
	}
}

func TestTBPredictorMutatesCalledGeneToNullVariant(t *testing.T) {
	mapPath := filepath.Join(t.TempDir(), "staph_map.json")
	if err := os.WriteFile(mapPath, []byte(`{"ermC":["Erythromycin"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	calledGenes := map[string]Call{
		"ermC": {
			Class:               "Call.SequenceCall",
			Genotype:            []int{1, 1},
			GenotypeLikelihoods: []float64{-10, -5, -1},
			Info: map[string]any{
				"contamination_depths": []float64{},
				"coverage": map[string]any{
					"percent_coverage":   97.92,
					"median_depth":       576.0,
					"min_non_zero_depth": 465.0,
					"kmer_count":         416715,
					"klen":               721,
				},
				"expected_depths": []float64{36},
				"version":         "2",
			},
		},
	}
	p, err := NewTBPredictor(nil, calledGenes, mapPath)
	if err != nil {
		t.Fatal(err)
	}
	got := p.Run().Susceptibility["Erythromycin"]
	if got["predict"] != "R" {
		t.Fatalf("predict=%v", got)
	}
	if mutated := calledGenes["ermC"]; !mutated.ForceVariantField || mutated.Variant != nil {
		t.Fatalf("expected source gene call to be marked for null variant, got %+v", mutated)
	}
}

func TestTBPredictorNamesForAlleleKeepsMutationPrefixesWithHyphens(t *testing.T) {
	p := &TBPredictor{}
	got := p.namesForAllele("eis_G-10A-C2715342T")
	if len(got) == 0 || got[len(got)-1] != "eis_G-10A" {
		t.Fatalf("unexpected names: %v", got)
	}
}

func TestVariantCallJSONKeepsNullVariant(t *testing.T) {
	data, err := json.Marshal(Call{
		Class:               "Call.VariantCall",
		Variant:             nil,
		ForceVariantField:   true,
		Genotype:            []int{1, 1},
		GenotypeLikelihoods: []float64{-1, -2, -3},
		Info:                map[string]any{"contamination_depths": []float64{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, `"variant":null`) {
		t.Fatalf("expected null variant in JSON, got %s", got)
	}
}

func TestSequenceCallJSONKeepsNullVariantWhenForced(t *testing.T) {
	data, err := json.Marshal(Call{
		Class:               "Call.SequenceCall",
		Variant:             nil,
		ForceVariantField:   true,
		Genotype:            []int{1, 1},
		GenotypeLikelihoods: []float64{-1, -2, -3},
		Info:                map[string]any{"contamination_depths": []float64{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, `"variant":null`) {
		t.Fatalf("expected null variant in JSON, got %s", got)
	}
}

func TestVariantCallJSONFormatsIntegralFloatsLikePython(t *testing.T) {
	data, err := json.Marshal(Call{
		Class:               "Call.VariantCall",
		Variant:             "ref-A",
		Genotype:            []int{0, 0},
		GenotypeLikelihoods: []float64{-1.00000000000001, -2},
		Info: map[string]any{
			"expected_depths": []float64{7},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, `"expected_depths":[7.0]`) {
		t.Fatalf("expected python-like float formatting, got %s", got)
	}
	if !strings.Contains(got, `"genotype_likelihoods":[-1.0,-2.0]`) {
		t.Fatalf("expected rounded float formatting, got %s", got)
	}
}

func TestVariantCallJSONDoesNotEscapeAmpersands(t *testing.T) {
	var buf bytes.Buffer
	err := WriteJSONLikePython(&buf, map[string]any{
		"call": Call{
		Class:               "Call.VariantCall",
		Variant:             "ref-A1692141C?var_name=A1692141C&num_alts=1&ref=NC_000962.3&enum=0&gene=NA&mut=A1692141C",
		Genotype:            []int{1, 1},
		GenotypeLikelihoods: []float64{-1, -2, -3},
		Info:                map[string]any{"contamination_depths": []float64{}},
		},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if strings.Contains(got, `\u0026`) {
		t.Fatalf("expected ampersands to remain literal, got %s", got)
	}
	if !strings.Contains(got, `&num_alts=1&ref=NC_000962.3&enum=0&gene=NA&mut=A1692141C`) {
		t.Fatalf("expected literal ampersands in variant, got %s", got)
	}
}
