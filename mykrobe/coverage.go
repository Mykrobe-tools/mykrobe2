package mykrobe

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Mykrobe-tools/mykrobe2/mccortex"
)

type ProbeCoverage struct {
	PercentCoverage float64
	MedianDepth     float64
	MinDepth        float64
	KCount          int
	KLen            int
}

func (p ProbeCoverage) CoverageDict() map[string]any {
	return map[string]any{
		"percent_coverage":   round2(p.PercentCoverage),
		"median_depth":       round2(p.MedianDepth),
		"min_non_zero_depth": round2(p.MinDepth),
		"kmer_count":         p.KCount,
		"klen":               p.KLen,
	}
}

type SequenceProbeCoverage struct {
	Name                     string
	ProbeCoverage            ProbeCoverage
	PercentCoverageThreshold float64
	Version                  string
	Length                   string
}

func (s SequenceProbeCoverage) MedianDepth() float64     { return s.ProbeCoverage.MedianDepth }
func (s SequenceProbeCoverage) PercentCoverage() float64 { return s.ProbeCoverage.PercentCoverage }
func (s SequenceProbeCoverage) MinDepth() float64        { return s.ProbeCoverage.MinDepth }
func (s SequenceProbeCoverage) CoverageDict() map[string]any {
	return s.ProbeCoverage.CoverageDict()
}

type VariantProbeCoverage struct {
	ReferenceCoverages []ProbeCoverage
	AlternateCoverages []ProbeCoverage
	VarName            string
	Params             map[string]string

	BestReferenceCoverage *ProbeCoverage
	BestAlternateCoverage *ProbeCoverage
}

func (v *VariantProbeCoverage) chooseBestCoverage(coverages []ProbeCoverage) *ProbeCoverage {
	sorted := make([]ProbeCoverage, len(coverages))
	copy(sorted, coverages)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].KCount > sorted[j].KCount
	})
	best := sorted[0]
	for _, cov := range sorted[1:] {
		if cov.KCount < best.KCount {
			continue
		}
		if cov.PercentCoverage > best.PercentCoverage {
			best = cov
		} else if cov.MinDepth > best.MinDepth {
			best = cov
		} else if cov.MinDepth <= best.MinDepth && cov.MedianDepth > best.MedianDepth {
			best = cov
		}
	}
	return &best
}

func (v *VariantProbeCoverage) ChooseBestAlternateCoverage() *ProbeCoverage {
	best := v.chooseBestCoverage(v.AlternateCoverages)
	v.BestAlternateCoverage = best
	return best
}

func (v *VariantProbeCoverage) ChooseBestReferenceCoverage() *ProbeCoverage {
	best := v.chooseBestCoverage(v.ReferenceCoverages)
	v.BestReferenceCoverage = best
	return best
}

func (v *VariantProbeCoverage) ensureBest() {
	if v.BestReferenceCoverage == nil && len(v.ReferenceCoverages) > 0 {
		v.ChooseBestReferenceCoverage()
	}
	if v.BestAlternateCoverage == nil && len(v.AlternateCoverages) > 0 {
		v.ChooseBestAlternateCoverage()
	}
}

func (v *VariantProbeCoverage) CoverageDict() map[string]any {
	v.ensureBest()
	return map[string]any{
		"reference": v.BestReferenceCoverage.CoverageDict(),
		"alternate": v.BestAlternateCoverage.CoverageDict(),
	}
}

type CoverageSet struct {
	Variant  map[string]*VariantProbeCoverage
	Presence map[string]map[string]SequenceProbeCoverage
	Groups   map[string]map[string]*TaxonCoverage
}

type TaxonCoverage struct {
	TotalBases      int
	PercentCoverage []float64
	Length          []int
	Median          []float64
}

type Panel struct {
	FilePath string
	Name     string
}

func NewPanel(path string) Panel {
	base := filepath.Base(path)
	return Panel{FilePath: path, Name: strings.Split(base, ".")[0]}
}

func ParseCoverageFile(path string) (*CoverageSet, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseCoverageReader(f)
}

func ParseCoverageReader(r io.Reader) (*CoverageSet, error) {
	set := &CoverageSet{
		Variant:  map[string]*VariantProbeCoverage{},
		Presence: map[string]map[string]SequenceProbeCoverage{},
		Groups:   map[string]map[string]*TaxonCoverage{},
	}
	reader := csv.NewReader(r)
	reader.Comma = '\t'
	for {
		row, err := reader.Read()
		if err == io.EOF {
			return set, nil
		}
		if err != nil {
			return nil, err
		}
		allele, medianDepth, minDepth, percentCoverage, kCount, kLen := parseSummaryRow(row)
		alleleName := strings.Split(allele, "?")[0]
		if isVariantPanel(alleleName) {
			parseVariantRow(set, row, allele, medianDepth, minDepth, percentCoverage, kCount, kLen)
		} else {
			parseSeqRow(set, allele, medianDepth, minDepth, percentCoverage, kCount, kLen)
		}
	}
}

func CoverageSetFromSummaries(summaries []mccortex.CoverageSummary) *CoverageSet {
	set := &CoverageSet{
		Variant:  map[string]*VariantProbeCoverage{},
		Presence: map[string]map[string]SequenceProbeCoverage{},
		Groups:   map[string]map[string]*TaxonCoverage{},
	}
	for _, s := range summaries {
		medianDepth := float64(s.MedianDepth)
		minDepth := float64(s.MinDepth)
		percentCoverage := 100 * quantizeCoverageRatioLikeTSV(s.PercentCoverage)
		kCount := int(s.KmerCount)
		kLen := s.KmerLength
		alleleName := strings.Split(s.Name, "?")[0]
		if isVariantPanel(alleleName) {
			parseVariantRow(set, nil, s.Name, medianDepth, minDepth, percentCoverage, kCount, kLen)
		} else {
			parseSeqRow(set, s.Name, medianDepth, minDepth, percentCoverage, kCount, kLen)
		}
	}
	return set
}

func quantizeCoverageRatioLikeTSV(v float64) float64 {
	s := fmt.Sprintf("%f", v)
	out, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return v
	}
	return out
}

func parseSummaryRow(row []string) (string, float64, float64, float64, int, int) {
	if len(row) < 7 {
		return row[0], 0, 0, 0, 0, 0
	}
	medianDepth, _ := strconv.Atoi(row[2])
	minDepth, _ := strconv.Atoi(row[3])
	percentCoverage, _ := strconv.ParseFloat(row[4], 64)
	kCount, _ := strconv.Atoi(row[5])
	kLen, _ := strconv.Atoi(row[6])
	return row[0], float64(medianDepth), float64(minDepth), 100 * percentCoverage, kCount, kLen
}

func isVariantPanel(allele string) bool {
	parts := strings.Split(allele, "-")
	if len(parts) == 0 {
		return false
	}
	return parts[0] == "ref" || parts[0] == "alt"
}

func parseSeqRow(set *CoverageSet, allele string, medianDepth, minDepth, percentCoverage float64, kCount, kLen int) {
	pc := ProbeCoverage{
		PercentCoverage: percentCoverage,
		MedianDepth:     medianDepth,
		MinDepth:        minDepth,
		KCount:          kCount,
		KLen:            kLen,
	}
	params := GetParams(allele)
	panelType := params["panel_type"]
	if panelType == "" {
		panelType = "presence"
	}
	name := params["name"]
	version := params["version"]
	if version == "" {
		version = "1"
	}
	if panelType != "variant" && panelType != "presence" {
		length, _ := strconv.Atoi(params["length"])
		if set.Groups[panelType] == nil {
			set.Groups[panelType] = map[string]*TaxonCoverage{}
		}
		if set.Groups[panelType][name] == nil {
			set.Groups[panelType][name] = &TaxonCoverage{}
		}
		group := set.Groups[panelType][name]
		group.TotalBases += length
		if percentCoverage > 75 && medianDepth > 0 {
			group.PercentCoverage = append(group.PercentCoverage, percentCoverage)
			group.Length = append(group.Length, length)
			group.Median = append(group.Median, medianDepth)
		}
		return
	}
	if set.Presence[name] == nil {
		set.Presence[name] = map[string]SequenceProbeCoverage{}
	}
	set.Presence[name][version] = SequenceProbeCoverage{
		Name:                     name,
		ProbeCoverage:            pc,
		PercentCoverageThreshold: 30,
		Version:                  version,
		Length:                   params["length"],
	}
}

func parseVariantRow(set *CoverageSet, row []string, probe string, medianDepth, minDepth, percentCoverage float64, kCount, kLen int) {
	params := GetParams(probe)
	probeType := strings.Split(probe, "-")[0]
	varName := ""
	if v, ok := params["var_name"]; ok {
		varName = params["gene"] + "_" + params["mut"] + "-" + v
	} else {
		parts := strings.Split(strings.Split(probe, "?")[0], "-")
		if len(parts) > 1 {
			varName = parts[1]
		}
	}
	if set.Variant[varName] == nil {
		set.Variant[varName] = &VariantProbeCoverage{
			ReferenceCoverages: []ProbeCoverage{},
			AlternateCoverages: []ProbeCoverage{},
			VarName:            probe,
			Params:             params,
		}
	}
	pc := ProbeCoverage{
		PercentCoverage: percentCoverage,
		MedianDepth:     medianDepth,
		MinDepth:        minDepth,
		KCount:          kCount,
		KLen:            kLen,
	}
	switch probeType {
	case "ref":
		set.Variant[varName].ReferenceCoverages = append(set.Variant[varName].ReferenceCoverages, pc)
		set.Variant[varName].ChooseBestReferenceCoverage()
	case "alt":
		set.Variant[varName].AlternateCoverages = append(set.Variant[varName].AlternateCoverages, pc)
		set.Variant[varName].ChooseBestAlternateCoverage()
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
