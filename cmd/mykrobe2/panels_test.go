package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/martinghunt/mykrobe2/mykrobe/speciesdata"
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
	if _, err := os.Stat(sdir.PanelIndexFile()); err != nil {
		t.Fatalf("expected panel index to be built: %v", err)
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
	if _, err := os.Stat(sdir.PanelIndexFile()); err != nil {
		t.Fatalf("expected panel index in default panels dir: %v", err)
	}
}
