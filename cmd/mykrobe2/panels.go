package main

import "github.com/Mykrobe-tools/mykrobe2/mykrobe/speciesdata"

func runPanelsUpdateMetadata(opts *panelsUpdateMetadataOptions) error {
	ddir, err := speciesdata.NewDataDir(opts.panelsDir)
	if err != nil {
		return err
	}
	if opts.manifestFile != "" {
		return ddir.UpdateManifestFromFile(opts.manifestFile)
	}
	return ddir.UpdateManifestFromURL(opts.manifestURL)
}

func runPanelsUpdateSpecies(opts *panelsUpdateSpeciesOptions, species string) error {
	progressWriter, err := openGUIProgressWriter(opts.guiProgressFile)
	if err != nil {
		return err
	}
	if progressWriter != nil {
		defer progressWriter.Close()
	}
	ddir, err := speciesdata.NewDataDir(opts.panelsDir)
	if err != nil {
		return err
	}
	var progress speciesdata.PanelProgressFunc
	if progressWriter != nil {
		progress = func(event speciesdata.PanelProgressEvent) {
			progressWriter.Write(event)
		}
	}
	if species == "all" {
		return ddir.UpdateAllSpeciesWithProgress(progress)
	}
	return ddir.UpdateSpeciesWithProgress(species, progress)
}
