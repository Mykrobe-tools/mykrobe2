package mykrobe

import "testing"

func TestAMRSpeciesPredictorMixedMTBCNTM(t *testing.T) {
	p, err := NewAMRSpeciesPredictor(nil, nil, nil, nil, "/Users/martin/git/mykrobe/tests/ref_data/mtbc_hierarchy.json")
	if err != nil {
		t.Fatal(err)
	}
	p.OutJSON["phylogenetics"] = map[string]any{
		"phylo_group": map[string]map[string]any{
			"Non_tuberculosis_mycobacterium_complex": {"percent_coverage": 58.71542975006994, "median_depth": 36},
			"Mycobacterium_tuberculosis_complex":     {"percent_coverage": 62.81850563578579, "median_depth": 2},
		},
	}
	if !p.IsMTBCPresent() || !p.IsNTMPresent() {
		t.Fatal(p.OutJSON)
	}
	if got := len(p.getPresentPhyloGroups(p.OutJSON["phylogenetics"].(map[string]any)["phylo_group"].(map[string]map[string]any), 50)); got != 2 {
		t.Fatal(got)
	}
}

func TestAMRSpeciesPredictorGetBestCoverageDict(t *testing.T) {
	p, err := NewAMRSpeciesPredictor(nil, nil, nil, nil, "/Users/martin/git/mykrobe/tests/ref_data/mtbc_hierarchy.json")
	if err != nil {
		t.Fatal(err)
	}
	best := p.getBestCoverageDict(map[string]map[string]any{
		"Mycobacterium_chimaera":       {"percent_coverage": 99.162, "median_depth": 39},
		"Mycobacterium_intracellulare": {"percent_coverage": 98.662, "median_depth": 45},
		"Mycobacterium_bovis":          {"percent_coverage": 9.894, "median_depth": 12.0},
	})
	if len(best) != 1 {
		t.Fatal(best)
	}
	if _, ok := best["Mycobacterium_chimaera"]; !ok {
		t.Fatal(best)
	}
}

func TestAMRSpeciesPredictorMixedChimera(t *testing.T) {
	p, err := NewAMRSpeciesPredictor(nil, nil, nil, nil, "/Users/martin/git/mykrobe/tests/ref_data/mtbc_hierarchy.json")
	if err != nil {
		t.Fatal(err)
	}
	phylo := map[string]map[string]map[string]any{
		"sub_complex": {
			"Mycobacterium_avium_complex": {"percent_coverage": 98.346, "median_depth": 54.0},
		},
		"phylo_group": {
			"Non_tuberculosis_mycobacterium_complex": {"percent_coverage": 82.846, "median_depth": 49},
		},
		"species": {
			"Mycobacterium_chimaera":       {"percent_coverage": 99.162, "median_depth": 39},
			"Mycobacterium_intracellulare": {"percent_coverage": 98.662, "median_depth": 45},
			"Mycobacterium_bovis":          {"percent_coverage": 9.894, "median_depth": 12.0},
		},
		"lineage": {},
	}
	out := p.ChooseBest(phylo)
	if _, ok := out["species"]["Mycobacterium_chimaera"]; !ok {
		t.Fatal(out)
	}
	if _, ok := out["species"]["Mycobacterium_intracellulare"]; !ok {
		t.Fatal(out)
	}
	if _, ok := out["species"]["Mycobacterium_bovis"]; ok {
		t.Fatal(out)
	}
}

func TestDetectSpeciesAndGetDepths(t *testing.T) {
	set := &CoverageSet{
		Variant:  map[string]*VariantProbeCoverage{},
		Presence: map[string]map[string]SequenceProbeCoverage{},
		Groups: map[string]map[string]*TaxonCoverage{
			"species": {
				"Mycobacterium_chimaera": {
					TotalBases:      200,
					PercentCoverage: []float64{100, 98},
					Length:          []int{100, 100},
					Median:          []float64{39, 39},
				},
				"Mycobacterium_intracellulare": {
					TotalBases:      200,
					PercentCoverage: []float64{99, 98},
					Length:          []int{100, 100},
					Median:          []float64{45, 45},
				},
			},
		},
	}

	phylo, depths, err := DetectSpeciesAndGetDepths(set, "/Users/martin/git/mykrobe/tests/ref_data/mtbc_hierarchy.json", "Non_tuberculosis_mycobacterium_complex")
	if err != nil {
		t.Fatal(err)
	}
	if len(depths) != 1 || depths[0] == 0 {
		t.Fatalf("unexpected depths: %v", depths)
	}
	species := phylo["species"].(map[string]map[string]any)
	if _, ok := species["Mycobacterium_intracellulare"]; !ok {
		t.Fatalf("missing expected species call: %v", phylo)
	}
	phyloGroups := phylo["phylo_group"].(map[string]map[string]any)
	if _, ok := phyloGroups["Non_tuberculosis_mycobacterium_complex"]; !ok {
		t.Fatalf("missing inferred phylo group: %v", phylo)
	}
}
