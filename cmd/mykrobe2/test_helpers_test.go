package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mykrobe-tools/mykrobe2/internal/testutil"
	"github.com/Mykrobe-tools/mykrobe2/mykrobe"
	"github.com/Mykrobe-tools/mykrobe2/mykrobe/speciesdata"
)

var mykrobeTestRefData = testutil.MykrobePath("tests", "ref_data")

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

func buildCustomIndexFile(t *testing.T, outputPath string, k int, fastaPaths []string, amrPath, lineagePath string) {
	t.Helper()
	if err := mykrobe.BuildCustomIndex(outputPath, k, fastaPaths, amrPath, lineagePath); err != nil {
		t.Fatal(err)
	}
}
