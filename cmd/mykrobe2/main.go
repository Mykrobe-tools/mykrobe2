package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Mykrobe-tools/mykrobe2/internal/buildinfo"
	"github.com/Mykrobe-tools/mykrobe2/mykrobe"
	"github.com/Mykrobe-tools/mykrobe2/mykrobe/speciesdata"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	seqPaths             []string
	indexPath            string
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
	guiProgressFile      string
}

type panelsUpdateMetadataOptions struct {
	panelsDir    string
	manifestURL  string
	manifestFile string
}

type panelsUpdateSpeciesOptions struct {
	panelsDir string
}

type panelsDescribeOptions struct {
	panelsDir string
	format    string
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

type indexOptions struct {
	fastaPaths  []string
	amrPath     string
	lineagePath string
	outputPath  string
	kmer        int
}

func newRootCmd() *cobra.Command {
	// Keep the root help ordered by the typical user workflow rather than by name.
	cobra.EnableCommandSorting = false
	const (
		mainCommandGroup       = "main"
		additionalCommandGroup = "additional"
	)
	cmd := &cobra.Command{
		Use:   "mykrobe2",
		Short: "Antimicrobial resistance and species prediction from sequence data",
		Long: "Mykrobe2 predicts antimicrobial resistance, species, and lineage from " +
			"whole-genome sequence data using installed or custom probe panels.",
		Version: displayVersion(buildinfo.Version),
	}
	cmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")
	cmd.SetGlobalNormalizationFunc(normalizeFlagName)
	cmd.AddGroup(
		&cobra.Group{ID: mainCommandGroup, Title: "Available Commands:"},
		&cobra.Group{ID: additionalCommandGroup, Title: "Additional Commands:"},
	)
	cmd.SetHelpCommandGroupID(mainCommandGroup)
	cmd.SetCompletionCommandGroupID(additionalCommandGroup)
	commands := []*cobra.Command{
		newPredictCmd(),
		newPanelsCmd(),
		newMakeProbesCmd(),
		newIndexCmd(),
		newDownloadTestReadsCmd(),
		newCompareOutputCmd(),
	}
	for _, child := range commands {
		child.GroupID = mainCommandGroup
	}
	cmd.AddCommand(commands...)
	return cmd
}

func normalizeFlagName(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	return pflag.NormalizedName(strings.ReplaceAll(name, "_", "-"))
}

func displayVersion(raw string) string {
	if len(raw) > 1 && (raw[0] == 'v' || raw[0] == 'V') && raw[1] >= '0' && raw[1] <= '9' {
		return raw[1:]
	}
	return raw
}

func newPredictCmd() *cobra.Command {
	opts := &predictOptions{}
	defaultPanels := defaultPanelsDir()
	cmd := &cobra.Command{
		Use:   "predict",
		Short: "Predict antimicrobial resistance and phylogeny",
		Long: "Predict antimicrobial resistance, species, and lineage from sequence data.\n\n" +
			"Select an installed panel with --species (and optionally --panel), or use a " +
			"custom panel built by 'mykrobe2 index' with --index.",
		Example: "  mykrobe2 predict --sample SAMPLE --species tb --seq reads.fastq.gz --output result.json\n" +
			"  mykrobe2 predict --sample SAMPLE --index custom.panelindex --seq reads.fastq.gz --output result.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPredict(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.sample, "sample", "s", "sample", "Sample identifier")
	cmd.Flags().StringSliceVarP(&opts.seqPaths, "seq", "i", nil, "Sequence file(s) in FASTA, FASTQ, or BAM format (required)")
	cmd.Flags().StringVar(&opts.indexPath, "index", "", "Custom bundled panel index from 'mykrobe2 index'")
	cmd.Flags().StringVar(&opts.panelArg, "panel", "", "Panel name for the selected species (defaults to that species' default panel)")
	cmd.Flags().StringVar(&opts.mapPath, "variant-to-resistance-json", "", "Override the installed panel's variant-to-resistance JSON file")
	cmd.Flags().StringVar(&opts.lineagePath, "lineage-json", "", "Override the installed panel's lineage JSON file")
	cmd.Flags().StringVar(&opts.panelsDir, "panels-dir", defaultPanels, "Directory containing installed panel data")
	cmd.Flags().StringVarP(&opts.species, "species", "S", "", "Species name; run 'mykrobe2 panels describe' to list available species")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Output file path (required)")
	cmd.Flags().StringVarP(&opts.outputFormat, "format", "O", "json", "Output format: json, csv, or json_and_csv")
	cmd.Flags().StringVar(&opts.model, "model", "kmer_count", "Genotype model: kmer_count or median_depth")
	cmd.Flags().StringVar(&opts.ploidy, "ploidy", "diploid", "Genotyping model ploidy: diploid or haploid")
	cmd.Flags().IntVarP(&opts.k, "kmer", "k", 0, "K-mer length (0 uses the panel value)")
	cmd.Flags().IntVar(&opts.k, "k", 0, "K-mer length (alias for --kmer)")
	_ = cmd.Flags().MarkHidden("k")
	cmd.Flags().Float64Var(&opts.expectedDepth, "expected-depth", 0, "Expected sequencing depth (0 estimates it from the data)")
	cmd.Flags().Float64Var(&opts.minDepth, "min-depth", 1, "Minimum depth for resistance prediction")
	cmd.Flags().Float64VarP(&opts.errorRate, "expected-error-rate", "e", mykrobe.DefaultErrorRate, "Expected sequencing error rate")
	cmd.Flags().Float64Var(&opts.minorFreq, "minor-freq", mykrobe.DefaultMinorFreq, "Expected frequency of the minor allele in a mixed call")
	cmd.Flags().Float64VarP(&opts.minPropExpectedDepth, "min-proportion-expected-depth", "D", 0.3, "Minimum allele depth as a proportion of expected depth")
	cmd.Flags().IntVar(&opts.minVariantConf, "min-variant-conf", 150, "Minimum genotype confidence for variant calls")
	cmd.Flags().IntVar(&opts.minGeneConf, "min-gene-conf", 1, "Minimum genotype confidence for gene calls")
	cmd.Flags().BoolVarP(&opts.reportAllCalls, "report-all-calls", "A", false, "Include all variant, sequence, and lineage calls in the output")
	cmd.Flags().BoolVar(&opts.ignoreMinorCalls, "ignore-minor-calls", false, "Ignore minor calls when predicting resistance")
	cmd.Flags().BoolVar(&opts.ncbiNames, "ncbi-names", false, "Include NCBI species names in TB phylogenetics output")
	cmd.Flags().BoolVar(&opts.ont, "ont", false, "Use Oxford Nanopore defaults for error rate and ploidy")
	cmd.Flags().BoolVar(&opts.guessSequenceMethod, "guess-sequence-method", false, "Infer Illumina or Nanopore data from the estimated error rate")
	cmd.Flags().Float64Var(&opts.confPercentCutoff, "conf-percent-cutoff", 100, "Percent of simulated variants used to set the confidence threshold (0-100)")
	cmd.Flags().StringVar(&opts.writeCovgs, "write-covgs", "", "Write intermediate coverage summary TSV to file")
	cmd.Flags().StringVar(&opts.guiProgressFile, "gui-progress-file", "", "Write JSON progress events for the desktop GUI")
	_ = cmd.Flags().MarkHidden("gui-progress-file")
	return cmd
}

func newIndexCmd() *cobra.Command {
	opts := &indexOptions{}
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Build a bundled custom panel index",
		Long: "Build a self-contained custom panel index from one or more probe FASTA " +
			"files, with optional resistance and lineage metadata.",
		Example: "  mykrobe2 index --fasta probes.fa --variant-to-resistance-json amr.json --output custom.panelindex",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIndex(opts)
		},
	}
	cmd.Flags().StringSliceVar(&opts.fastaPaths, "fasta", nil, "Probe FASTA file(s) (required)")
	cmd.Flags().StringVar(&opts.amrPath, "variant-to-resistance-json", "", "Variant-to-resistance JSON file to embed")
	cmd.Flags().StringVar(&opts.lineagePath, "lineage-json", "", "Lineage JSON file to embed")
	cmd.Flags().StringVar(&opts.outputPath, "output", "", "Output .panelindex file (required)")
	cmd.Flags().IntVar(&opts.kmer, "k", mykrobe.DefaultKmerSize, "K-mer length")
	return cmd
}

func newPanelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "panels",
		Short: "Inspect and update installed panels",
		Long:  "Inspect available panel data, refresh panel metadata, or install and update species panels.",
	}
	cmd.AddCommand(newPanelsDescribeCmd(), newPanelsUpdateMetadataCmd(), newPanelsUpdateSpeciesCmd())
	return cmd
}

func newPanelsDescribeCmd() *cobra.Command {
	opts := &panelsDescribeOptions{}
	defaultPanels := defaultPanelsDir()
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describe all known panels",
		Long:  "Show the available species and panels, their installed versions, and whether updates are available.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPanelsDescribe(opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&opts.panelsDir, "panels-dir", defaultPanels, "Directory containing installed panel data")
	cmd.Flags().StringVar(&opts.format, "format", "text", "Output format: text or json")
	return cmd
}

func newPanelsUpdateMetadataCmd() *cobra.Command {
	opts := &panelsUpdateMetadataOptions{}
	defaultPanels := defaultPanelsDir()
	cmd := &cobra.Command{
		Use:     "update-metadata",
		Aliases: []string{"update_metadata"},
		Short:   "Update metadata about available panels",
		Long:    "Refresh the metadata that lists available species panels and their latest versions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPanelsUpdateMetadata(opts)
		},
	}
	cmd.Flags().StringVar(&opts.panelsDir, "panels-dir", defaultPanels, "Directory containing installed panel data")
	cmd.Flags().StringVar(&opts.manifestURL, "manifest-url", speciesdata.DefaultManifestURL, "URL of the panel manifest to download")
	cmd.Flags().StringVar(&opts.manifestFile, "manifest-file", "", "Read the panel manifest from a local file instead of a URL")
	return cmd
}

func newPanelsUpdateSpeciesCmd() *cobra.Command {
	opts := &panelsUpdateSpeciesOptions{}
	defaultPanels := defaultPanelsDir()
	cmd := &cobra.Command{
		Use:     "update-species <species|all>",
		Aliases: []string{"update_species"},
		Short:   "Install or update species panels",
		Long:    "Install or update panel data for one species, or use 'all' to update every known species.",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPanelsUpdateSpecies(opts, args[0])
		},
	}
	cmd.Flags().StringVar(&opts.panelsDir, "panels-dir", defaultPanels, "Directory containing installed panel data")
	return cmd
}

func newMakeProbesCmd() *cobra.Command {
	opts := &makeProbesOptions{}
	cmd := &cobra.Command{
		Use:   "make-probes <reference-filepath>",
		Short: "Make probes from a list of variants",
		Long: "Make variant probes against a reference genome. Variants may be supplied " +
			"directly, in a text file, or in a VCF file.",
		Example: "  mykrobe2 make-probes reference.fa --variants A1234T\n" +
			"  mykrobe2 make-probes reference.fa --vcf variants.vcf > probes.fa",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.referencePath = args[0]
			return runMakeProbes(opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVarP(&opts.vcfPath, "vcf", "f", "", "VCF file containing variants")
	cmd.Flags().StringSliceVar(&opts.backgroundVCF, "background-vcf", nil, "VCF file(s) containing background variants")
	cmd.Flags().StringVar(&opts.backgroundList, "background-vcf-list", "", "File containing background VCF filenames, one per line")
	cmd.Flags().StringSliceVarP(&opts.variants, "variants", "v", nil, "DNA variant(s), for example A1234T")
	cmd.Flags().StringVarP(&opts.textFile, "text-file", "t", "", "Text file containing variants, one per row (for example A1234T)")
	cmd.Flags().StringVarP(&opts.genbankPath, "genbank", "g", "", "GenBank file containing genes as features")
	cmd.Flags().IntVarP(&opts.kmer, "kmer", "k", mykrobe.DefaultKmerSize, "K-mer length")
	cmd.Flags().StringVar(&opts.lineagePath, "lineage", "", "Write lineage definitions to a JSON file")
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
