package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/martinghunt/mykrobe2/mykrobe/speciesdata"
)

func TestPredictCommand(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	reads := filepath.Join(dir, "reads.fa")
	out := filepath.Join(dir, "out.json")
	lineage := filepath.Join(dir, "lineage.json")

	panelData := "" +
		">katG?name=katG&panel_type=presence&version=1\nACGTGCACTA\n" +
		">ref-A123T?var_name=A123T&gene=katG&mut=A123T\nACGTGCACTA\n" +
		">alt-A123T?var_name=A123T&gene=katG&mut=A123T\nTTTTTCACTA\n"
	if err := os.WriteFile(panel, []byte(panelData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reads, []byte(">r1\nACGTGCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lineage, []byte(`{"A123T":{"name":"lineage1","use_ref_allele":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"predict",
		"--sample", "S1",
		"--seq", reads,
		"--panel", panel,
		"--variant_to_resistance_json", "/Users/martin/git/mykrobe/tests/ref_data/tb_variant_to_resistance_drug.json",
		"--lineage_json", lineage,
		"--output", out,
		"--k", "5",
		"--expected_depth", "100",
		"--report_all_calls",
	})
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]map[string]any{}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["S1"]; !ok {
		t.Fatalf("missing sample in output: %v", got)
	}
	s1 := got["S1"]
	if _, ok := s1["phylogenetics"]; !ok {
		t.Fatalf("missing phylogenetics: %v", s1)
	}
	if _, ok := s1["variant_calls"]; !ok {
		t.Fatalf("missing variant_calls: %v", s1)
	}
}

func TestPredictCommandWithPanelsDir(t *testing.T) {
	dir := t.TempDir()
	reads := filepath.Join(dir, "reads.fa")
	out := filepath.Join(dir, "out.json")
	panelsDir := filepath.Join(dir, "panels")
	if err := os.WriteFile(reads, []byte(">r1\nACGTGCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tbDir := filepath.Join(panelsDir, "tb")
	if err := os.MkdirAll(tbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	panelData := "" +
		">katG?name=katG&panel_type=presence&version=1\nACGTGCACTA\n" +
		">ref-A123T?var_name=A123T&gene=katG&mut=A123T\nACGTGCACTA\n" +
		">alt-A123T?var_name=A123T&gene=katG&mut=A123T\nTTTTTCACTA\n"
	if err := os.WriteFile(filepath.Join(tbDir, "panel.fa"), []byte(panelData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tbDir, "amr.json"), []byte(`{"katG_A123T":["Isoniazid"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tbDir, "lineage.json"), []byte(`{"katG_A123T-A123T":{"name":"lineage1","use_ref_allele":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	panelName := "202010"
	manifest := speciesdata.SpeciesManifest{
		SpeciesName:  "tb",
		Version:      "20240214",
		DefaultPanel: panelName,
		Panels: map[string]speciesdata.PanelManifest{
			panelName: {
				Description:       "tb panel",
				ReferenceGenome:   "NC_000962.3",
				SpeciesPhyloGroup: "mtbc",
				FASTAFiles:        []string{"panel.fa"},
				Kmer:              5,
				JSONFiles: map[string]*string{
					"amr":       strPtr("amr.json"),
					"lineage":   strPtr("lineage.json"),
					"hierarchy": nil,
				},
			},
		},
	}
	writeJSONFile(t, filepath.Join(tbDir, "manifest.json"), manifest)
	writeJSONFile(t, filepath.Join(panelsDir, "manifest.json"), map[string]map[string]map[string]string{
		"tb": {
			"installed": {"version": "20240214", "url": "local"},
			"latest":    {"version": "20240214", "url": "local"},
		},
	})

	err := run([]string{
		"predict",
		"--sample", "S1",
		"--seq", reads,
		"--species", "tb",
		"--panel", panelName,
		"--panels_dir", panelsDir,
		"--output", out,
		"--expected_depth", "100",
		"--report_all_calls",
	})
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]map[string]any{}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&got); err != nil {
		t.Fatal(err)
	}
	s1, ok := got["S1"]
	if !ok {
		t.Fatalf("missing sample in output: %v", got)
	}
	phylo, ok := s1["phylogenetics"].(map[string]any)
	if !ok {
		t.Fatalf("missing phylogenetics object: %T", s1["phylogenetics"])
	}
	if _, ok := phylo["lineage"]; !ok {
		t.Fatalf("missing lineage in phylogenetics: %v", phylo)
	}
	if _, ok := s1["probe_sets"]; !ok {
		t.Fatalf("missing probe_sets metadata: %v", s1)
	}
}

func TestPanelsCommands(t *testing.T) {
	dir := t.TempDir()
	panelsDir := filepath.Join(dir, "panels")
	speciesTar := makeSpeciesTarball(t, "tb", "20240214", "202010")
	manifestPath := filepath.Join(dir, "manifest.json")
	writeJSONFile(t, manifestPath, map[string]map[string]string{
		"tb": {"version": "20240214", "url": speciesTar},
	})

	if err := run([]string{
		"panels", "update_metadata",
		"--panels_dir", panelsDir,
		"--manifest_file", manifestPath,
	}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{
		"panels", "update_species",
		"--panels_dir", panelsDir,
		"tb",
	}); err != nil {
		t.Fatal(err)
	}

	ddir, err := speciesdata.NewDataDir(panelsDir)
	if err != nil {
		t.Fatal(err)
	}
	if !ddir.SpeciesIsInstalled("tb") {
		t.Fatalf("expected tb to be installed: %+v", ddir.Manifest)
	}
	sdir, err := ddir.GetSpeciesDir("tb")
	if err != nil {
		t.Fatal(err)
	}
	if sdir == nil || sdir.DefaultPanel() != "202010" {
		t.Fatalf("unexpected species dir after install: %#v", sdir)
	}
}

func TestPredictCommandCSV(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	reads := filepath.Join(dir, "reads.fa")
	out := filepath.Join(dir, "out.csv")
	panelData := "" +
		">katG?name=katG&panel_type=presence&version=1\nACGTGCACTA\n" +
		">ref-A123T?var_name=A123T&gene=katG&mut=A123T\nACGTGCACTA\n" +
		">alt-A123T?var_name=A123T&gene=katG&mut=A123T\nTTTTTCACTA\n"
	if err := os.WriteFile(panel, []byte(panelData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reads, []byte(">r1\nACGTGCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{
		"predict",
		"--sample", "S1",
		"--seq", reads,
		"--panel", panel,
		"--variant_to_resistance_json", "/Users/martin/git/mykrobe/tests/ref_data/tb_variant_to_resistance_drug.json",
		"--output", out,
		"--k", "5",
		"--expected_depth", "100",
		"--format", "csv",
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "sample,drug,predict") {
		t.Fatalf("unexpected csv output: %s", string(data))
	}
}

func writeJSONFile(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func strPtr(s string) *string { return &s }

func makeSpeciesTarball(t *testing.T, species, version, panel string) string {
	t.Helper()
	base := filepath.Join(t.TempDir(), species+"_data")
	root := filepath.Join(base, "mykrobe.panel."+species+"."+version)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	panelData := "" +
		">katG?name=katG&panel_type=presence&version=1\nACGTGCACTA\n" +
		">ref-A123T?var_name=A123T&gene=katG&mut=A123T\nACGTGCACTA\n" +
		">alt-A123T?var_name=A123T&gene=katG&mut=A123T\nTTTTTCACTA\n"
	writeGzipFile(t, filepath.Join(root, "panel.fa.gz"), []byte(panelData))
	writeGzipFile(t, filepath.Join(root, "amr.json.gz"), []byte(`{"katG_A123T-A123T":["Isoniazid"]}`))
	writeJSONFile(t, filepath.Join(root, "lineage.json"), map[string]map[string]any{
		"katG_A123T-A123T": {"name": "lineage1", "use_ref_allele": true},
	})
	manifest := speciesdata.SpeciesManifest{
		SpeciesName:  species,
		Version:      version,
		DefaultPanel: panel,
		Panels: map[string]speciesdata.PanelManifest{
			panel: {
				Description:       "tb panel",
				ReferenceGenome:   "NC_000962.3",
				SpeciesPhyloGroup: "mtbc",
				FASTAFiles:        []string{"panel.fa.gz"},
				Kmer:              5,
				JSONFiles: map[string]*string{
					"amr":       strPtr("amr.json.gz"),
					"lineage":   strPtr("lineage.json"),
					"hierarchy": nil,
				},
			},
		},
	}
	writeJSONFile(t, filepath.Join(root, "manifest.json"), manifest)

	tarball := filepath.Join(t.TempDir(), species+"."+version+".tar.gz")
	f, err := os.Create(tarball)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	if err := addTreeToTar(tw, base); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return tarball
}

func addTreeToTar(tw *tar.Writer, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			hdr.Name += "/"
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tw.Write(data)
		return err
	})
}

func writeGzipFile(t *testing.T, path string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	if _, err := gw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
