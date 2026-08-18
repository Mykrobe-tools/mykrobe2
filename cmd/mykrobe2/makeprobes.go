package main

import (
	"encoding/json"
	"io"
	"os"

	"github.com/Mykrobe-tools/mykrobe2/mykrobe/probes"
)

func runMakeProbes(opts *makeProbesOptions, out io.Writer) error {
	lineages, err := probes.WritePanels(out, probes.RunOptions{
		ReferencePath:  opts.referencePath,
		VCFPath:        opts.vcfPath,
		BackgroundVCF:  opts.backgroundVCF,
		BackgroundList: opts.backgroundList,
		Variants:       opts.variants,
		TextFile:       opts.textFile,
		GenbankPath:    opts.genbankPath,
		Kmer:           opts.kmer,
	})
	if err != nil {
		return err
	}
	if opts.lineagePath == "" {
		return nil
	}
	data, err := json.MarshalIndent(lineages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(opts.lineagePath, data, 0o644)
}
