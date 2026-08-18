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
	cmd := &cobra.Command{
		Use:     "mykrobe2",
		Version: displayVersion(buildinfo.Version),
	}
	cmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")
	cmd.SetGlobalNormalizationFunc(normalizeFlagName)
	cmd.AddCommand(newPredictCmd(), newPanelsCmd(), newMakeProbesCmd(), newIndexCmd(), newCompareOutputCmd(), newDownloadTestReadsCmd())
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
		Short: "Run prediction",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPredict(opts)
		},
	}
	cmd.Flags().StringVar(&opts.sample, "sample", "sample", "")
	cmd.Flags().StringSliceVarP(&opts.seqPaths, "seq", "i", nil, "")
	cmd.Flags().StringVar(&opts.indexPath, "index", "", "Custom bundled panel index file")
	cmd.Flags().StringVar(&opts.panelArg, "panel", "", "")
	cmd.Flags().StringVar(&opts.mapPath, "variant-to-resistance-json", "", "")
	cmd.Flags().StringVar(&opts.lineagePath, "lineage-json", "", "")
	cmd.Flags().StringVar(&opts.panelsDir, "panels-dir", defaultPanels, "Installed panels directory")
	cmd.Flags().StringVar(&opts.species, "species", "", "")
	cmd.Flags().StringVar(&opts.output, "output", "", "")
	cmd.Flags().StringVar(&opts.outputFormat, "format", "json", "")
	cmd.Flags().StringVar(&opts.model, "model", "kmer_count", "")
	cmd.Flags().StringVar(&opts.ploidy, "ploidy", "diploid", "")
	cmd.Flags().IntVar(&opts.k, "k", 0, "")
	cmd.Flags().Float64Var(&opts.expectedDepth, "expected-depth", 0, "")
	cmd.Flags().Float64Var(&opts.minDepth, "min-depth", 1, "")
	cmd.Flags().Float64Var(&opts.errorRate, "expected-error-rate", mykrobe.DefaultErrorRate, "")
	cmd.Flags().Float64Var(&opts.minorFreq, "minor-freq", mykrobe.DefaultMinorFreq, "")
	cmd.Flags().Float64Var(&opts.minPropExpectedDepth, "min-proportion-expected-depth", 0.3, "")
	cmd.Flags().IntVar(&opts.minVariantConf, "min-variant-conf", 150, "")
	cmd.Flags().IntVar(&opts.minGeneConf, "min-gene-conf", 1, "")
	cmd.Flags().BoolVar(&opts.reportAllCalls, "report-all-calls", false, "")
	cmd.Flags().BoolVar(&opts.ignoreMinorCalls, "ignore-minor-calls", false, "")
	cmd.Flags().BoolVar(&opts.ncbiNames, "ncbi-names", false, "")
	cmd.Flags().BoolVar(&opts.ont, "ont", false, "")
	cmd.Flags().BoolVar(&opts.guessSequenceMethod, "guess-sequence-method", false, "")
	cmd.Flags().Float64Var(&opts.confPercentCutoff, "conf-percent-cutoff", 100, "")
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIndex(opts)
		},
	}
	cmd.Flags().StringSliceVar(&opts.fastaPaths, "fasta", nil, "Probe FASTA file(s)")
	cmd.Flags().StringVar(&opts.amrPath, "variant-to-resistance-json", "", "Variant-to-resistance JSON file")
	cmd.Flags().StringVar(&opts.lineagePath, "lineage-json", "", "Lineage JSON file")
	cmd.Flags().StringVar(&opts.outputPath, "output", "", "Output .panelindex file")
	cmd.Flags().IntVar(&opts.kmer, "k", mykrobe.DefaultKmerSize, "kmer length")
	return cmd
}

func newPanelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "panels",
	}
	cmd.AddCommand(newPanelsDescribeCmd(), newPanelsUpdateMetadataCmd(), newPanelsUpdateSpeciesCmd())
	return cmd
}

func newPanelsDescribeCmd() *cobra.Command {
	opts := &panelsDescribeOptions{}
	defaultPanels := defaultPanelsDir()
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describe known panels",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPanelsDescribe(opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&opts.panelsDir, "panels-dir", defaultPanels, "Installed panels directory")
	cmd.Flags().StringVar(&opts.format, "format", "text", "Output format: text or json")
	return cmd
}

func newPanelsUpdateMetadataCmd() *cobra.Command {
	opts := &panelsUpdateMetadataOptions{}
	defaultPanels := defaultPanelsDir()
	cmd := &cobra.Command{
		Use:     "update-metadata",
		Aliases: []string{"update_metadata"},
		Short:   "Update available panel metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPanelsUpdateMetadata(opts)
		},
	}
	cmd.Flags().StringVar(&opts.panelsDir, "panels-dir", defaultPanels, "Installed panels directory")
	cmd.Flags().StringVar(&opts.manifestURL, "manifest-url", speciesdata.DefaultManifestURL, "")
	cmd.Flags().StringVar(&opts.manifestFile, "manifest-file", "", "")
	return cmd
}

func newPanelsUpdateSpeciesCmd() *cobra.Command {
	opts := &panelsUpdateSpeciesOptions{}
	defaultPanels := defaultPanelsDir()
	cmd := &cobra.Command{
		Use:     "update-species <species|all>",
		Aliases: []string{"update_species"},
		Short:   "Install or update species panels",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPanelsUpdateSpecies(opts, args[0])
		},
	}
	cmd.Flags().StringVar(&opts.panelsDir, "panels-dir", defaultPanels, "Installed panels directory")
	return cmd
}

func newMakeProbesCmd() *cobra.Command {
	opts := &makeProbesOptions{}
	cmd := &cobra.Command{
		Use:   "make-probes <reference-filepath>",
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
	cmd.Flags().StringVarP(&opts.textFile, "text-file", "t", "", "Tab-delimited file containing DNA variants")
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
