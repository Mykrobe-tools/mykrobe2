package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mykrobe-tools/mykrobe2/mykrobe/speciesdata"
)

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
	if _, err := os.Stat(sdir.RuntimeIndexFile()); err != nil {
		t.Fatalf("expected runtime index to be built: %v", err)
	}
}

func TestPanelsCommandsUseDefaultPanelsDir(t *testing.T) {
	home := t.TempDir()
	oldHome, hadHome := os.LookupEnv("HOME")
	oldData, hadData := os.LookupEnv("MYKROBE_DATA_HOME")
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatal(err)
	}
	if err := os.Unsetenv("MYKROBE_DATA_HOME"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if hadHome {
			_ = os.Setenv("HOME", oldHome)
		} else {
			_ = os.Unsetenv("HOME")
		}
		if hadData {
			_ = os.Setenv("MYKROBE_DATA_HOME", oldData)
		} else {
			_ = os.Unsetenv("MYKROBE_DATA_HOME")
		}
	}()

	panelsDir := defaultPanelsDir()
	speciesTar := makeSpeciesTarball(t, "tb", "20240214", "202010")
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	writeJSONFile(t, manifestPath, map[string]map[string]string{
		"tb": {"version": "20240214", "url": speciesTar},
	})

	if err := run([]string{
		"panels", "update_metadata",
		"--manifest_file", manifestPath,
	}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{
		"panels", "update_species",
		"tb",
	}); err != nil {
		t.Fatal(err)
	}

	ddir, err := speciesdata.NewDataDir(panelsDir)
	if err != nil {
		t.Fatal(err)
	}
	if !ddir.SpeciesIsInstalled("tb") {
		t.Fatalf("expected tb to be installed in default panels dir: %+v", ddir.Manifest)
	}
	sdir, err := ddir.GetSpeciesDir("tb")
	if err != nil {
		t.Fatal(err)
	}
	if sdir == nil {
		t.Fatal("expected species dir in default panels dir")
	}
	if _, err := os.Stat(sdir.RuntimeIndexFile()); err != nil {
		t.Fatalf("expected runtime index in default panels dir: %v", err)
	}
}

func TestPanelsDescribeJSON(t *testing.T) {
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

	cmd := newRootCmd()
	var output strings.Builder
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"panels", "describe", "--panels_dir", panelsDir, "--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var got struct {
		PanelsDir string `json:"panels_dir"`
		Species   []struct {
			Species      string `json:"species"`
			Installed    bool   `json:"installed"`
			DefaultPanel string `json:"default_panel"`
			Panels       []struct {
				Name string `json:"name"`
			} `json:"panels"`
		} `json:"species"`
	}
	if err := json.Unmarshal([]byte(output.String()), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Species) != 1 || got.Species[0].Species != "tb" {
		t.Fatalf("unexpected species output: %+v", got.Species)
	}
	if !got.Species[0].Installed || got.Species[0].DefaultPanel != "202010" {
		t.Fatalf("unexpected installed/default data: %+v", got.Species[0])
	}
	if len(got.Species[0].Panels) != 1 || got.Species[0].Panels[0].Name != "202010" {
		t.Fatalf("unexpected panel list: %+v", got.Species[0].Panels)
	}
}

func TestPanelsDescribeText(t *testing.T) {
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

	cmd := newRootCmd()
	var output strings.Builder
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"panels", "describe", "--panels_dir", panelsDir})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "Species summary:") || !strings.Contains(text, "tb") {
		t.Fatalf("unexpected describe text output:\n%s", text)
	}
}

func TestPanelsDescribePanelsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	panelsDir := filepath.Join(dir, "panels")
	speciesDir := filepath.Join(dir, "species-src")
	if err := os.MkdirAll(panelsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(speciesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONFile(t, filepath.Join(speciesDir, "manifest.json"), map[string]any{
		"species_name":  "tb",
		"version":       "20240214",
		"default_panel": "202309",
		"panels": map[string]any{
			"bradley-2015": map[string]any{
				"description":         "legacy bradley",
				"reference_genome":    "ref-bradley",
				"species_phylo_group": "mtbc",
				"fasta_files":         []string{"bradley.fa.gz"},
				"kmer":                21,
				"json_files":          map[string]any{"amr": "bradley.json.gz"},
			},
			"walker-2015": map[string]any{
				"description":         "legacy walker",
				"reference_genome":    "ref-walker",
				"species_phylo_group": "mtbc",
				"fasta_files":         []string{"walker.fa.gz"},
				"kmer":                21,
				"json_files":          map[string]any{"amr": "walker.json.gz"},
			},
			"202001": map[string]any{
				"description":         "older",
				"reference_genome":    "ref-old",
				"species_phylo_group": "mtbc",
				"fasta_files":         []string{"probes1.fa.gz"},
				"kmer":                21,
				"json_files":          map[string]any{"amr": "amr1.json.gz"},
			},
			"202309": map[string]any{
				"description":         "newer",
				"reference_genome":    "ref-new",
				"species_phylo_group": "mtbc",
				"fasta_files":         []string{"probes2.fa.gz"},
				"kmer":                21,
				"json_files":          map[string]any{"amr": "amr2.json.gz"},
			},
		},
	})
	writeJSONFile(t, filepath.Join(panelsDir, "manifest.json"), map[string]any{
		"tb": map[string]any{
			"installed": map[string]string{"version": "20240214", "url": "file:///tmp/tb.tar.gz"},
			"latest":    map[string]string{"version": "20240214", "url": "file:///tmp/tb.tar.gz"},
		},
	})
	if err := os.Rename(speciesDir, filepath.Join(panelsDir, "tb")); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	var output strings.Builder
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"panels", "describe", "--panels_dir", panelsDir, "--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var got struct {
		Species []struct {
			Panels []struct {
				Name string `json:"name"`
			} `json:"panels"`
		} `json:"species"`
	}
	if err := json.Unmarshal([]byte(output.String()), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Species) != 1 || len(got.Species[0].Panels) != 4 {
		t.Fatalf("unexpected panel output: %+v", got)
	}
	gotNames := []string{
		got.Species[0].Panels[0].Name,
		got.Species[0].Panels[1].Name,
		got.Species[0].Panels[2].Name,
		got.Species[0].Panels[3].Name,
	}
	wantNames := []string{"202309", "202001", "bradley-2015", "walker-2015"}
	if strings.Join(gotNames, ",") != strings.Join(wantNames, ",") {
		t.Fatalf("expected newest first, got %+v", got.Species[0].Panels)
	}
}
