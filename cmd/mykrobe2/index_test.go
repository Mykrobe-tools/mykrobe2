package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mykrobe-tools/mykrobe2/mykrobe"
)

func TestIndexCommandReportsAliasWarningsOnStderr(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	amr := filepath.Join(dir, "amr.json")
	index := filepath.Join(dir, "custom.panelindex")
	if err := os.WriteFile(panel, []byte(">ref-A10T?gene=geneA&mut=A10T&var_name=C100T\nACGTGCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(amr, []byte(`{"A10T":["drug1"],"geneA_A10T":["drug2"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"index", "--k", "5", "--fasta", panel, "--variant-to-resistance-json", amr, "--output", index})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("warning leaked to stdout: %s", stdout.String())
	}
	for _, want := range []string{"Warning:", "ref-A10T", `"A10T"`, `"geneA_A10T"`, "drug1", "drug2", "prediction applies all matching rules"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("warning lacks %q: %s", want, stderr.String())
		}
	}
	bundle, err := mykrobe.LoadCustomIndex(index)
	if err != nil {
		t.Fatalf("warning prevented usable index: %v", err)
	}
	bundle.Close()
}
