package mykrobe

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAndLoadCustomIndexWithOptionalJSONSidecars(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	amr := filepath.Join(dir, "amr.json")
	lineage := filepath.Join(dir, "lineage.json")
	index := filepath.Join(dir, "custom.panelindex")

	panelData := "" +
		">katG?name=katG&panel_type=presence&version=1\nACGTGCACTA\n" +
		">ref-A123T?var_name=A123T&gene=katG&mut=A123T\nACGTGCACTA\n" +
		">alt-A123T?var_name=A123T&gene=katG&mut=A123T\nTTTTTCACTA\n"
	if err := os.WriteFile(panel, []byte(panelData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(amr, []byte(`{"katG_A123T-A123T":["Isoniazid"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lineage, []byte(`{"A123T":{"name":"lineage1","use_ref_allele":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := BuildCustomIndex(index, 5, []string{panel}, amr, lineage); err != nil {
		t.Fatal(err)
	}
	bundle, err := LoadCustomIndex(index)
	if err != nil {
		t.Fatal(err)
	}
	defer bundle.Close()

	if len(bundle.VariantToResistance) == 0 {
		t.Fatal("expected bundled amr JSON")
	}
	if len(bundle.Lineage) == 0 {
		t.Fatal("expected bundled lineage JSON")
	}
}

func TestCustomIndexMappingAliasWarnings(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mapping string
		warn    bool
	}{
		{"single alias", `{"A10T":["drug1"]}`, false},
		{"same sets", `{"A10T":["drug1","drug2"],"geneA_A10T":["drug2","drug1","drug1"]}`, false},
		{"overlapping sets", `{"A10T":["drug1"],"geneA_A10T":["drug1","drug2"]}`, true},
		{"disjoint sets", `{"A10T":["drug1"],"geneA_A10T":["drug2"]}`, true},
		{"lowercase fallback", `{"a10t":["drug1"],"genea_a10t":["drug2"]}`, true},
		{"unmatched rule", `{"A10T":["drug1"],"geneB_A10T":["drug2"]}`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			panel := filepath.Join(dir, "panel.fa")
			amr := filepath.Join(dir, "amr.json")
			index := filepath.Join(dir, "custom.panelindex")
			// Reference, alternate and enumerated probes share these aliases.
			probes := ">ref-A10T?gene=geneA&mut=A10T&var_name=C100T&enum=0\nACGTGCACTA\n" +
				">alt-A10T?gene=geneA&mut=A10T&var_name=C100T&enum=0\nTTTTTCACTA\n" +
				">ref-A10T?gene=geneA&mut=A10T&var_name=C100T&enum=1\nACGTGCACTA\n"
			if err := os.WriteFile(panel, []byte(probes), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(amr, []byte(tt.mapping), 0o644); err != nil {
				t.Fatal(err)
			}
			var warnings []MappingAliasWarning
			if err := BuildCustomIndexWithWarnings(index, 5, []string{panel}, amr, "", func(w MappingAliasWarning) {
				warnings = append(warnings, w)
			}); err != nil {
				t.Fatal(err)
			}
			wantWarnings := 0
			if tt.warn {
				wantWarnings = 1
			}
			if len(warnings) != wantWarnings {
				t.Fatalf("warnings = %v, want %d", warnings, wantWarnings)
			}
			if tt.warn {
				w := warnings[0]
				if len(w.Aliases) != 2 || len(w.Aliases[0].Drugs) == 0 || !strings.Contains(w.String(), "prediction applies all matching rules") {
					t.Fatalf("warning lacks mapping details: %+v", w)
				}
			}
			bundle, err := LoadCustomIndex(index)
			if err != nil {
				t.Fatal(err)
			}
			defer bundle.Close()
			if !bytes.Equal(bundle.VariantToResistance, []byte(tt.mapping)) {
				t.Fatalf("warning changed embedded mapping: %s", bundle.VariantToResistance)
			}
		})
	}
}

func TestCustomIndexWithoutMappingDoesNotWarn(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	if err := os.WriteFile(panel, []byte(">ref-A10T?gene=geneA&mut=A10T\nACGTGCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := BuildCustomIndexWithWarnings(filepath.Join(dir, "custom.panelindex"), 5, []string{panel}, "", "", func(w MappingAliasWarning) {
		t.Errorf("unexpected warning without AMR mapping: %v", w)
	}); err != nil {
		t.Fatal(err)
	}
}
