package main

import (
	"fmt"
	"io"

	"github.com/Mykrobe-tools/mykrobe2/mykrobe"
)

func runIndex(opts *indexOptions, stderr io.Writer) error {
	if len(opts.fastaPaths) == 0 || opts.outputPath == "" {
		return fmt.Errorf("index requires --fasta and --output")
	}
	return mykrobe.BuildCustomIndexWithWarnings(opts.outputPath, opts.kmer, opts.fastaPaths, opts.amrPath, opts.lineagePath, func(warning mykrobe.MappingAliasWarning) {
		fmt.Fprintf(stderr, "Warning: %s\n", warning)
	})
}
