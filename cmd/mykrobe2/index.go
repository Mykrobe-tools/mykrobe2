package main

import (
	"fmt"

	"github.com/martinghunt/mykrobe2/mykrobe"
)

func runIndex(opts *indexOptions) error {
	if len(opts.fastaPaths) == 0 || opts.outputPath == "" {
		return fmt.Errorf("index requires --fasta and --output")
	}
	return mykrobe.BuildCustomIndex(opts.outputPath, opts.kmer, opts.fastaPaths, opts.amrPath, opts.lineagePath)
}
