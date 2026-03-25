package mykrobe

import (
	"encoding/json"
	"math"
	"slices"
	"sort"
)

type Call struct {
	Class               string         `json:"_cls,omitempty"`
	Variant             any            `json:"variant,omitempty"`
	Genotype            []int          `json:"genotype"`
	GenotypeLikelihoods []float64      `json:"genotype_likelihoods"`
	Info                map[string]any `json:"info"`
}

func (c Call) MarshalJSON() ([]byte, error) {
	out := map[string]any{
		"genotype":             c.Genotype,
		"genotype_likelihoods": c.GenotypeLikelihoods,
		"info":                 c.Info,
	}
	if c.Class != "" {
		out["_cls"] = c.Class
	}
	if c.Variant != nil || c.Class == "Call.VariantCall" {
		out["variant"] = c.Variant
	}
	return json.Marshal(out)
}

type Typer struct {
	ExpectedDepths      []float64
	ContaminationDepths []float64
	ErrorRate           float64
	IgnoreFiltered      bool
	Filters             []string
	ConfidenceThreshold int
}

func (t Typer) HasContamination() bool {
	return len(t.ContaminationDepths) > 0 || len(t.ExpectedDepths) > 1
}

func (t Typer) LikelihoodsToGenotype(likelihoods []float64) string {
	ml := likelihoods[0]
	i := 0
	for j := range likelihoods {
		if likelihoods[j] > ml {
			ml = likelihoods[j]
			i = j
		}
	}
	switch i {
	case 0:
		if ml <= MinLLK {
			return "-/-"
		}
		return "0/0"
	case 1:
		return "0/1"
	default:
		return "1/1"
	}
}

type PresenceTyper struct {
	Typer
}

func NewPresenceTyper(expectedDepths []float64, contaminationDepths []float64, confidenceThreshold int) PresenceTyper {
	return PresenceTyper{Typer: Typer{
		ExpectedDepths:      expectedDepths,
		ContaminationDepths: contaminationDepths,
		ErrorRate:           DefaultErrorRate,
		ConfidenceThreshold: confidenceThreshold,
	}}
}

func (p PresenceTyper) MinimumDetectableFrequency() float64 {
	if p.ErrorRate < 0.1 {
		return 0.05
	}
	return 0.25
}

func (p PresenceTyper) Type(s SequenceProbeCoverage) Call {
	homAltLikelihoods := []float64{}
	hetLikelihoods := []float64{}
	homRefLikelihoods := []float64{}
	for _, expectedDepth := range p.ExpectedDepths {
		homAltLikelihoods = append(homAltLikelihoods, LogLikDepth(s.MedianDepth(), expectedDepth*0.75))
		if !p.HasContamination() {
			hetLikelihoods = append(hetLikelihoods, LogLikDepth(s.MedianDepth(), expectedDepth*p.MinimumDetectableFrequency()))
		} else {
			hetLikelihoods = append(hetLikelihoods, MinLLK)
		}
		homRefLikelihoods = append(homRefLikelihoods, LogLikDepth(s.MedianDepth(), expectedDepth*0.001))
		for _, contaminationDepth := range p.ContaminationDepths {
			homAltLikelihoods = append(homAltLikelihoods, LogLikDepth(s.MedianDepth(), expectedDepth+contaminationDepth*0.75))
			homRefLikelihoods = append(homRefLikelihoods, LogLikDepth(s.MedianDepth(), contaminationDepth))
		}
	}
	expectedDepth := p.ExpectedDepths[0]
	homRefLikelihood := maxFloat(homRefLikelihoods)
	homAltLikelihood := p.logPostHetOrAlt(maxFloat(homAltLikelihoods), expectedDepth*0.75, s)
	hetLikelihood := p.logPostHetOrAlt(maxFloat(hetLikelihoods), expectedDepth*p.MinimumDetectableFrequency(), s)
	likelihoods := []float64{homRefLikelihood, hetLikelihood, homAltLikelihood}
	gt := p.LikelihoodsToGenotype(likelihoods)
	info := map[string]any{
		"copy_number":          s.MedianDepth() / expectedDepth,
		"coverage":             s.CoverageDict(),
		"expected_depths":      p.ExpectedDepths,
		"contamination_depths": p.ContaminationDepths,
	}
	if gt != "0/0" && gt != "-/-" {
		info["version"] = s.Version
	}
	if s.Length != "" {
		info["length"] = s.Length
	}
	return Call{
		Class:               "Call.SequenceCall",
		Genotype:            parseGT(gt),
		GenotypeLikelihoods: likelihoods,
		Info:                info,
	}
}

func (p PresenceTyper) logPostHetOrAlt(llk, expectedDepth float64, s SequenceProbeCoverage) float64 {
	expectedPct := PercentCoverageFromExpectedCoverage(expectedDepth)
	minPctRequired := expectedPct * s.PercentCoverageThreshold
	if s.PercentCoverage() > minPctRequired {
		return llk
	}
	return MinLLK
}

type GeneCollectionTyper struct {
	Typer
	PresenceTyper PresenceTyper
}

func NewGeneCollectionTyper(expectedDepths []float64, contaminationDepths []float64, confidenceThreshold int) GeneCollectionTyper {
	return GeneCollectionTyper{
		Typer:         Typer{ExpectedDepths: expectedDepths, ContaminationDepths: contaminationDepths, ConfidenceThreshold: confidenceThreshold},
		PresenceTyper: NewPresenceTyper(expectedDepths, contaminationDepths, confidenceThreshold),
	}
}

func (g GeneCollectionTyper) Type(collection map[string]SequenceProbeCoverage, minGenePercentCovgThreshold float64) []Call {
	best := g.GetBestVersion(collection, minGenePercentCovgThreshold)
	out := make([]Call, 0, len(best))
	for _, spc := range best {
		out = append(out, g.PresenceTyper.Type(spc))
	}
	return out
}

func (g GeneCollectionTyper) GetBestVersion(collection map[string]SequenceProbeCoverage, threshold float64) []SequenceProbeCoverage {
	items := make([]SequenceProbeCoverage, 0, len(collection))
	for _, v := range collection {
		items = append(items, v)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].PercentCoverage() > items[j].PercentCoverage() })
	best := []SequenceProbeCoverage{items[0]}
	for _, gene := range items[1:] {
		if gene.PercentCoverage() >= threshold {
			best = append(best, gene)
		} else {
			return best
		}
	}
	return best
}

func LikelihoodsToConfidence(l []float64) int {
	sorted := slices.Clone(l)
	sort.Sort(sort.Reverse(sort.Float64Slice(sorted)))
	if sorted[2] == l[0] {
		return int(math.Round(sorted[0] - l[0]))
	}
	return int(math.Round(sorted[0] - sorted[1]))
}

type VariantTyper struct {
	Typer
	MinorFreq                  float64
	KmerSize                   int
	MinProportionExpectedDepth float64
	Ploidy                     string
	Model                      string
}

func NewVariantTyper(expectedDepths, contaminationDepths []float64, errorRate, minorFreq float64, ignoreFiltered bool, filters []string, confidenceThreshold int, model string, kmerSize int, minProp float64, ploidy string) VariantTyper {
	return VariantTyper{
		Typer: Typer{
			ExpectedDepths:      expectedDepths,
			ContaminationDepths: contaminationDepths,
			ErrorRate:           errorRate,
			IgnoreFiltered:      ignoreFiltered,
			Filters:             filters,
			ConfidenceThreshold: confidenceThreshold,
		},
		MinorFreq:                  minorFreq,
		KmerSize:                   kmerSize,
		MinProportionExpectedDepth: minProp,
		Ploidy:                     ploidy,
		Model:                      model,
	}
}

func (v VariantTyper) diploid() bool { return v.Ploidy == "diploid" }

func (v VariantTyper) Type(items ...*VariantProbeCoverage) Call {
	if len(items) == 1 {
		return v.typeOne(items[0], items[0].VarName)
	}
	calls := make([]Call, 0, len(items))
	for _, item := range items {
		calls = append(calls, v.typeOne(item, item.VarName))
	}
	sort.Slice(calls, func(i, j int) bool {
		return calls[i].Info["conf"].(int) > calls[j].Info["conf"].(int)
	})
	for _, call := range calls {
		sum := call.Genotype[0] + call.Genotype[1]
		if sum > 1 {
			return call
		}
	}
	for _, call := range calls {
		sum := call.Genotype[0] + call.Genotype[1]
		if sum == 1 {
			return call
		}
	}
	return calls[0]
}

func (v VariantTyper) typeOne(cov *VariantProbeCoverage, variant any) Call {
	cov.ensureBest()
	homRef := v.homRefLik(cov)
	homAlt := v.homAltLik(cov)
	het := MinLLK
	if !v.HasContamination() && v.diploid() {
		het = v.hetLik(cov)
	}
	likelihoods := []float64{homRef, het, homAlt}
	conf := LikelihoodsToConfidence(likelihoods)
	gt := v.LikelihoodsToGenotype(likelihoods)
	info := map[string]any{
		"coverage":             cov.CoverageDict(),
		"expected_depths":      v.ExpectedDepths,
		"contamination_depths": v.ContaminationDepths,
		"filter":               []string{},
		"conf":                 conf,
	}
	if gt == "-/-" && !v.IgnoreFiltered {
		if cov.BestAlternateCoverage.PercentCoverage > cov.BestReferenceCoverage.PercentCoverage {
			gt = "1/1"
		} else {
			gt = "0/0"
		}
		if contains(v.Filters, "MISSING_WT") {
			info["filter"] = append(info["filter"].([]string), "MISSING_WT")
		}
	} else if contains(v.Filters, "LOW_PERCENT_COVERAGE") && cov.BestAlternateCoverage.PercentCoverage < 100 && cov.BestReferenceCoverage.PercentCoverage < 100 {
		info["filter"] = append(info["filter"].([]string), "LOW_PERCENT_COVERAGE")
		if v.IgnoreFiltered {
			gt = "0/0"
		}
	}
	if contains(v.Filters, "LOW_GT_CONF") && conf < v.ConfidenceThreshold {
		info["filter"] = append(info["filter"].([]string), "LOW_GT_CONF")
	}
	if contains(v.Filters, "LOW_TOTAL_DEPTH") {
		totalDepth := cov.BestReferenceCoverage.MedianDepth + cov.BestAlternateCoverage.MedianDepth
		if totalDepth < v.MinProportionExpectedDepth*v.ExpectedDepths[0] {
			info["filter"] = append(info["filter"].([]string), "LOW_TOTAL_DEPTH")
		}
	}
	gl := likelihoods
	if !v.diploid() {
		gl = []float64{likelihoods[0], likelihoods[2]}
	}
	return Call{Class: "Call.VariantCall", Variant: variant, Genotype: parseGT(gt), GenotypeLikelihoods: gl, Info: info}
}

func (v VariantTyper) depthToExpectedKmerCount(depth float64, alleleLength int) float64 {
	return float64(alleleLength)*depth + 0.01
}
func (v VariantTyper) klenToDNAlen(klen int) int { return klen + v.KmerSize - 1 }

func (v VariantTyper) homRefLik(c *VariantProbeCoverage) float64 {
	if v.Model == "median_depth" {
		if c.BestReferenceCoverage.PercentCoverage < 100*PercentCoverageFromExpectedCoverage(maxFloat(v.ExpectedDepths)) {
			return MinLLK
		}
		likes := []float64{}
		for _, expectedDepth := range v.ExpectedDepths {
			likes = append(likes, LogLikRScoverage(c.BestReferenceCoverage.MedianDepth, c.BestAlternateCoverage.MedianDepth, expectedDepth, expectedDepth*v.ErrorRate/3))
		}
		return maxFloat(likes)
	}
	likes := []float64{}
	for _, expectedDepth := range v.ExpectedDepths {
		kmerLike := LogLikRSkmerCount(
			float64(c.BestReferenceCoverage.KCount), float64(c.BestAlternateCoverage.KCount),
			v.depthToExpectedKmerCount(expectedDepth, c.BestReferenceCoverage.KLen),
			v.depthToExpectedKmerCount(expectedDepth*v.ErrorRate/3, c.BestAlternateCoverage.KLen),
		)
		gaps := LogLikProbabilityOfNGaps(expectedDepth, c.BestReferenceCoverage.PercentCoverage, v.klenToDNAlen(c.BestReferenceCoverage.KLen)) +
			LogLikProbabilityOfNGaps(expectedDepth*v.ErrorRate/3, c.BestAlternateCoverage.PercentCoverage, v.klenToDNAlen(c.BestAlternateCoverage.KLen))
		likes = append(likes, kmerLike+gaps)
	}
	return maxFloat(likes)
}

func (v VariantTyper) homAltLik(c *VariantProbeCoverage) float64 {
	if v.Model == "median_depth" {
		if c.BestAlternateCoverage.PercentCoverage < 100*PercentCoverageFromExpectedCoverage(maxFloat(v.ExpectedDepths)) {
			return MinLLK
		}
		likes := []float64{}
		for _, expectedDepth := range v.ExpectedDepths {
			likes = append(likes, LogLikRScoverage(c.BestAlternateCoverage.MedianDepth, c.BestReferenceCoverage.MedianDepth, expectedDepth, expectedDepth*v.ErrorRate/3))
		}
		return maxFloat(likes)
	}
	likes := []float64{}
	for _, expectedDepth := range v.ExpectedDepths {
		kmerLike := LogLikRSkmerCount(
			float64(c.BestAlternateCoverage.KCount), float64(c.BestReferenceCoverage.KCount),
			v.depthToExpectedKmerCount(expectedDepth, c.BestReferenceCoverage.KLen),
			v.depthToExpectedKmerCount(expectedDepth*v.ErrorRate/3, c.BestAlternateCoverage.KLen),
		)
		gaps := LogLikProbabilityOfNGaps(expectedDepth*v.ErrorRate/3, c.BestReferenceCoverage.PercentCoverage, v.klenToDNAlen(c.BestReferenceCoverage.KLen)) +
			LogLikProbabilityOfNGaps(expectedDepth, c.BestAlternateCoverage.PercentCoverage, v.klenToDNAlen(c.BestAlternateCoverage.KLen))
		likes = append(likes, kmerLike+gaps)
	}
	return maxFloat(likes)
}

func (v VariantTyper) hetLik(c *VariantProbeCoverage) float64 {
	if c.BestAlternateCoverage.PercentCoverage < 100 || c.BestReferenceCoverage.PercentCoverage < 100 {
		return MinLLK
	}
	if v.Model == "median_depth" {
		likes := []float64{}
		for _, expectedDepth := range v.ExpectedDepths {
			likes = append(likes, LogLikRScoverage(c.BestAlternateCoverage.MedianDepth, c.BestReferenceCoverage.MedianDepth, expectedDepth*v.MinorFreq, expectedDepth*(1-v.MinorFreq)))
		}
		return maxFloat(likes)
	}
	if c.BestAlternateCoverage.KCount+c.BestReferenceCoverage.KCount == 0 {
		return MinLLK
	}
	likes := []float64{}
	for _, expectedDepth := range v.ExpectedDepths {
		kmerLike := LogLikRSkmerCount(
			float64(c.BestAlternateCoverage.KCount), float64(c.BestReferenceCoverage.KCount),
			v.depthToExpectedKmerCount(expectedDepth/2+(expectedDepth/2*v.ErrorRate/3), c.BestAlternateCoverage.KLen),
			v.depthToExpectedKmerCount(expectedDepth/2+(expectedDepth/2*v.ErrorRate/3), c.BestReferenceCoverage.KLen),
		)
		gaps := LogLikProbabilityOfNGaps(expectedDepth/2+(expectedDepth/2*v.ErrorRate/3), c.BestReferenceCoverage.PercentCoverage, v.klenToDNAlen(c.BestReferenceCoverage.KLen)) +
			LogLikProbabilityOfNGaps(expectedDepth/2+(expectedDepth/2*v.ErrorRate/3), c.BestAlternateCoverage.PercentCoverage, v.klenToDNAlen(c.BestAlternateCoverage.KLen))
		likes = append(likes, kmerLike+gaps)
	}
	return maxFloat(likes)
}

func parseGT(gt string) []int {
	switch gt {
	case "0/0":
		return []int{0, 0}
	case "0/1":
		return []int{0, 1}
	case "1/1":
		return []int{1, 1}
	default:
		return []int{}
	}
}

func maxFloat(xs []float64) float64 {
	best := xs[0]
	for _, x := range xs[1:] {
		if x > best {
			best = x
		}
	}
	return best
}

func contains(xs []string, target string) bool {
	for _, x := range xs {
		if x == target {
			return true
		}
	}
	return false
}
