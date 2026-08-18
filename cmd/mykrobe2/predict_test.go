package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mykrobe-tools/mykrobe2/mykrobe/speciesdata"
)

func TestPredictCommand(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	reads := filepath.Join(dir, "reads.fa")
	index := filepath.Join(dir, "custom.panelindex")
	out := filepath.Join(dir, "out.json")
	covgs := filepath.Join(dir, "out.covgs")
	progressPath := filepath.Join(dir, "gui-progress.jsonl")
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
	buildCustomIndexFile(t, index, 5, []string{panel}, filepath.Join(mykrobeTestRefData, "tb_variant_to_resistance_drug.json"), lineage)

	err := run([]string{
		"predict",
		"--sample", "S1",
		"--species", "custom",
		"--seq", reads,
		"--index", index,
		"--output", out,
		"--write_covgs", covgs,
		"--gui-progress-file", progressPath,
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
	covgsData, err := os.ReadFile(covgs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(covgsData), "name\tcolour\tmedian_depth\tmin_depth\tpercent_coverage\tkmer_count\tkmer_length\n") {
		t.Fatalf("missing covgs header: %s", string(covgsData))
	}
	if !strings.Contains(string(covgsData), "katG?name=katG&panel_type=presence&version=1\t0\t1\t1\t1.000000\t7\t6") {
		t.Fatalf("unexpected covgs output: %s", string(covgsData))
	}
	progressData, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatal(err)
	}
	var events []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(progressData)), "\n") {
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode progress event %q: %v", line, err)
		}
		events = append(events, event)
	}
	wantStages := []string{"loading_panel", "processing_reads", "calculating_coverage", "identifying_species", "predicting_resistance", "preparing_results", "complete"}
	stageIndex := 0
	for _, event := range events {
		if stageIndex < len(wantStages) && event["stage"] == wantStages[stageIndex] {
			stageIndex++
		}
	}
	if stageIndex != len(wantStages) {
		t.Fatalf("progress stages = %#v, missing ordered stage %q", events, wantStages[stageIndex])
	}
	last := events[len(events)-1]
	if last["stage"] != "complete" || last["fraction"] != 1.0 || last["determinate"] != true {
		t.Fatalf("final progress event = %#v", last)
	}
}

func TestPredictCommandWithMultipleSeqFiles(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	reads1 := filepath.Join(dir, "reads1.fa")
	reads2 := filepath.Join(dir, "reads2.fa")
	index := filepath.Join(dir, "custom.panelindex")
	out := filepath.Join(dir, "out.json")

	panelData := "" +
		">katG?name=katG&panel_type=presence&version=1\nACGTGCACTA\n" +
		">ref-A123T?var_name=A123T&gene=katG&mut=A123T\nACGTGCACTA\n" +
		">alt-A123T?var_name=A123T&gene=katG&mut=A123T\nTTTTTCACTA\n"
	if err := os.WriteFile(panel, []byte(panelData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reads1, []byte(">r1\nACGTGCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reads2, []byte(">r2\nTTTTTCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	buildCustomIndexFile(t, index, 5, []string{panel}, filepath.Join(mykrobeTestRefData, "tb_variant_to_resistance_drug.json"), "")

	err := run([]string{
		"predict",
		"--sample", "S1",
		"--species", "custom",
		"--seq", reads1,
		"--seq", reads2,
		"--index", index,
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
	files, ok := got["S1"]["files"].([]any)
	if !ok {
		t.Fatalf("missing files list: %v", got["S1"])
	}
	if len(files) != 2 || files[0] != reads1 || files[1] != reads2 {
		t.Fatalf("unexpected files list: %#v", files)
	}
}

func TestPredictCommandWithONTAndConfThresholdFlags(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	reads := filepath.Join(dir, "reads.fa")
	index := filepath.Join(dir, "custom.panelindex")
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
	buildCustomIndexFile(t, index, 5, []string{panel}, filepath.Join(mykrobeTestRefData, "tb_variant_to_resistance_drug.json"), lineage)

	err := run([]string{
		"predict",
		"--sample", "S1",
		"--species", "custom",
		"--seq", reads,
		"--index", index,
		"--output", out,
		"--expected_depth", "100",
		"--ont",
		"--guess_sequence_method",
		"--conf_percent_cutoff", "95",
		"--report_all_calls",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPredictCommandWithCustomIndexGenotypingOnly(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	reads := filepath.Join(dir, "reads.fa")
	index := filepath.Join(dir, "custom.panelindex")
	out := filepath.Join(dir, "out.json")

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
		"index",
		"--fasta", panel,
		"--output", index,
		"--k", "5",
	}); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{
		"predict",
		"--sample", "S1",
		"--species", "custom",
		"--seq", reads,
		"--index", index,
		"--output", out,
	}); err != nil {
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
	s1 := got["S1"]
	if _, ok := s1["variant_calls"]; !ok {
		t.Fatalf("expected variant_calls in genotyping-only custom output: %v", s1)
	}
	susc, ok := s1["susceptibility"].(map[string]any)
	if !ok || len(susc) != 0 {
		t.Fatalf("expected empty susceptibility for genotyping-only custom output: %#v", s1["susceptibility"])
	}
}

func TestPredictCommandWithCustomIndexAndAMR(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	reads := filepath.Join(dir, "reads.fa")
	amr := filepath.Join(dir, "amr.json")
	index := filepath.Join(dir, "custom.panelindex")
	out := filepath.Join(dir, "out.json")

	panelData := "" +
		">katG?name=katG&panel_type=presence&version=1\nACGTGCACTA\n" +
		">ref-A123T?var_name=A123T&gene=katG&mut=A123T\nACGTGCACTA\n" +
		">alt-A123T?var_name=A123T&gene=katG&mut=A123T\nTTTTTCACTA\n"
	if err := os.WriteFile(panel, []byte(panelData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reads, []byte(">r1\nTTTTTCACTA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(amr, []byte(`{"katG_A123T-A123T":["Isoniazid"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{
		"index",
		"--fasta", panel,
		"--variant_to_resistance_json", amr,
		"--output", index,
		"--k", "5",
	}); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{
		"predict",
		"--sample", "S1",
		"--species", "custom",
		"--seq", reads,
		"--index", index,
		"--output", out,
		"--report_all_calls",
	}); err != nil {
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
	s1 := got["S1"]
	susc, ok := s1["susceptibility"].(map[string]any)
	if !ok || len(susc) == 0 {
		t.Fatalf("expected susceptibility predictions in custom indexed output: %#v", s1["susceptibility"])
	}
}

func TestCustomWorkflowMakeProbesToIndexToPredictWithLineage(t *testing.T) {
	dir := t.TempDir()
	refPath := filepath.Join(dir, "ref.fa")
	varsPath := filepath.Join(dir, "vars.tsv")
	probesPath := filepath.Join(dir, "probes.fa")
	lineagePath := filepath.Join(dir, "lineage.json")
	indexPath := filepath.Join(dir, "custom.panelindex")
	readsPath := filepath.Join(dir, "reads.fa")
	outPath := filepath.Join(dir, "out.json")

	refSeq := []byte(strings.Repeat("A", 80))
	refSeq[41] = 'G'
	if err := os.WriteFile(refPath, []byte(">ref\n"+string(refSeq)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(varsPath, []byte("ref\t42\tG\tA\tDNA\t*lineage1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldStdout := os.Stdout
	f, err := os.Create(probesPath)
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = f
	err = run([]string{
		"make-probes",
		refPath,
		"--text_file", varsPath,
		"--lineage", lineagePath,
		"--kmer", "5",
	})
	os.Stdout = oldStdout
	if closeErr := f.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatal(err)
	}

	if err := run([]string{
		"index",
		"--fasta", probesPath,
		"--lineage_json", lineagePath,
		"--output", indexPath,
		"--k", "5",
	}); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(readsPath, []byte(">r1\n"+string(refSeq[37:47])+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{
		"predict",
		"--sample", "S1",
		"--species", "custom",
		"--seq", readsPath,
		"--index", indexPath,
		"--output", outPath,
		"--report_all_calls",
	}); err != nil {
		t.Fatal(err)
	}

	got := map[string]map[string]any{}
	fh, err := os.Open(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	if err := json.NewDecoder(fh).Decode(&got); err != nil {
		t.Fatal(err)
	}
	s1 := got["S1"]
	phylo, ok := s1["phylogenetics"].(map[string]any)
	if !ok {
		t.Fatalf("missing phylogenetics: %#v", s1)
	}
	lineage, ok := phylo["lineage"].(map[string]any)
	if !ok || len(lineage) == 0 {
		t.Fatalf("expected lineage call in custom workflow output: %#v", phylo)
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

func TestPredictCommandWithPanelsDirUsesRuntimeIndex(t *testing.T) {
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
	panelFile := filepath.Join(tbDir, "panel.fa")
	if err := os.WriteFile(panelFile, []byte(panelData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tbDir, "amr.json"), []byte(`{"katG_A123T-A123T":["Isoniazid"]}`), 0o644); err != nil {
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

	ddir, err := speciesdata.NewDataDir(panelsDir)
	if err != nil {
		t.Fatal(err)
	}
	sdir, err := ddir.GetSpeciesDir("tb")
	if err != nil {
		t.Fatal(err)
	}
	if sdir == nil {
		t.Fatal("expected installed species dir")
	}
	if err := sdir.BuildRuntimeIndex(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(panelFile); err != nil {
		t.Fatal(err)
	}

	err = run([]string{
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
	if _, ok := got["S1"]; !ok {
		t.Fatalf("missing sample in output: %v", got)
	}
}

func TestPredictCommandCSV(t *testing.T) {
	dir := t.TempDir()
	panel := filepath.Join(dir, "panel.fa")
	reads := filepath.Join(dir, "reads.fa")
	index := filepath.Join(dir, "custom.panelindex")
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
	buildCustomIndexFile(t, index, 5, []string{panel}, filepath.Join(mykrobeTestRefData, "tb_variant_to_resistance_drug.json"), "")
	if err := run([]string{
		"predict",
		"--sample", "S1",
		"--species", "custom",
		"--seq", reads,
		"--index", index,
		"--output", out,
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
