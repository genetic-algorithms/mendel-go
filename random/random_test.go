package random


import (
	"math"
	"math/rand"
	"testing"
)


func TestPoisson(t *testing.T) {
	kCounts := make([]uint32, 100)
	var iterations uint32 = 10E3
	var epsilon float64 = 0.006
	uniform_random := rand.New(rand.NewSource(1))
	var lambda float64 = 20

	for i := uint32(0); i < iterations; i += 1 {
		k := f(uniform_random, lambda)
		kCounts[k] += 1
	}

	for k, kCount := range kCounts {
		expectedProb := poissonProbability(lambda, uint32(k))
		actualProb := float64(kCount) / float64(iterations)

		delta := math.Abs(expectedProb - actualProb)
		if delta > epsilon {
			t.Error("For k =", k, " and lambda =", lambda, "expected probability", expectedProb, "and actual probability", actualProb, "differ by", delta, "which is more than tolerance", epsilon)
		}
	}
}

// Taken from Wikipedia
// (https://en.wikipedia.org/wiki/Poisson_distribution#Definition)
func poissonProbability(lambda float64, k uint32) float64 {
	g, _ := math.Lgamma(float64(k + 1))
	return math.Exp(float64(k) * math.Log(lambda) - lambda - g)
}
