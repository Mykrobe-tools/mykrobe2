package mykrobe

type AnalysisResult struct {
	VariantCalls map[string]Call
	GeneCalls    map[string][]Call
	Predictor    SusceptibilityResult
	Lineage      map[string]any
	LineageCalls map[string]map[string]Call
}

type AnalysisOptions struct {
	ExpectedDepth               float64
	VariantToResistancePath     string
	LineagePath                 string
	ErrorRate                   float64
	MinorFreq                   float64
	VariantConfidenceThreshold  int
	SequenceConfidenceThreshold int
	Model                       string
	KmerSize                    int
	MinProportionExpectedDepth  float64
	Ploidy                      string
	IgnoreMinorCalls            bool
	MinDepth                    float64
}

func AnalyzeCoverageSetTB(set *CoverageSet, expectedDepth float64, variantToResistancePath string, lineagePath string) (*AnalysisResult, error) {
	return AnalyzeCoverageSetTBWithOptions(set, AnalysisOptions{
		ExpectedDepth:               expectedDepth,
		VariantToResistancePath:     variantToResistancePath,
		LineagePath:                 lineagePath,
		ErrorRate:                   DefaultErrorRate,
		MinorFreq:                   DefaultMinorFreq,
		VariantConfidenceThreshold:  3,
		SequenceConfidenceThreshold: 0,
		Model:                       "kmer_count",
		KmerSize:                    DefaultKmerSize,
		MinProportionExpectedDepth:  0.3,
		Ploidy:                      "diploid",
		IgnoreMinorCalls:            false,
		MinDepth:                    3,
	})
}

func AnalyzeCoverageSetTBWithOptions(set *CoverageSet, opts AnalysisOptions) (*AnalysisResult, error) {
	vt := NewVariantTyper([]float64{opts.ExpectedDepth}, nil, opts.ErrorRate, opts.MinorFreq, false, nil, opts.VariantConfidenceThreshold, opts.Model, opts.KmerSize, opts.MinProportionExpectedDepth, opts.Ploidy)
	gt := NewGeneCollectionTyper([]float64{opts.ExpectedDepth}, nil, opts.SequenceConfidenceThreshold)

	variantCalls := make(map[string]Call, len(set.Variant))
	geneCalls := make(map[string][]Call, len(set.Presence))

	flatGeneCalls := map[string]Call{}
	for name, cov := range set.Variant {
		variantCalls[name] = vt.Type(cov)
	}
	for name, versions := range set.Presence {
		calls := gt.Type(versions, 100)
		geneCalls[name] = calls
		if len(calls) > 0 {
			flatGeneCalls[name] = calls[0]
		}
	}

	predictor, err := NewTBPredictor(variantCalls, flatGeneCalls, opts.VariantToResistancePath)
	if err != nil {
		return nil, err
	}
	predictor.DepthThreshold = opts.MinDepth
	predictor.IgnoreMinorCalls = opts.IgnoreMinorCalls
	result := predictor.Run()
	lineageCalls := map[string]map[string]Call{}
	lineageResult := map[string]any(nil)
	if opts.LineagePath != "" {
		var variantToLineage map[string]LineageVariant
		if err := LoadJSON(opts.LineagePath, &variantToLineage); err != nil {
			return nil, err
		}
		for varName, call := range variantCalls {
			if lineage, ok := variantToLineage[varName]; ok {
				if lineageCalls[lineage.Name] == nil {
					lineageCalls[lineage.Name] = map[string]Call{}
				}
				lineageCalls[lineage.Name][varName] = call
			}
		}
		if len(lineageCalls) > 0 {
			lineageResult = NewLineagePredictor(variantToLineage).CallLineage(lineageCalls, 0.5)
		}
	}
	return &AnalysisResult{
		VariantCalls: variantCalls,
		GeneCalls:    geneCalls,
		Predictor:    result,
		Lineage:      lineageResult,
		LineageCalls: lineageCalls,
	}, nil
}
