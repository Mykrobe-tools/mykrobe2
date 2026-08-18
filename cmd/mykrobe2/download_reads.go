package main

import (
	"fmt"

	"github.com/Mykrobe-tools/mykrobe2/mykrobe"
	"github.com/spf13/cobra"
)

var downloadTBTestReads = mykrobe.DownloadTBTestReads

func newDownloadTestReadsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download-test-reads <output-filename>",
		Short: "Download test reads for the TB panels",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := downloadTBTestReads(cmd.Context(), args[0]); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Downloaded TB test reads to %s\n", args[0])
			return err
		},
	}
	return cmd
}
