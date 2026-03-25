package speciesdata

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const DefaultManifestURL = "https://raw.githubusercontent.com/Mykrobe-tools/mykrobe-data/main/mykrobe_panels_manifest.json"

type PanelManifest struct {
	Description       string             `json:"description"`
	ReferenceGenome   string             `json:"reference_genome"`
	SpeciesPhyloGroup string             `json:"species_phylo_group"`
	FASTAFiles        []string           `json:"fasta_files"`
	Kmer              int                `json:"kmer"`
	JSONFiles         map[string]*string `json:"json_files"`
}

type SpeciesManifest struct {
	SpeciesName  string                   `json:"species_name"`
	Version      string                   `json:"version"`
	DefaultPanel string                   `json:"default_panel"`
	Panels       map[string]PanelManifest `json:"panels"`
}

type SpeciesDir struct {
	RootDir      string
	ManifestJSON string
	Manifest     SpeciesManifest
	PanelName    string
	Panel        PanelManifest
}

func NewSpeciesDir(rootDir string) (*SpeciesDir, error) {
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(rootDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("species directory %s not found", rootDir)
		}
		return nil, err
	}
	manifestPath := filepath.Join(rootDir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("manifest file not found in species directory %s", rootDir)
		}
		return nil, err
	}

	var manifest SpeciesManifest
	if err := loadJSON(manifestPath, &manifest); err != nil {
		return nil, err
	}
	s := &SpeciesDir{
		RootDir:      rootDir,
		ManifestJSON: manifestPath,
		Manifest:     manifest,
	}
	if err := s.SetPanel(s.DefaultPanel()); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SpeciesDir) SetPanel(panelName string) error {
	panel, ok := s.Manifest.Panels[panelName]
	if !ok {
		return fmt.Errorf("panel %q not found in %s", panelName, s.RootDir)
	}
	s.PanelName = panelName
	s.Panel = panel
	return nil
}

func (s *SpeciesDir) DefaultPanel() string {
	return s.Manifest.DefaultPanel
}

func (s *SpeciesDir) PanelNames() []string {
	out := make([]string, 0, len(s.Manifest.Panels))
	for name := range s.Manifest.Panels {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (s *SpeciesDir) SpeciesName() string {
	return s.Manifest.SpeciesName
}

func (s *SpeciesDir) Version() string {
	return s.Manifest.Version
}

func (s *SpeciesDir) Description() string {
	return s.Panel.Description
}

func (s *SpeciesDir) ReferenceGenome() string {
	return s.Panel.ReferenceGenome
}

func (s *SpeciesDir) Kmer() int {
	return s.Panel.Kmer
}

func (s *SpeciesDir) SpeciesPhyloGroup() string {
	return s.Panel.SpeciesPhyloGroup
}

func (s *SpeciesDir) FASTAFiles() []string {
	if s.Panel.FASTAFiles == nil {
		return nil
	}
	out := make([]string, 0, len(s.Panel.FASTAFiles))
	for _, name := range s.Panel.FASTAFiles {
		out = append(out, filepath.Join(s.RootDir, name))
	}
	return out
}

func (s *SpeciesDir) JSONFile(kind string) string {
	if s.Panel.JSONFiles == nil {
		return ""
	}
	name := s.Panel.JSONFiles[kind]
	if name == nil || *name == "" {
		return ""
	}
	return filepath.Join(s.RootDir, *name)
}

func (s *SpeciesDir) SanityCheck() bool {
	if s.Manifest.SpeciesName == "" || s.Manifest.Version == "" || s.Manifest.DefaultPanel == "" || s.Manifest.Panels == nil {
		return false
	}
	if _, ok := s.Manifest.Panels[s.Manifest.DefaultPanel]; !ok {
		return false
	}

	for _, panelName := range s.PanelNames() {
		if err := s.SetPanel(panelName); err != nil {
			return false
		}
		panel := s.Panel
		if panel.Description == "" || panel.ReferenceGenome == "" || panel.Kmer == 0 || panel.SpeciesPhyloGroup == "" {
			return false
		}
		fastas := s.FASTAFiles()
		if len(fastas) == 0 {
			return false
		}
		for _, path := range fastas {
			if _, err := os.Stat(path); err != nil {
				return false
			}
		}
		foundJSON := false
		for _, kind := range []string{"amr", "lineage", "hierarchy", "ncbi_names"} {
			path := s.JSONFile(kind)
			if path == "" {
				continue
			}
			if _, err := os.Stat(path); err != nil {
				return false
			}
			foundJSON = true
		}
		if !foundJSON {
			return false
		}
	}

	return true
}

type manifestVersion struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

type DataManifestEntry struct {
	Installed *manifestVersion `json:"installed"`
	Latest    *manifestVersion `json:"latest"`
}

type DataDir struct {
	RootDir      string
	ManifestJSON string
	LockFile     string
	Manifest     map[string]DataManifestEntry
}

func NewDataDir(rootDir string) (*DataDir, error) {
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}
	d := &DataDir{
		RootDir:      rootDir,
		ManifestJSON: filepath.Join(rootDir, "manifest.json"),
		LockFile:     filepath.Join(rootDir, ".lock"),
		Manifest:     map[string]DataManifestEntry{},
	}
	if err := d.LoadManifest(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DataDir) IsLocked() bool {
	_, err := os.Stat(d.LockFile)
	return err == nil
}

func (d *DataDir) StartLock() error {
	if d.IsLocked() {
		return fmt.Errorf("lock file found: %s", d.LockFile)
	}
	f, err := os.Create(d.LockFile)
	if err != nil {
		return err
	}
	return f.Close()
}

func (d *DataDir) StopLock() error {
	return os.Remove(d.LockFile)
}

func (d *DataDir) LoadManifest() error {
	if _, err := os.Stat(d.ManifestJSON); err != nil {
		if os.IsNotExist(err) {
			d.Manifest = map[string]DataManifestEntry{}
			return nil
		}
		return err
	}
	return loadJSON(d.ManifestJSON, &d.Manifest)
}

func (d *DataDir) CreateRoot() error {
	return os.MkdirAll(d.RootDir, 0o755)
}

func (d *DataDir) SaveManifest() error {
	if err := d.CreateRoot(); err != nil {
		return err
	}
	f, err := os.Create(d.ManifestJSON)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(d.Manifest)
}

func (d *DataDir) UpdateManifestFromFile(filename string) error {
	if err := d.CreateRoot(); err != nil {
		return err
	}
	if err := d.StartLock(); err != nil {
		return err
	}
	defer d.StopLock()

	latest := map[string]manifestVersion{}
	if err := loadJSON(filename, &latest); err != nil {
		return err
	}
	for species, meta := range latest {
		entry := d.Manifest[species]
		entry.Latest = &manifestVersion{Version: meta.Version, URL: meta.URL}
		d.Manifest[species] = entry
	}
	return d.SaveManifest()
}

func (d *DataDir) UpdateManifestFromURL(url string) error {
	if err := d.CreateRoot(); err != nil {
		return err
	}
	if err := d.StartLock(); err != nil {
		return err
	}
	defer d.StopLock()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch manifest: %s", resp.Status)
	}
	latest := map[string]manifestVersion{}
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return err
	}
	for species, meta := range latest {
		entry := d.Manifest[species]
		entry.Latest = &manifestVersion{Version: meta.Version, URL: meta.URL}
		d.Manifest[species] = entry
	}
	return d.SaveManifest()
}

func (d *DataDir) AddOrReplaceSpeciesData(tarballName string, force bool) error {
	if err := d.CreateRoot(); err != nil {
		return err
	}
	if err := d.StartLock(); err != nil {
		return err
	}

	tmpDir := filepath.Join(d.RootDir, "tmp.add_species")
	_ = os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}

	fromFile := !regexp.MustCompile(`^https?://`).MatchString(tarballName)
	toExtract := tarballName
	if !fromFile {
		downloadURL := resolveDownloadURL(tarballName)
		resp, err := http.Get(downloadURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download species data: %s", resp.Status)
		}
		toExtract = filepath.Join(tmpDir, "download.tar.gz")
		f, err := os.Create(toExtract)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, resp.Body); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	if err := extractTarGz(toExtract, tmpDir); err != nil {
		return err
	}
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return err
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(tmpDir, entry.Name()))
		}
	}
	if len(dirs) != 1 {
		return fmt.Errorf("expected one extracted directory in %s", tmpDir)
	}

	spdir, err := NewSpeciesDir(dirs[0])
	if err != nil {
		return err
	}
	if !spdir.SanityCheck() {
		return fmt.Errorf("invalid species data in %s", dirs[0])
	}

	species := spdir.SpeciesName()
	if d.SpeciesIsInstalled(species) && !force {
		return fmt.Errorf("species %q already exists", species)
	}

	newDir := filepath.Join(d.RootDir, species)
	_ = os.RemoveAll(newDir)
	if err := os.Rename(dirs[0], newDir); err != nil {
		return err
	}
	_ = os.RemoveAll(tmpDir)

	entry := d.Manifest[species]
	entry.Installed = &manifestVersion{Version: spdir.Version(), URL: tarballName}
	if entry.Latest == nil {
		entry.Latest = &manifestVersion{Version: spdir.Version(), URL: tarballName}
	}
	d.Manifest[species] = entry
	if err := d.SaveManifest(); err != nil {
		return err
	}
	return d.StopLock()
}

func (d *DataDir) RemoveSpecies(species string) error {
	if !d.SpeciesIsInstalled(species) {
		return fmt.Errorf("species %s is not installed", species)
	}
	if err := d.StartLock(); err != nil {
		return err
	}
	defer d.StopLock()

	entry := d.Manifest[species]
	entry.Installed = nil
	d.Manifest[species] = entry
	if err := os.RemoveAll(filepath.Join(d.RootDir, species)); err != nil {
		return err
	}
	return d.SaveManifest()
}

func (d *DataDir) AllSpeciesList() []string {
	out := make([]string, 0, len(d.Manifest))
	for species := range d.Manifest {
		out = append(out, species)
	}
	sort.Strings(out)
	return out
}

func (d *DataDir) InstalledSpecies() []string {
	out := make([]string, 0)
	for _, species := range d.AllSpeciesList() {
		if d.SpeciesIsInstalled(species) {
			out = append(out, species)
		}
	}
	return out
}

func (d *DataDir) SpeciesIsInstalled(species string) bool {
	entry, ok := d.Manifest[species]
	return ok && entry.Installed != nil
}

func (d *DataDir) SpeciesIsUpToDate(species string) bool {
	entry, ok := d.Manifest[species]
	return ok && entry.Installed != nil && entry.Latest != nil && entry.Installed.Version >= entry.Latest.Version
}

func (d *DataDir) GetSpeciesDir(species string) (*SpeciesDir, error) {
	entry, ok := d.Manifest[species]
	if !ok {
		return nil, fmt.Errorf("species %q not found in data directory %s", species, d.RootDir)
	}
	if entry.Installed == nil {
		return nil, nil
	}
	return NewSpeciesDir(filepath.Join(d.RootDir, species))
}

func resolveDownloadURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if parsed.Host == "figshare.com" && strings.HasPrefix(parsed.Path, "/ndownloader/files/") {
		fileID := parsed.Path[strings.LastIndex(parsed.Path, "/")+1:]
		return "https://ndownloader.figshare.com/files/" + fileID
	}
	return rawURL
}

func (d *DataDir) UpdateSpecies(species string) error {
	entry, ok := d.Manifest[species]
	if !ok {
		return fmt.Errorf("unknown species %q", species)
	}
	if entry.Latest == nil {
		return fmt.Errorf("no latest metadata for species %q", species)
	}
	if d.SpeciesIsUpToDate(species) {
		return nil
	}
	return d.AddOrReplaceSpeciesData(entry.Latest.URL, true)
}

func (d *DataDir) UpdateAllSpecies() error {
	for _, species := range d.AllSpeciesList() {
		if err := d.UpdateSpecies(species); err != nil {
			return err
		}
	}
	return nil
}

func loadJSON(path string, dst any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var r io.Reader = f
	head := make([]byte, 2)
	if _, err := f.Read(head); err == nil {
		_, _ = f.Seek(0, 0)
		if head[0] == 0x1f && head[1] == 0x8b {
			gr, err := gzip.NewReader(f)
			if err != nil {
				return err
			}
			defer gr.Close()
			r = gr
		}
	}
	return json.NewDecoder(r).Decode(dst)
}

func extractTarGz(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, hdr.Name)
		cleanDest := filepath.Clean(dest) + string(filepath.Separator)
		if !strings.HasPrefix(filepath.Clean(target)+string(filepath.Separator), cleanDest) && filepath.Clean(target) != filepath.Clean(dest) {
			return fmt.Errorf("tar entry escapes destination: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}
