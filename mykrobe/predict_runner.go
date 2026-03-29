package mykrobe

import (
	"fmt"
	"os"

	"github.com/martinghunt/mykrobe2/mccortex"
	"github.com/martinghunt/mykrobe2/mykrobe/speciesdata"
)

type PredictRunOptions struct {
	Sample               string
	SeqPaths             []string
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
}

type PredictRunResult struct {
	Output            map[string]any
	CoverageSummaries []mccortex.CoverageSummary
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

func RunTBPredict(opts PredictRunOptions) (*PredictRunResult, error) {
	inputs, err := resolvePredictInputs(opts.PanelArg, opts.MapPath, opts.LineagePath, opts.PanelsDir, opts.Species)
	if err != nil {
		return nil, err
	}
	if len(opts.SeqPaths) == 0 || len(inputs.PanelPaths) == 0 || inputs.MapPath == "" {
		return nil, fmt.Errorf("predict requires --seq, a panel source, and variant-to-resistance data")
	}
	k := opts.K
	if k == 0 {
		if inputs.K != 0 {
			k = inputs.K
		} else {
			k = DefaultKmerSize
		}
	}

	var summaries []mccortex.CoverageSummary
	if inputs.PanelIndexPath != "" {
		idx, err := mccortex.LoadPanelIndex(inputs.PanelIndexPath)
		if err != nil {
			return nil, err
		}
		counter, err := mccortex.NewFilteredCounter(k, idx.KmerSet())
		if err != nil {
			return nil, err
		}
		for _, seqPath := range opts.SeqPaths {
			if err := counter.AddPath(seqPath); err != nil {
				return nil, err
			}
		}
		summaries, err = counter.SummarizePanelIndex(idx)
		if err != nil {
			return nil, err
		}
	} else {
		counter, err := mccortex.NewCounter(k)
		if err != nil {
			return nil, err
		}
		for _, seqPath := range opts.SeqPaths {
			if err := counter.AddPath(seqPath); err != nil {
				return nil, err
			}
		}
		summaries, err = summarizePanels(counter, inputs.PanelPaths)
		if err != nil {
			return nil, err
		}
	}

	coverageSet := CoverageSetFromSummaries(summaries)
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
		LineagePath:                 inputs.LineagePath,
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
	if opts.ReportAllCalls {
		sampleOut["variant_calls"] = FixAminoAcidXVariantKeys(result.VariantCalls)
		sampleOut["sequence_calls"] = result.GeneCalls
		sampleOut["lineage_calls"] = result.LineageCalls
	}
	return &PredictRunResult{
		Output:            map[string]any{opts.Sample: sampleOut},
		CoverageSummaries: summaries,
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
