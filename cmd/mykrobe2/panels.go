package main

import "github.com/martinghunt/mykrobe2/mykrobe/speciesdata"

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
	ddir, err := speciesdata.NewDataDir(opts.panelsDir)
	if err != nil {
		return err
	}
	if species == "all" {
		return ddir.UpdateAllSpecies()
	}
	return ddir.UpdateSpecies(species)
}
