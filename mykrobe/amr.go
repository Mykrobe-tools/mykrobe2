package mykrobe

import (
	"math"
	"math/rand"
	"sort"
)

const (
	ONTErrorRate = 0.08
	ONTPloidy    = "haploid"
)

type ConfThresholder struct {
	errorRate            float64
	meanDepth            float64
	kmerLength           int
	incorrectKmerToPCCov map[int]float64
	iterations           int
	logConfAndCoverage   [][2]float64
	random               *rand.Rand
}

func NewConfThresholder(errorRate, meanDepth float64, kmerLength int, incorrectKmerToPCCov map[int]float64, iterations int) *ConfThresholder {
	if iterations <= 0 {
		iterations = 10000
	}
	return &ConfThresholder{
		errorRate:            errorRate,
		meanDepth:            meanDepth,
		kmerLength:           kmerLength,
		incorrectKmerToPCCov: incorrectKmerToPCCov,
		iterations:           iterations,
		random:               rand.New(rand.NewSource(42)),
	}
}

func EstimateKmerCountErrorRateAndIncorrectKmerPercentCov(variantCalls map[string]Call, fallbackErrorRate float64) (float64, map[int]float64) {
	correctKmerCount := 0.0
	incorrectKmerCount := 0.0
	incorrectKmerToPCCov := map[int][]float64{}

	for _, variantCall := range variantCalls {
		covAny, ok := variantCall.Info["coverage"]
		if !ok {
			continue
		}
		covDict, ok := covAny.(map[string]any)
		if !ok {
			continue
		}
		refCov, okRef := covDict["reference"].(map[string]any)
		altCov, okAlt := covDict["alternate"].(map[string]any)
		if !okRef || !okAlt {
			continue
		}

		var incorrectKmer int
		var pcCov float64
		switch {
		case len(variantCall.Genotype) == 2 && variantCall.Genotype[0] == 0 && variantCall.Genotype[1] == 0:
			correctKmerCount += asFloat(refCov["kmer_count"])
			incorrectKmer = int(math.Round(asFloat(altCov["kmer_count"])))
			pcCov = asFloat(altCov["percent_coverage"])
		case len(variantCall.Genotype) == 2 && variantCall.Genotype[0] == 1 && variantCall.Genotype[1] == 1:
			correctKmerCount += asFloat(altCov["kmer_count"])
			incorrectKmer = int(math.Round(asFloat(refCov["kmer_count"])))
			pcCov = asFloat(refCov["percent_coverage"])
		default:
			continue
		}

		incorrectKmerCount += float64(incorrectKmer)
		incorrectKmerToPCCov[incorrectKmer] = append(incorrectKmerToPCCov[incorrectKmer], pcCov)
	}

	out := map[int]float64{0: 0}
	for incorrectKmer, covList := range incorrectKmerToPCCov {
		total := 0.0
		for _, cov := range covList {
			total += cov
		}
		out[incorrectKmer] = total / float64(len(covList))
	}

	if incorrectKmerCount+correctKmerCount == 0 {
		return fallbackErrorRate, out
	}
	return incorrectKmerCount / (incorrectKmerCount + correctKmerCount), out
}

func ApplyONTDefaults(errorRate float64, ploidy string, ont bool) (float64, string) {
	if ont {
		return ONTErrorRate, ONTPloidy
	}
	return errorRate, ploidy
}

func GuessSequenceMethod(errorRate float64, ploidy string, enabled bool, kmerCountErrorRate float64) (float64, string, bool) {
	if enabled && kmerCountErrorRate > 0.001 {
		return ONTErrorRate, ONTPloidy, true
	}
	return errorRate, ploidy, false
}

func (c *ConfThresholder) GetConfThreshold(percentToKeep float64) int {
	if len(c.logConfAndCoverage) == 0 {
		c.simulateSNPs()
	}
	if len(c.logConfAndCoverage) == 0 {
		return 0
	}
	idx := int(0.01 * percentToKeep * float64(len(c.logConfAndCoverage)))
	if idx >= len(c.logConfAndCoverage) {
		idx = len(c.logConfAndCoverage) - 1
	}
	return int(c.logConfAndCoverage[idx][0])
}

func (c *ConfThresholder) simulateSNPs() {
	vtyper := NewVariantTyper([]float64{c.meanDepth}, nil, c.errorRate, DefaultMinorFreq, false, nil, 0, "kmer_count", c.kmerLength, 0.3, "diploid")
	for i := 0; i < c.iterations; i++ {
		correctCovg := poisson(c.random, c.meanDepth)
		incorrectCovg := binomial(c.random, int(math.Round(c.meanDepth)), c.errorRate)
		if correctCovg+incorrectCovg == 0 {
			continue
		}

		correctKCount := float64(c.kmerLength*correctCovg) + 0.01
		incorrectKCount := float64(c.kmerLength*incorrectCovg) + 0.01
		vpc := &VariantProbeCoverage{
			ReferenceCoverages: []ProbeCoverage{{
				PercentCoverage: 100,
				MedianDepth:     c.meanDepth,
				MinDepth:        1,
				KCount:          int(math.Round(correctKCount)),
				KLen:            c.kmerLength,
			}},
			AlternateCoverages: []ProbeCoverage{{
				PercentCoverage: c.getIncorrectKmerPercentCov(int(math.Round(incorrectKCount))),
				MedianDepth:     c.meanDepth,
				MinDepth:        1,
				KCount:          int(math.Round(incorrectKCount)),
				KLen:            c.kmerLength,
			}},
		}
		call := vtyper.Type(vpc)
		cov := math.Log10(float64(correctCovg + incorrectCovg))
		conf := float64(call.Info["conf"].(int))
		c.logConfAndCoverage = append(c.logConfAndCoverage, [2]float64{conf, cov})
	}
	sort.Slice(c.logConfAndCoverage, func(i, j int) bool {
		if c.logConfAndCoverage[i][0] == c.logConfAndCoverage[j][0] {
			return c.logConfAndCoverage[i][1] > c.logConfAndCoverage[j][1]
		}
		return c.logConfAndCoverage[i][0] > c.logConfAndCoverage[j][0]
	})
}

func (c *ConfThresholder) getIncorrectKmerPercentCov(kCount int) float64 {
	for i := kCount; i >= 0; i-- {
		if cov, ok := c.incorrectKmerToPCCov[i]; ok {
			return cov
		}
	}
	return 0
}

func poisson(r *rand.Rand, lambda float64) int {
	if lambda <= 0 {
		return 0
	}
	l := math.Exp(-lambda)
	k := 0
	p := 1.0
	for p > l {
		k++
		p *= r.Float64()
	}
	return k - 1
}

func binomial(r *rand.Rand, n int, p float64) int {
	if n <= 0 || p <= 0 {
		return 0
	}
	if p >= 1 {
		return n
	}
	out := 0
	for i := 0; i < n; i++ {
		if r.Float64() < p {
			out++
		}
	}
	return out
}
