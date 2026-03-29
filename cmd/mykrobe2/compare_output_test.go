package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompareOutputMatchesWithinTolerance(t *testing.T) {
	dir := t.TempDir()
	left := dir + "/left.json"
	right := dir + "/right.json"
	writeJSONFile(t, left, map[string]any{
		"SAMPLE": map[string]any{
			"susceptibility": map[string]any{
				"Isoniazid": map[string]any{"predict": "R"},
			},
			"phylogenetics": map[string]any{
				"species": map[string]any{"species": []any{"Mycobacterium tuberculosis"}},
			},
		},
	})
	writeJSONFile(t, right, map[string]any{
		"SAMPLE": map[string]any{
			"susceptibility": map[string]any{
				"Isoniazid": map[string]any{"predict": "R"},
			},
			"phylogenetics": map[string]any{
				"species": map[string]any{"species": []any{"Mycobacterium tuberculosis"}},
			},
		},
	})
	var out bytes.Buffer
	err := runCompareOutput(&compareOutputOptions{floatTolerance: defaultCompareFloatTolerance}, &out, left, right)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Outputs match within tolerance.") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestCompareOutputReportAllCallsMismatch(t *testing.T) {
	dir := t.TempDir()
	left := dir + "/left.json"
	right := dir + "/right.json"
	writeJSONFile(t, left, map[string]any{
		"SAMPLE": map[string]any{
			"susceptibility": map[string]any{},
		},
	})
	writeJSONFile(t, right, map[string]any{
		"SAMPLE": map[string]any{
			"susceptibility": map[string]any{},
			"variant_calls":  map[string]any{},
		},
	})
	err := runCompareOutput(&compareOutputOptions{floatTolerance: defaultCompareFloatTolerance}, &bytes.Buffer{}, left, right)
	if err == nil || !strings.Contains(err.Error(), "report_all_calls mismatch") {
		t.Fatalf("expected report_all_calls mismatch, got %v", err)
	}
}

func TestCompareOutputReportsMeaningfulDiffs(t *testing.T) {
	dir := t.TempDir()
	left := dir + "/left.json"
	right := dir + "/right.json"
	writeJSONFile(t, left, map[string]any{
		"SAMPLE": map[string]any{
			"susceptibility": map[string]any{
				"Isoniazid": map[string]any{"predict": "R"},
			},
			"variant_calls": map[string]any{
				"var1": map[string]any{
					"genotype_likelihoods": []any{-10.0, -2.0},
				},
			},
		},
	})
	writeJSONFile(t, right, map[string]any{
		"SAMPLE": map[string]any{
			"susceptibility": map[string]any{
				"Isoniazid": map[string]any{"predict": "S"},
			},
			"variant_calls": map[string]any{
				"var1": map[string]any{
					"genotype_likelihoods": []any{-10.0, -2.5},
				},
			},
		},
	})
	var out bytes.Buffer
	err := runCompareOutput(&compareOutputOptions{floatTolerance: defaultCompareFloatTolerance}, &out, left, right)
	if err == nil {
		t.Fatal("expected differences")
	}
	got := out.String()
	if !strings.Contains(got, "susceptibility differences") {
		t.Fatalf("missing susceptibility section: %s", got)
	}
	if !strings.Contains(got, "variant_calls differences") {
		t.Fatalf("missing variant_calls section: %s", got)
	}
	if !strings.Contains(got, "root.SAMPLE.susceptibility.Isoniazid.predict") {
		t.Fatalf("missing predict diff path: %s", got)
	}
}

func TestCompareOutputIgnoresTinyFloatDiffs(t *testing.T) {
	dir := t.TempDir()
	left := dir + "/left.json"
	right := dir + "/right.json"
	writeJSONFile(t, left, map[string]any{
		"SAMPLE": map[string]any{
			"variant_calls": map[string]any{
				"var1": map[string]any{
					"genotype_likelihoods": []any{-1176.51688736382},
				},
			},
		},
	})
	writeJSONFile(t, right, map[string]any{
		"SAMPLE": map[string]any{
			"variant_calls": map[string]any{
				"var1": map[string]any{
					"genotype_likelihoods": []any{-1176.51688736383},
				},
			},
		},
	})
	var out bytes.Buffer
	err := runCompareOutput(&compareOutputOptions{floatTolerance: defaultCompareFloatTolerance}, &out, left, right)
	if err != nil {
		t.Fatalf("expected tiny float difference to be ignored, got %v with output %s", err, out.String())
	}
}
