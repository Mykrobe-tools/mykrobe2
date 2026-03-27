package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/martinghunt/mykrobe2/mykrobe"
	"github.com/martinghunt/mykrobe2/mykrobe/speciesdata"
	"github.com/spf13/cobra"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := newRootCmd()
	cmd.SetArgs(args)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd.Execute()
}

type predictOptions struct {
	sample               string
	seqPath              string
	panelArg             string
	mapPath              string
	lineagePath          string
	panelsDir            string
	species              string
	output               string
	outputFormat         string
	model                string
	ploidy               string
	k                    int
	expectedDepth        float64
	minDepth             float64
	errorRate            float64
	minorFreq            float64
	minPropExpectedDepth float64
	minVariantConf       int
	minGeneConf          int
	reportAllCalls       bool
	ignoreMinorCalls     bool
	ncbiNames            bool
	ont                  bool
	guessSequenceMethod  bool
	confPercentCutoff    float64
	writeCovgs           string
}

type panelsUpdateMetadataOptions struct {
	panelsDir    string
	manifestURL  string
	manifestFile string
}

type panelsUpdateSpeciesOptions struct {
	panelsDir string
}

type makeProbesOptions struct {
	referencePath  string
	vcfPath        string
	backgroundVCF  []string
	backgroundList string
	variants       []string
	textFile       string
	genbankPath    string
	kmer           int
	lineagePath    string
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "mykrobe2",
	}
	cmd.AddCommand(newPredictCmd(), newPanelsCmd(), newMakeProbesCmd())
	return cmd
}

func newPredictCmd() *cobra.Command {
	opts := &predictOptions{}
	defaultPanels := defaultPanelsDir()
	cmd := &cobra.Command{
		Use:   "predict",
		Short: "Run prediction",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPredict(opts)
		},
	}
	cmd.Flags().StringVar(&opts.sample, "sample", "sample", "")
	cmd.Flags().StringVar(&opts.seqPath, "seq", "", "")
	cmd.Flags().StringVar(&opts.panelArg, "panel", "", "")
	cmd.Flags().StringVar(&opts.mapPath, "variant_to_resistance_json", "", "")
	cmd.Flags().StringVar(&opts.lineagePath, "lineage_json", "", "")
	cmd.Flags().StringVar(&opts.panelsDir, "panels_dir", defaultPanels, "Installed panels directory")
	cmd.Flags().StringVar(&opts.species, "species", "", "")
	cmd.Flags().StringVar(&opts.output, "output", "", "")
	cmd.Flags().StringVar(&opts.outputFormat, "format", "json", "")
	cmd.Flags().StringVar(&opts.model, "model", "kmer_count", "")
	cmd.Flags().StringVar(&opts.ploidy, "ploidy", "diploid", "")
	cmd.Flags().IntVar(&opts.k, "k", 0, "")
	cmd.Flags().Float64Var(&opts.expectedDepth, "expected_depth", 0, "")
	cmd.Flags().Float64Var(&opts.minDepth, "min_depth", 3, "")
	cmd.Flags().Float64Var(&opts.errorRate, "expected_error_rate", mykrobe.DefaultErrorRate, "")
	cmd.Flags().Float64Var(&opts.minorFreq, "minor_freq", mykrobe.DefaultMinorFreq, "")
	cmd.Flags().Float64Var(&opts.minPropExpectedDepth, "min_proportion_expected_depth", 0.3, "")
	cmd.Flags().IntVar(&opts.minVariantConf, "min_variant_conf", 150, "")
	cmd.Flags().IntVar(&opts.minGeneConf, "min_gene_conf", 1, "")
	cmd.Flags().BoolVar(&opts.reportAllCalls, "report_all_calls", false, "")
	cmd.Flags().BoolVar(&opts.ignoreMinorCalls, "ignore_minor_calls", false, "")
	cmd.Flags().BoolVar(&opts.ncbiNames, "ncbi_names", false, "")
	cmd.Flags().BoolVar(&opts.ont, "ont", false, "")
	cmd.Flags().BoolVar(&opts.guessSequenceMethod, "guess_sequence_method", false, "")
	cmd.Flags().Float64Var(&opts.confPercentCutoff, "conf_percent_cutoff", 100, "")
	cmd.Flags().StringVar(&opts.writeCovgs, "write_covgs", "", "Write intermediate coverage summary TSV to file")
	return cmd
}

func newPanelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "panels",
	}
	cmd.AddCommand(newPanelsUpdateMetadataCmd(), newPanelsUpdateSpeciesCmd())
	return cmd
}

func newPanelsUpdateMetadataCmd() *cobra.Command {
	opts := &panelsUpdateMetadataOptions{}
	defaultPanels := defaultPanelsDir()
	cmd := &cobra.Command{
		Use:   "update_metadata",
		Short: "Update available panel metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPanelsUpdateMetadata(opts)
		},
	}
	cmd.Flags().StringVar(&opts.panelsDir, "panels_dir", defaultPanels, "Installed panels directory")
	cmd.Flags().StringVar(&opts.manifestURL, "manifest_url", speciesdata.DefaultManifestURL, "")
	cmd.Flags().StringVar(&opts.manifestFile, "manifest_file", "", "")
	return cmd
}

func newPanelsUpdateSpeciesCmd() *cobra.Command {
	opts := &panelsUpdateSpeciesOptions{}
	defaultPanels := defaultPanelsDir()
	cmd := &cobra.Command{
		Use:   "update_species <species|all>",
		Short: "Install or update species panels",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPanelsUpdateSpecies(opts, args[0])
		},
	}
	cmd.Flags().StringVar(&opts.panelsDir, "panels_dir", defaultPanels, "Installed panels directory")
	return cmd
}

func newMakeProbesCmd() *cobra.Command {
	opts := &makeProbesOptions{}
	cmd := &cobra.Command{
		Use:   "make-probes <reference_filepath>",
		Short: "Make probes from a list of variants",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.referencePath = args[0]
			return runMakeProbes(opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVarP(&opts.vcfPath, "vcf", "f", "", "Use variants defined in a VCF file")
	cmd.Flags().StringSliceVar(&opts.backgroundVCF, "background-vcf", nil, "VCF file(s) containing background variants")
	cmd.Flags().StringVar(&opts.backgroundList, "background-vcf-list", "", "File containing background VCF filenames, one per line")
	cmd.Flags().StringSliceVarP(&opts.variants, "variants", "v", nil, "Variant in DNA positions e.g. A1234T")
	cmd.Flags().StringVarP(&opts.textFile, "text_file", "t", "", "Tab-delimited file containing DNA variants")
	cmd.Flags().StringVarP(&opts.genbankPath, "genbank", "g", "", "Genbank file containing genes as features")
	cmd.Flags().IntVarP(&opts.kmer, "kmer", "k", mykrobe.DefaultKmerSize, "kmer length")
	cmd.Flags().StringVar(&opts.lineagePath, "lineage", "", "Write lineage JSON to file")
	return cmd
}

func defaultPanelsDir() string {
	if dir := userDataDir(); dir != "" {
		return filepath.Join(dir, "mykrobe2", "panels")
	}
	if dir, err := os.UserHomeDir(); err == nil && dir != "" {
		return filepath.Join(dir, ".mykrobe2", "panels")
	}
	return filepath.Join(".", "panels")
}

func userDataDir() string {
	if runtime.GOOS == "windows" {
		if dir, err := os.UserConfigDir(); err == nil && dir != "" {
			return dir
		}
		return ""
	}
	if dir := os.Getenv("MYKROBE_DATA_HOME"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".local", "share")
	}
	return ""
}
