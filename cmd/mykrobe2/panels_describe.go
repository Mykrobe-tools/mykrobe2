package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/martinghunt/mykrobe2/mykrobe/speciesdata"
)

type panelDescription struct {
	Name        string `json:"name"`
	Reference   string `json:"reference"`
	Description string `json:"description"`
}

type speciesDescription struct {
	Species          string             `json:"species"`
	UpdateAvailable  bool               `json:"update_available"`
	Installed        bool               `json:"installed"`
	InstalledVersion string             `json:"installed_version"`
	InstalledURL     string             `json:"installed_url"`
	LatestVersion    string             `json:"latest_version"`
	LatestURL        string             `json:"latest_url"`
	DefaultPanel     string             `json:"default_panel"`
	Panels           []panelDescription `json:"panels"`
}

type describeOutput struct {
	PanelsDir string               `json:"panels_dir"`
	Species   []speciesDescription `json:"species"`
}

var tbLegacyPanelsLast = map[string]int{
	"bradley-2015": 0,
	"walker-2015":  1,
}

func runPanelsDescribe(opts *panelsDescribeOptions, w io.Writer) error {
	ddir, err := speciesdata.NewDataDir(opts.panelsDir)
	if err != nil {
		return err
	}
	out, err := buildPanelsDescribeOutput(ddir)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(opts.format)) {
	case "", "text":
		writePanelsDescribeText(w, out)
		return nil
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	default:
		return fmt.Errorf("unknown --format %q", opts.format)
	}
}

func buildPanelsDescribeOutput(ddir *speciesdata.DataDir) (*describeOutput, error) {
	out := &describeOutput{
		PanelsDir: ddir.RootDir,
		Species:   make([]speciesDescription, 0, len(ddir.Manifest)),
	}
	for _, species := range ddir.AllSpeciesList() {
		entry := ddir.Manifest[species]
		item := speciesDescription{
			Species:         species,
			UpdateAvailable: !ddir.SpeciesIsUpToDate(species),
			Installed:       ddir.SpeciesIsInstalled(species),
		}
		if entry.Installed != nil {
			item.InstalledVersion = entry.Installed.Version
			item.InstalledURL = entry.Installed.URL
		} else {
			item.InstalledVersion = "None"
			item.InstalledURL = "NA"
		}
		if entry.Latest != nil {
			item.LatestVersion = entry.Latest.Version
			item.LatestURL = entry.Latest.URL
		}
		if item.Installed {
			sdir, err := ddir.GetSpeciesDir(species)
			if err != nil {
				return nil, err
			}
			if sdir != nil {
				item.DefaultPanel = sdir.DefaultPanel()
				panelNames := sdir.PanelNames()
				sortPanelNames(species, panelNames)
				item.Panels = make([]panelDescription, 0, len(panelNames))
				for _, panelName := range panelNames {
					if err := sdir.SetPanel(panelName); err != nil {
						return nil, err
					}
					item.Panels = append(item.Panels, panelDescription{
						Name:        panelName,
						Reference:   sdir.ReferenceGenome(),
						Description: sdir.Description(),
					})
				}
			}
		}
		out.Species = append(out.Species, item)
	}
	return out, nil
}

func sortPanelNames(species string, panelNames []string) {
	sort.Slice(panelNames, func(i, j int) bool {
		a := panelNames[i]
		b := panelNames[j]
		if strings.EqualFold(species, "tb") {
			aRank, aLegacy := tbLegacyPanelsLast[a]
			bRank, bLegacy := tbLegacyPanelsLast[b]
			if aLegacy != bLegacy {
				return !aLegacy
			}
			if aLegacy && bLegacy {
				return aRank < bRank
			}
		}
		return a > b
	})
}

func writePanelsDescribeText(w io.Writer, out *describeOutput) {
	if len(out.Species) == 0 {
		fmt.Fprintln(w, "No data")
		return
	}
	fmt.Fprintf(w, "Gathering data from %s\n\n", out.PanelsDir)
	fmt.Fprintln(w, "Species summary:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Species\tUpdate_available\tInstalled_version\tInstalled_url\tLatest_version\tLatest_url")
	for _, item := range out.Species {
		update := "yes"
		if !item.UpdateAvailable {
			update = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Species,
			update,
			item.InstalledVersion,
			item.InstalledURL,
			item.LatestVersion,
			item.LatestURL,
		)
	}
	installedCount := 0
	for _, item := range out.Species {
		if item.Installed {
			installedCount++
		}
	}
	if installedCount == 0 {
		fmt.Fprintln(w, "\nNo panels are installed")
		return
	}
	for _, item := range out.Species {
		if !item.Installed {
			continue
		}
		fmt.Fprintf(w, "\n%s default panel: %s\n", item.Species, item.DefaultPanel)
		fmt.Fprintf(w, "%s panels:\n", item.Species)
		fmt.Fprintln(w, "Panel\tReference\tDescription")
		for _, panel := range item.Panels {
			fmt.Fprintf(w, "%s\t%s\t%s\n", panel.Name, panel.Reference, panel.Description)
		}
	}
}
