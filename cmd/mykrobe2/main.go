package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/martinghunt/mykrobe2/mccortex"
	"github.com/martinghunt/mykrobe2/mykrobe"
	"github.com/martinghunt/mykrobe2/mykrobe/probes"
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
	referencePath string
	vcfPath       string
	variants      []string
	textFile      string
	genbankPath   string
	kmer          int
	noBackgrounds bool
	lineagePath   string
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
	cmd.Flags().StringSliceVarP(&opts.variants, "variants", "v", nil, "Variant in DNA positions e.g. A1234T")
	cmd.Flags().StringVarP(&opts.textFile, "text_file", "t", "", "Tab-delimited file containing DNA variants")
	cmd.Flags().StringVarP(&opts.genbankPath, "genbank", "g", "", "Genbank file containing genes as features")
	cmd.Flags().IntVarP(&opts.kmer, "kmer", "k", mykrobe.DefaultKmerSize, "kmer length")
	cmd.Flags().BoolVar(&opts.noBackgrounds, "no-backgrounds", false, "Ignore nearby variants when building probes")
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

func runPredict(opts *predictOptions) error {
	inputs, err := resolvePredictInputs(opts.panelArg, opts.mapPath, opts.lineagePath, opts.panelsDir, opts.species)
	if err != nil {
		return err
	}
	if opts.seqPath == "" || len(inputs.PanelPaths) == 0 || inputs.MapPath == "" || opts.output == "" {
		return fmt.Errorf("predict requires --seq, a panel source, variant-to-resistance data, and --output")
	}
	k := opts.k
	if k == 0 {
		if inputs.K != 0 {
			k = inputs.K
		} else {
			k = mykrobe.DefaultKmerSize
		}
	}

	counter, err := mccortex.NewCounter(k)
	if err != nil {
		return err
	}
	if err := counter.AddPath(opts.seqPath); err != nil {
		return err
	}
	var summaries []mccortex.CoverageSummary
	if inputs.PanelIndexPath != "" {
		idx, err := mccortex.LoadPanelIndex(inputs.PanelIndexPath)
		if err != nil {
			return err
		}
		summaries, err = counter.SummarizePanelIndex(idx)
		if err != nil {
			return err
		}
	} else {
		summaries, err = summarizePanels(counter, inputs.PanelPaths)
		if err != nil {
			return err
		}
	}
	pr, pw := io.Pipe()
	writeErr := make(chan error, 1)
	go func() {
		err := mccortex.WriteCoverageTSV(pw, summaries)
		if err != nil {
			_ = pw.CloseWithError(err)
			writeErr <- err
			return
		}
		writeErr <- pw.Close()
	}()

	coverageSet, err := mykrobe.ParseCoverageReader(pr)
	if err != nil {
		_ = pr.Close()
		<-writeErr
		return err
	}
	if err := <-writeErr; err != nil {
		return err
	}
	phylo, depths, err := mykrobe.DetectSpeciesAndGetDepths(coverageSet, inputs.HierarchyPath, inputs.SpeciesPhyloGroup)
	if err != nil {
		return err
	}
	if opts.ncbiNames && inputs.NCBINamesPath != "" {
		namesMap := map[string]string{}
		if err := mykrobe.LoadJSON(inputs.NCBINamesPath, &namesMap); err != nil {
			return err
		}
		mykrobe.AddNCBINamesToPhylo(phylo, namesMap)
	}
	expectedDepth := opts.expectedDepth
	if expectedDepth == 0 {
		if len(depths) > 0 {
			expectedDepth = depths[0]
		} else {
			expectedDepth = 100
		}
	}
	errorRate, ploidy := mykrobe.ApplyONTDefaults(opts.errorRate, opts.ploidy, opts.ont)
	analysisOpts := mykrobe.AnalysisOptions{
		ExpectedDepth:               expectedDepth,
		VariantToResistancePath:     inputs.MapPath,
		LineagePath:                 inputs.LineagePath,
		ErrorRate:                   errorRate,
		MinorFreq:                   opts.minorFreq,
		VariantConfidenceThreshold:  opts.minVariantConf,
		SequenceConfidenceThreshold: opts.minGeneConf,
		Model:                       opts.model,
		KmerSize:                    k,
		MinProportionExpectedDepth:  opts.minPropExpectedDepth,
		Ploidy:                      ploidy,
		IgnoreMinorCalls:            opts.ignoreMinorCalls,
		MinDepth:                    opts.minDepth,
	}
	result, err := mykrobe.AnalyzeCoverageSetTBWithOptions(coverageSet, analysisOpts)
	if err != nil {
		return err
	}
	kmerCountErrorRate, incorrectKmerToPCCov := mykrobe.EstimateKmerCountErrorRateAndIncorrectKmerPercentCov(result.VariantCalls, analysisOpts.ErrorRate)
	_, guessedPloidy, _ := mykrobe.GuessSequenceMethod(analysisOpts.ErrorRate, analysisOpts.Ploidy, opts.guessSequenceMethod, kmerCountErrorRate)
	if opts.confPercentCutoff < 100 {
		confThresholder := mykrobe.NewConfThresholder(kmerCountErrorRate, expectedDepth, k, incorrectKmerToPCCov, 10000)
		analysisOpts.ErrorRate = kmerCountErrorRate
		analysisOpts.Ploidy = guessedPloidy
		analysisOpts.VariantConfidenceThreshold = confThresholder.GetConfThreshold(opts.confPercentCutoff)
		result, err = mykrobe.AnalyzeCoverageSetTBWithOptions(coverageSet, analysisOpts)
		if err != nil {
			return err
		}
	}
	if result.Lineage != nil {
		phylo["lineage"] = result.Lineage
	}
	susceptibility := mykrobe.FixAminoAcidXVariantKeysInSusceptibility(result.Predictor.Susceptibility)
	sampleOut := map[string]any{
		"susceptibility": susceptibility,
		"phylogenetics":  phylo,
		"kmer":           k,
		"probe_sets":     inputs.PanelPaths,
		"files":          []string{opts.seqPath},
		"version": map[string]any{
			"mykrobe-predictor": "mykrobe2",
			"mykrobe-atlas":     "mykrobe2",
			"panel":             inputs.PanelVersion,
		},
		"genotype_model": opts.model,
	}
	if opts.reportAllCalls {
		sampleOut["variant_calls"] = mykrobe.FixAminoAcidXVariantKeys(result.VariantCalls)
		sampleOut["sequence_calls"] = result.GeneCalls
		sampleOut["lineage_calls"] = result.LineageCalls
	}
	out := map[string]any{opts.sample: sampleOut}

	f, err := os.Create(opts.output)
	if err != nil {
		return err
	}
	defer f.Close()
	switch opts.outputFormat {
	case "json":
		return mykrobe.WriteJSONLikePython(f, out, "  ")
	case "csv":
		_, err := f.WriteString(formatCSV(out))
		return err
	case "json_and_csv":
		if err := mykrobe.WriteJSONLikePython(f, out, "  "); err != nil {
			return err
		}
		return os.WriteFile(opts.output+".csv", []byte(formatCSV(out)), 0o644)
	default:
		return fmt.Errorf("output format must be one of csv,json,json_and_csv")
	}
}

func runMakeProbes(opts *makeProbesOptions, out io.Writer) error {
	if opts.vcfPath != "" {
		return fmt.Errorf("make-probes --vcf is not implemented yet")
	}
	if opts.genbankPath != "" {
		return fmt.Errorf("make-probes --genbank is not implemented yet")
	}
	if len(opts.variants) == 0 && opts.textFile == "" {
		return fmt.Errorf("make-probes requires --variants or --text_file")
	}
	reference := probes.DefaultReferenceName(opts.referencePath)
	var mutations []probes.Mutation
	lineages := map[string]probes.LineageInfo{}
	if opts.textFile != "" {
		var err error
		mutations, lineages, err = probes.LoadDNAVarsTextFile(opts.textFile, reference)
		if err != nil {
			return err
		}
		if opts.lineagePath != "" {
			data, err := json.MarshalIndent(lineages, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(opts.lineagePath, data, 0o644); err != nil {
				return err
			}
		}
	} else {
		mutations = make([]probes.Mutation, 0, len(opts.variants))
		for _, v := range opts.variants {
			mutations = append(mutations, probes.Mutation{Reference: reference, VarName: v})
		}
	}
	ag, err := probes.NewAlleleGenerator(opts.referencePath, opts.kmer)
	if err != nil {
		return err
	}
	for _, mut := range mutations {
		v, err := mut.Variant()
		if err != nil {
			return err
		}
		panel, err := ag.Create(v, nil)
		if err != nil {
			return err
		}
		for i, ref := range panel.Refs {
			_, err := fmt.Fprintf(out, ">ref-%s?var_name=%s&num_alts=%d&ref=%s&enum=%d&gene=%s&mut=%s\n%s\n",
				mut.MutationOutputName(), mut.VarName, len(panel.Alts), mut.Reference, i, "NA", mut.MutationOutputName(), ref)
			if err != nil {
				return err
			}
		}
		for i, alt := range panel.Alts {
			_, err := fmt.Fprintf(out, ">alt-%s?var_name=%s&enum=%d&gene=%s&mut=%s\n%s\n",
				mut.MutationOutputName(), mut.VarName, i, "NA", mut.MutationOutputName(), alt)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

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

type predictInputs struct {
	PanelPaths        []string
	PanelIndexPath    string
	K                 int
	MapPath           string
	LineagePath       string
	HierarchyPath     string
	NCBINamesPath     string
	SpeciesPhyloGroup string
	PanelVersion      string
}

func resolvePredictInputs(panelArg, mapPath, lineagePath, panelsDir, species string) (predictInputs, error) {
	if panelsDir == "" || species == "" {
		if panelArg == "" {
			return predictInputs{}, fmt.Errorf("predict requires --panel or --panels_dir with --species")
		}
		return predictInputs{
			PanelPaths:   []string{panelArg},
			MapPath:      mapPath,
			LineagePath:  lineagePath,
			PanelVersion: "custom",
		}, nil
	}

	dataDir, err := speciesdata.NewDataDir(panelsDir)
	if err != nil {
		return predictInputs{}, err
	}
	sdir, err := dataDir.GetSpeciesDir(species)
	if err != nil {
		return predictInputs{}, err
	}
	if sdir == nil {
		return predictInputs{}, fmt.Errorf("species %q is not installed in %s", species, panelsDir)
	}
	if panelArg != "" {
		if err := sdir.SetPanel(panelArg); err != nil {
			return predictInputs{}, err
		}
	}
	if _, err := os.Stat(sdir.PanelIndexFile()); err != nil {
		if os.IsNotExist(err) {
			if err := sdir.BuildPanelIndex(); err != nil {
				return predictInputs{}, err
			}
		} else {
			return predictInputs{}, err
		}
	}
	if mapPath == "" {
		mapPath = sdir.JSONFile("amr")
	}
	if lineagePath == "" {
		lineagePath = sdir.JSONFile("lineage")
	}
	return predictInputs{
		PanelPaths:        sdir.FASTAFiles(),
		PanelIndexPath:    sdir.PanelIndexFile(),
		K:                 sdir.Kmer(),
		MapPath:           mapPath,
		LineagePath:       lineagePath,
		HierarchyPath:     sdir.JSONFile("hierarchy"),
		NCBINamesPath:     sdir.JSONFile("ncbi_names"),
		SpeciesPhyloGroup: sdir.SpeciesPhyloGroup(),
		PanelVersion:      sdir.Version() + "/" + sdir.PanelName,
	}, nil
}

type panelSummarizer interface {
	SummarizePanelPath(path string) ([]mccortex.CoverageSummary, error)
}

func summarizePanels(s panelSummarizer, paths []string) ([]mccortex.CoverageSummary, error) {
	out := make([]mccortex.CoverageSummary, 0)
	for _, path := range paths {
		summaries, err := s.SummarizePanelPath(path)
		if err != nil {
			return nil, err
		}
		out = append(out, summaries...)
	}
	return out, nil
}

func formatCSV(out map[string]any) string {
	var b bytes.Buffer
	b.WriteString("sample,drug,predict\n")
	samples := make([]string, 0, len(out))
	for sample := range out {
		samples = append(samples, sample)
	}
	sort.Strings(samples)
	for _, sample := range samples {
		sampleDict, _ := out[sample].(map[string]any)
		susc, _ := sampleDict["susceptibility"].(map[string]map[string]any)
		drugs := make([]string, 0, len(susc))
		for drug := range susc {
			drugs = append(drugs, drug)
		}
		sort.Strings(drugs)
		for _, drug := range drugs {
			b.WriteString(strings.Join([]string{sample, drug, fmt.Sprint(susc[drug]["predict"])}, ","))
			b.WriteByte('\n')
		}
	}
	return b.String()
}
