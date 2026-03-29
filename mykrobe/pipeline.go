package mykrobe

import "strings"

var DefaultVariantFilters = []string{"MISSING_WT", "LOW_PERCENT_COVERAGE", "LOW_GT_CONF", "LOW_TOTAL_DEPTH"}

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
	Filters                     []string
}

func AnalyzeCoverageSetTB(set *CoverageSet, expectedDepth float64, variantToResistancePath string, lineagePath string) (*AnalysisResult, error) {
	return AnalyzeCoverageSetTBWithOptions(set, AnalysisOptions{
		ExpectedDepth:               expectedDepth,
		VariantToResistancePath:     variantToResistancePath,
		LineagePath:                 lineagePath,
		ErrorRate:                   DefaultErrorRate,
		MinorFreq:                   DefaultMinorFreq,
		VariantConfidenceThreshold:  150,
		SequenceConfidenceThreshold: 1,
		Model:                       "kmer_count",
		KmerSize:                    DefaultKmerSize,
		MinProportionExpectedDepth:  0.3,
		Ploidy:                      "diploid",
		IgnoreMinorCalls:            false,
		MinDepth:                    3,
		Filters:                     DefaultVariantFilters,
	})
}

func AnalyzeCoverageSetTBWithOptions(set *CoverageSet, opts AnalysisOptions) (*AnalysisResult, error) {
	filters := opts.Filters
	if filters == nil {
		filters = DefaultVariantFilters
	}
	vt := NewVariantTyper([]float64{opts.ExpectedDepth}, []float64{}, opts.ErrorRate, opts.MinorFreq, true, filters, opts.VariantConfidenceThreshold, opts.Model, opts.KmerSize, opts.MinProportionExpectedDepth, opts.Ploidy)
	gt := NewGeneCollectionTyper([]float64{opts.ExpectedDepth}, []float64{}, opts.SequenceConfidenceThreshold)

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
	for name, mutated := range predictor.CalledGenes {
		calls, ok := geneCalls[name]
		if !ok || len(calls) == 0 {
			continue
		}
		calls[0] = mutated
		geneCalls[name] = calls
	}
	lineageCalls := map[string]map[string]Call{}
	lineageResult := map[string]any(nil)
	if opts.LineagePath != "" {
		var variantToLineage map[string]LineageVariant
		if err := LoadJSON(opts.LineagePath, &variantToLineage); err != nil {
			return nil, err
		}
		for varName, call := range variantCalls {
			for _, key := range lineageLookupKeys(varName) {
				lineage, ok := variantToLineage[key]
				if !ok {
					continue
				}
				if lineageCalls[lineage.Name] == nil {
					lineageCalls[lineage.Name] = map[string]Call{}
				}
				lineageCalls[lineage.Name][key] = call
				break
			}
		}
		for geneName, call := range flatGeneCalls {
			lineage, ok := variantToLineage[geneName]
			if !ok {
				continue
			}
			if lineageCalls[lineage.Name] == nil {
				lineageCalls[lineage.Name] = map[string]Call{}
			}
			lineageCalls[lineage.Name][geneName] = call
		}
		predictor := NewLineagePredictor(variantToLineage)
		if len(lineageCalls) > 0 {
			lineageResult = predictor.CallLineage(lineageCalls, 0.5)
		}
		lineageCalls = predictor.ApplyReportNamesToLineageCalls(lineageCalls)
	}
	return &AnalysisResult{
		VariantCalls: variantCalls,
		GeneCalls:    geneCalls,
		Predictor:    result,
		Lineage:      lineageResult,
		LineageCalls: lineageCalls,
	}, nil
}

func lineageLookupKeys(varName string) []string {
	keys := []string{varName}
	if i := len("NA_"); strings.HasPrefix(varName, "NA_") && len(varName) > i {
		keys = append(keys, varName[i:])
	}
	if i := strings.IndexByte(varName, '-'); i >= 0 && i+1 < len(varName) {
		keys = append(keys, varName[i+1:])
	}
	return Unique(keys)
}
