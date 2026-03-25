package mykrobe

import "math"

const (
	MinLLK           = -99999999.0
	DefaultErrorRate = 0.05
	DefaultMinorFreq = 0.2
	DefaultKmerSize  = 31
)

func PercentCoverageFromExpectedCoverage(coverage float64) float64 {
	return 1 - math.Exp(-coverage)
}

func LogLikProbabilityOfNGaps(depth, percentCoverage float64, length int) float64 {
	pc := percentCoverage / 100.0
	nGaps := int(math.Round(float64(length) - (float64(length) * pc)))
	expectedNGaps := math.Exp(-depth) * float64(length)
	if expectedNGaps <= 0 {
		expectedNGaps = 1e-308
	}
	return LogPoissonProb(expectedNGaps, float64(nGaps))
}

func LogPoissonProb(lambda, k float64) float64 {
	return -lambda + k*math.Log(lambda) - logFactorial(int(k))
}

func logFactorial(n int) float64 {
	if n < 0 {
		panic("negative factorial")
	}
	out := 0.0
	for i := 1; i <= n; i++ {
		out += math.Log(float64(i))
	}
	return out
}

func LogLikDepth(depth, expectedDepth float64) float64 {
	if expectedDepth <= 0 {
		panic("expected depth must be > 0")
	}
	if depth < 0 {
		panic("depth must be >= 0")
	}
	return LogPoissonProb(expectedDepth, depth)
}

func LogLikRScoverage(observedAlt, observedRef, expectedAlt, expectedRef float64) float64 {
	return LogPoissonProb(expectedAlt, observedAlt) + LogPoissonProb(expectedRef, observedRef)
}

func LogLikRSkmerCount(observedRef, observedAlt, expectedRef, expectedAlt float64) float64 {
	return LogPoissonProb(expectedRef, observedRef) + LogPoissonProb(expectedAlt, observedAlt)
}
