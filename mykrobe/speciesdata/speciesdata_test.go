package speciesdata

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSpeciesDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "species")
	if _, err := NewSpeciesDir(root); err == nil {
		t.Fatal("expected missing species dir error")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewSpeciesDir(root); err == nil {
		t.Fatal("expected missing manifest error")
	}

	manifest := SpeciesManifest{
		SpeciesName:  "species_name",
		Version:      "20200821",
		DefaultPanel: "panel1",
		Panels: map[string]PanelManifest{
			"panel1": {
				Description:       "description of panel1",
				ReferenceGenome:   "NC42",
				SpeciesPhyloGroup: "species_xyz",
				FASTAFiles:        []string{"probes.fa"},
				Kmer:              21,
				JSONFiles: map[string]*string{
					"amr":       ptr("amr.json"),
					"lineage":   nil,
					"hierarchy": nil,
				},
			},
		},
	}
	writeJSON(t, filepath.Join(root, "manifest.json"), manifest)

	sdir, err := NewSpeciesDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if sdir.SpeciesName() != manifest.SpeciesName || sdir.Version() != manifest.Version {
		t.Fatalf("unexpected species metadata: %+v", sdir.Manifest)
	}
	if sdir.PanelName != manifest.DefaultPanel {
		t.Fatalf("unexpected default panel: %s", sdir.PanelName)
	}
	if !reflect.DeepEqual(sdir.PanelNames(), []string{"panel1"}) {
		t.Fatalf("unexpected panel names: %v", sdir.PanelNames())
	}
	if sdir.Kmer() != 21 || sdir.SpeciesPhyloGroup() != "species_xyz" {
		t.Fatalf("unexpected panel values: %+v", sdir.Panel)
	}
	wantFasta := []string{filepath.Join(root, "probes.fa")}
	if !reflect.DeepEqual(sdir.FASTAFiles(), wantFasta) {
		t.Fatalf("unexpected fasta files: %v", sdir.FASTAFiles())
	}
	if err := os.WriteFile(wantFasta[0], []byte(">x\nAAAAA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	amrJSON := sdir.JSONFile("amr")
	if amrJSON != filepath.Join(root, "amr.json") {
		t.Fatalf("unexpected amr json path: %s", amrJSON)
	}
	if err := os.WriteFile(amrJSON, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !sdir.SanityCheck() {
		t.Fatal("expected sanity check to pass")
	}

	panel1 := sdir.Manifest.Panels["panel1"]
	panel1.FASTAFiles = []string{}
	sdir.Manifest.Panels["panel1"] = panel1
	if sdir.SanityCheck() {
		t.Fatal("expected empty fasta list to fail sanity check")
	}
	panel1.FASTAFiles = nil
	sdir.Manifest.Panels["panel1"] = panel1
	if sdir.SanityCheck() {
		t.Fatal("expected nil fasta list to fail sanity check")
	}
	panel1.FASTAFiles = []string{"probes.oops.fa"}
	sdir.Manifest.Panels["panel1"] = panel1
	if sdir.SanityCheck() {
		t.Fatal("expected missing fasta to fail sanity check")
	}
	panel1.FASTAFiles = []string{"probes.fa"}
	sdir.Manifest.Panels["panel1"] = panel1
	if !sdir.SanityCheck() {
		t.Fatal("expected restored fasta to pass sanity check")
	}
	if err := os.Remove(wantFasta[0]); err != nil {
		t.Fatal(err)
	}
	if sdir.SanityCheck() {
		t.Fatal("expected removed fasta to fail sanity check")
	}
	if err := os.WriteFile(wantFasta[0], []byte(">x\nAAAAA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !sdir.SanityCheck() {
		t.Fatal("expected recreated fasta to pass sanity check")
	}

	panel1 = sdir.Manifest.Panels["panel1"]
	panel1.JSONFiles["amr"] = nil
	sdir.Manifest.Panels["panel1"] = panel1
	if sdir.SanityCheck() {
		t.Fatal("expected missing json set to fail sanity check")
	}
	panel1.JSONFiles["amr"] = ptr("does-not-exist.json")
	sdir.Manifest.Panels["panel1"] = panel1
	if sdir.SanityCheck() {
		t.Fatal("expected absent json to fail sanity check")
	}
	panel1.JSONFiles["amr"] = ptr("amr.json")
	sdir.Manifest.Panels["panel1"] = panel1
	if !sdir.SanityCheck() {
		t.Fatal("expected valid json path to pass sanity check")
	}
	if err := os.Remove(amrJSON); err != nil {
		t.Fatal(err)
	}
	if sdir.SanityCheck() {
		t.Fatal("expected removed json to fail sanity check")
	}
	if err := os.WriteFile(amrJSON, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !sdir.SanityCheck() {
		t.Fatal("expected recreated json to pass sanity check")
	}

	manifest.Panels["panel2"] = PanelManifest{
		Description:       "description of panel2",
		ReferenceGenome:   "NC42",
		SpeciesPhyloGroup: "species_xyz",
		FASTAFiles:        []string{"probes2.fa"},
		Kmer:              31,
		JSONFiles: map[string]*string{
			"amr":       ptr("amr2.json"),
			"lineage":   nil,
			"hierarchy": nil,
		},
	}
	manifest.DefaultPanel = "panel2"
	writeJSON(t, filepath.Join(root, "manifest.json"), manifest)

	sdir, err = NewSpeciesDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if sdir.PanelName != "panel2" {
		t.Fatalf("unexpected default panel after update: %s", sdir.PanelName)
	}
	if sdir.SanityCheck() {
		t.Fatal("expected missing panel2 files to fail sanity check")
	}
	if err := os.WriteFile(filepath.Join(root, "probes2.fa"), []byte(">x\nAAAAA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "amr2.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !sdir.SanityCheck() {
		t.Fatal("expected panel2 sanity check to pass")
	}
}

func TestDataDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "panels")
	ddir, err := NewDataDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ddir.Manifest) != 0 {
		t.Fatalf("expected empty manifest, got %v", ddir.Manifest)
	}
	if err := ddir.CreateRoot(); err != nil {
		t.Fatal(err)
	}
	if ddir.IsLocked() {
		t.Fatal("did not expect initial lock")
	}
	if err := ddir.StartLock(); err != nil {
		t.Fatal(err)
	}
	if !ddir.IsLocked() {
		t.Fatal("expected lock file")
	}
	if err := ddir.StopLock(); err != nil {
		t.Fatal(err)
	}

	species1TarV1 := makeSpeciesTarball(t, "species1", "20200101", "panel1")
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	writeJSON(t, manifestPath, map[string]manifestVersion{
		"species1": {Version: "20200101", URL: species1TarV1},
		"species2": {Version: "20190211", URL: "species2_url"},
	})
	if err := ddir.UpdateManifestFromFile(manifestPath); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ddir.AllSpeciesList(), []string{"species1", "species2"}) {
		t.Fatalf("unexpected species list: %v", ddir.AllSpeciesList())
	}
	if len(ddir.InstalledSpecies()) != 0 {
		t.Fatalf("unexpected installed species: %v", ddir.InstalledSpecies())
	}
	if ddir.SpeciesIsInstalled("species1") || ddir.SpeciesIsInstalled("species2") {
		t.Fatal("expected no installed species yet")
	}
	if got, err := ddir.GetSpeciesDir("species1"); err != nil || got != nil {
		t.Fatalf("expected nil species dir for uninstalled species, got %v, %v", got, err)
	}
	if _, err := ddir.GetSpeciesDir("unknown species"); err == nil {
		t.Fatal("expected unknown species error")
	}

	if err := ddir.AddOrReplaceSpeciesData(species1TarV1, false); err != nil {
		t.Fatal(err)
	}
	if !ddir.SpeciesIsInstalled("species1") {
		t.Fatal("expected species1 to be installed")
	}
	if !reflect.DeepEqual(ddir.InstalledSpecies(), []string{"species1"}) {
		t.Fatalf("unexpected installed species: %v", ddir.InstalledSpecies())
	}
	sdir, err := ddir.GetSpeciesDir("species1")
	if err != nil || sdir == nil {
		t.Fatalf("expected installed species dir, got %v, %v", sdir, err)
	}

	species1TarV2 := makeSpeciesTarball(t, "species1", "20200801", "panel1")
	writeJSON(t, manifestPath, map[string]manifestVersion{
		"species1": {Version: "20200801", URL: species1TarV2},
		"species2": {Version: "20190211", URL: "species2_url"},
	})
	if err := ddir.UpdateManifestFromFile(manifestPath); err != nil {
		t.Fatal(err)
	}
	if err := ddir.AddOrReplaceSpeciesData(species1TarV2, false); err == nil {
		t.Fatal("expected duplicate install without force to fail")
	}
	if !ddir.IsLocked() {
		t.Fatal("expected failed install to leave lock in place")
	}
	if err := ddir.StopLock(); err != nil {
		t.Fatal(err)
	}
	if !ddir.SpeciesIsInstalled("species1") {
		t.Fatal("expected species1 to remain installed")
	}

	if err := ddir.AddOrReplaceSpeciesData(species1TarV2, true); err != nil {
		t.Fatal(err)
	}
	sdir, err = ddir.GetSpeciesDir("species1")
	if err != nil {
		t.Fatal(err)
	}
	if sdir.Version() != "20200801" {
		t.Fatalf("expected updated species version, got %s", sdir.Version())
	}
	if err := ddir.RemoveSpecies("unknown species"); err == nil {
		t.Fatal("expected unknown species removal error")
	}
	if err := ddir.RemoveSpecies("species1"); err != nil {
		t.Fatal(err)
	}
	if ddir.SpeciesIsInstalled("species1") {
		t.Fatal("expected species1 to be removed")
	}
	if len(ddir.InstalledSpecies()) != 0 {
		t.Fatalf("unexpected installed species after removal: %v", ddir.InstalledSpecies())
	}
}

func TestResolveDownloadURLForFigshare(t *testing.T) {
	oldURL := "https://figshare.com/ndownloader/files/42494211"
	newURL := "https://ndownloader.figshare.com/files/42494211"
	if got := resolveDownloadURL(oldURL); got != newURL {
		t.Fatalf("unexpected rewritten url: %s", got)
	}
	if got := resolveDownloadURL(newURL); got != newURL {
		t.Fatalf("unexpected direct url rewrite: %s", got)
	}
	other := "https://example.com/file.tar.gz"
	if got := resolveDownloadURL(other); got != other {
		t.Fatalf("unexpected rewrite of non-figshare url: %s", got)
	}
}

func makeSpeciesTarball(t *testing.T, species, version, panel string) string {
	t.Helper()
	base := filepath.Join(t.TempDir(), species+"_data")
	root := filepath.Join(base, species)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := SpeciesManifest{
		SpeciesName:  species,
		Version:      version,
		DefaultPanel: panel,
		Panels: map[string]PanelManifest{
			panel: {
				Description:       "description",
				ReferenceGenome:   "NC42",
				SpeciesPhyloGroup: species + "_phylo",
				FASTAFiles:        []string{"panel.fa"},
				Kmer:              5,
				JSONFiles: map[string]*string{
					"amr":       ptr("amr.json"),
					"lineage":   ptr("lineage.json"),
					"hierarchy": nil,
				},
			},
		},
	}
	writeJSON(t, filepath.Join(root, "manifest.json"), manifest)
	if err := os.WriteFile(filepath.Join(root, "panel.fa"), []byte(">gene?name=gene&panel_type=presence&version=1\nAAAAA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "amr.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lineage.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	tarball := filepath.Join(t.TempDir(), species+"."+version+".tar.gz")
	f, err := os.Create(tarball)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	if err := addTreeToTar(tw, base, ""); err != nil {
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

func addTreeToTar(tw *tar.Writer, root, prefix string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		name := rel
		if prefix != "" {
			name = filepath.Join(prefix, rel)
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(name)
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

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func ptr(s string) *string { return &s }
