package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/martinghunt/mykrobe2/mccortex"
	"github.com/martinghunt/mykrobe2/mykrobe"
	"github.com/martinghunt/mykrobe2/mykrobe/speciesdata"
)

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
	if opts.writeCovgs != "" {
		f, err := os.Create(opts.writeCovgs)
		if err != nil {
			return err
		}
		if err := mccortex.WriteCoverageTSVWithHeader(f, summaries); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	coverageSet, err := coverageSetFromSummaries(summaries)
	if err != nil {
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

func coverageSetFromSummaries(summaries []mccortex.CoverageSummary) (*mykrobe.CoverageSet, error) {
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
		return nil, err
	}
	if err := <-writeErr; err != nil {
		return nil, err
	}
	return coverageSet, nil
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
