package mykrobe

import (
	"os"
	"path/filepath"
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
