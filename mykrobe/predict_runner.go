package mykrobe

import (
	"fmt"
	"os"

	"github.com/Mykrobe-tools/mykrobe2/mccortex"
	"github.com/Mykrobe-tools/mykrobe2/mykrobe/speciesdata"
)

type PredictRunOptions struct {
	Sample               string
	SeqPaths             []string
	IndexPath            string
	PanelArg             string
	MapPath              string
	LineagePath          string
	PanelsDir            string
	Species              string
	Model                string
	Ploidy               string
	K                    int
	ExpectedDepth        float64
	MinDepth             float64
	ErrorRate            float64
	MinorFreq            float64
	MinPropExpectedDepth float64
	MinVariantConf       int
	MinGeneConf          int
	ReportAllCalls       bool
	IgnoreMinorCalls     bool
	NCBINames            bool
	ONT                  bool
	GuessSequenceMethod  bool
	ConfPercentCutoff    float64
	Progress             PredictProgressFunc
}

type PredictRunResult struct {
	Output            map[string]any
	CoverageSummaries []mccortex.CoverageSummary
}

type predictInputs struct {
	PanelPaths        []string
	RuntimeIndexPath  string
	K                 int
	MapPath           string
	MapData           []byte
	LineagePath       string
	LineageData       []byte
	HierarchyPath     string
	NCBINamesPath     string
	SpeciesPhyloGroup string
	PanelVersion      string
	IsCustom          bool
}

func RunTBPredict(opts PredictRunOptions) (*PredictRunResult, error) {
	reportPredictProgress(opts.Progress, PredictStageLoadingPanel, "Loading panel data", 0, false)
	inputs, err := resolvePredictInputs(opts.IndexPath, opts.PanelArg, opts.MapPath, opts.LineagePath, opts.PanelsDir, opts.Species)
	if err != nil {
		return nil, err
	}
	if len(opts.SeqPaths) == 0 || inputs.RuntimeIndexPath == "" {
		return nil, fmt.Errorf("predict requires --seq and a panel source")
	}
	k := opts.K
	if k == 0 {
		if inputs.K != 0 {
			k = inputs.K
		} else {
			k = DefaultKmerSize
		}
	}

	idx, err := mccortex.LoadRuntimeIndex(inputs.RuntimeIndexPath)
	if err != nil {
		return nil, err
	}
	defer idx.Close()
	counter, err := mccortex.NewRuntimeCounter(idx)
	if err != nil {
		return nil, err
	}
	totalBytes, hasTotalBytes := sequenceInputSize(opts.SeqPaths)
	reportPredictProgress(opts.Progress, PredictStageProcessingReads, "Processing reads", 0, hasTotalBytes)
	var completedBytes int64
	lastReportedFraction := -1.0
	for _, seqPath := range opts.SeqPaths {
		pathStart := completedBytes
		var progress func(int64)
		if hasTotalBytes {
			progress = func(pathBytes int64) {
				fraction := float64(pathStart+pathBytes) / float64(totalBytes)
				if fraction-lastReportedFraction >= 0.005 || fraction >= 1 {
					lastReportedFraction = fraction
					reportPredictProgress(opts.Progress, PredictStageProcessingReads, "Processing reads", fraction, true)
				}
			}
		}
		if err := counter.AddPathWithProgress(seqPath, progress); err != nil {
			return nil, err
		}
		if hasTotalBytes {
			info, err := os.Stat(seqPath)
			if err != nil {
				return nil, err
			}
			completedBytes += info.Size()
		}
	}
	reportPredictProgress(opts.Progress, PredictStageProcessingReads, "Processing reads", 1, hasTotalBytes)
	reportPredictProgress(opts.Progress, PredictStageCalculatingCoverage, "Calculating coverage", 0, false)
	summaries := counter.Summaries()

	coverageSet := CoverageSetFromSummaries(summaries)
	reportPredictProgress(opts.Progress, PredictStageIdentifyingSpecies, "Identifying species", 0, false)
	phylo, depths, err := DetectSpeciesAndGetDepths(coverageSet, inputs.HierarchyPath, inputs.SpeciesPhyloGroup)
	if err != nil {
		return nil, err
	}
	if opts.NCBINames && inputs.NCBINamesPath != "" {
		namesMap := map[string]string{}
		if err := LoadJSON(inputs.NCBINamesPath, &namesMap); err != nil {
			return nil, err
		}
		AddNCBINamesToPhylo(phylo, namesMap)
	}
	expectedDepth := opts.ExpectedDepth
	if expectedDepth == 0 {
		if len(depths) > 0 {
			expectedDepth = depths[0]
		} else {
			expectedDepth = 100
		}
	}
	errorRate, ploidy := ApplyONTDefaults(opts.ErrorRate, opts.Ploidy, opts.ONT)
	analysisOpts := AnalysisOptions{
		ExpectedDepth:               expectedDepth,
		VariantToResistancePath:     inputs.MapPath,
		VariantToResistanceData:     inputs.MapData,
		LineagePath:                 inputs.LineagePath,
		LineageData:                 inputs.LineageData,
		ErrorRate:                   errorRate,
		MinorFreq:                   opts.MinorFreq,
		VariantConfidenceThreshold:  opts.MinVariantConf,
		SequenceConfidenceThreshold: opts.MinGeneConf,
		Model:                       opts.Model,
		KmerSize:                    k,
		MinProportionExpectedDepth:  opts.MinPropExpectedDepth,
		Ploidy:                      ploidy,
		IgnoreMinorCalls:            opts.IgnoreMinorCalls,
		MinDepth:                    opts.MinDepth,
	}
	reportPredictProgress(opts.Progress, PredictStagePredictingResistance, "Predicting resistance", 0, false)
	result, err := AnalyzeCoverageSetTBWithOptions(coverageSet, analysisOpts)
	if err != nil {
		return nil, err
	}
	kmerCountErrorRate, incorrectKmerToPCCov := EstimateKmerCountErrorRateAndIncorrectKmerPercentCov(result.VariantCalls, analysisOpts.ErrorRate)
	_, guessedPloidy, _ := GuessSequenceMethod(analysisOpts.ErrorRate, analysisOpts.Ploidy, opts.GuessSequenceMethod, kmerCountErrorRate)
	if opts.ConfPercentCutoff < 100 {
		confThresholder := NewConfThresholder(kmerCountErrorRate, expectedDepth, k, incorrectKmerToPCCov, 10000)
		analysisOpts.ErrorRate = kmerCountErrorRate
		analysisOpts.Ploidy = guessedPloidy
		analysisOpts.VariantConfidenceThreshold = confThresholder.GetConfThreshold(opts.ConfPercentCutoff)
		result, err = AnalyzeCoverageSetTBWithOptions(coverageSet, analysisOpts)
		if err != nil {
			return nil, err
		}
	}
	if result.Lineage != nil {
		phylo["lineage"] = result.Lineage
	}
	susceptibility := FixAminoAcidXVariantKeysInSusceptibility(result.Predictor.Susceptibility)
	reportAllCalls := opts.ReportAllCalls || (inputs.IsCustom && inputs.MapPath == "" && len(inputs.MapData) == 0 && inputs.LineagePath == "" && len(inputs.LineageData) == 0)
	sampleOut := map[string]any{
		"susceptibility": susceptibility,
		"phylogenetics":  phylo,
		"kmer":           k,
		"probe_sets":     inputs.PanelPaths,
		"files":          append([]string(nil), opts.SeqPaths...),
		"version": map[string]any{
			"mykrobe-predictor": "mykrobe2",
			"mykrobe-atlas":     "mykrobe2",
			"panel":             inputs.PanelVersion,
		},
		"genotype_model": opts.Model,
	}
	if reportAllCalls {
		sampleOut["variant_calls"] = FixAminoAcidXVariantKeys(result.VariantCalls)
		sampleOut["sequence_calls"] = result.GeneCalls
		sampleOut["lineage_calls"] = result.LineageCalls
	}
	return &PredictRunResult{
		Output:            map[string]any{opts.Sample: sampleOut},
		CoverageSummaries: summaries,
	}, nil
}

func sequenceInputSize(paths []string) (int64, bool) {
	var total int64
	for _, path := range paths {
		if path == "-" {
			return 0, false
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
			return 0, false
		}
		total += info.Size()
	}
	return total, total > 0
}

func resolvePredictInputs(indexPath, panelArg, mapPath, lineagePath, panelsDir, species string) (predictInputs, error) {
	if indexPath != "" {
		bundle, err := LoadCustomIndex(indexPath)
		if err != nil {
			return predictInputs{}, err
		}
		defer bundle.Close()
		return predictInputs{
			PanelPaths:       append([]string(nil), bundle.ProbeSets...),
			RuntimeIndexPath: indexPath,
			K:                bundle.RuntimeIndex.K,
			MapData:          append([]byte(nil), bundle.VariantToResistance...),
			LineageData:      append([]byte(nil), bundle.Lineage...),
			PanelVersion:     bundle.PanelVersion,
			IsCustom:         true,
		}, nil
	}
	if species == "custom" {
		return predictInputs{}, fmt.Errorf("predict with --species custom requires --index")
	}
	if panelsDir == "" || species == "" {
		return predictInputs{}, fmt.Errorf("predict requires --index, or --panels-dir with --species")
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
	if _, err := os.Stat(sdir.RuntimeIndexFile()); err != nil {
		if os.IsNotExist(err) {
			if err := sdir.BuildRuntimeIndex(); err != nil {
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
		RuntimeIndexPath:  sdir.RuntimeIndexFile(),
		K:                 sdir.Kmer(),
		MapPath:           mapPath,
		LineagePath:       lineagePath,
		HierarchyPath:     sdir.JSONFile("hierarchy"),
		NCBINamesPath:     sdir.JSONFile("ncbi_names"),
		SpeciesPhyloGroup: sdir.SpeciesPhyloGroup(),
		PanelVersion:      sdir.Version() + "/" + sdir.PanelName,
	}, nil
}
