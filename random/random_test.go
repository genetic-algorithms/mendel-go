package random


import (
	"math"
	"math/rand"
	"testing"
)


// Runs many iterations of generating Poisson random numbers and makes sure
// the distribution matches the probability given by poissonProbability.
// As the number of iterations increases our accuracy will increase and therefore
// we could decrease epsilon. But we have kept the number of iterations reasonably
// low to ensure the tests run quickly.
func TestPoisson(t *testing.T) {
	kCounts := make([]uint32, 100)
	var iterations uint32 = 10E3
	var epsilon float64 = 0.006
	uniformRandom := rand.New(rand.NewSource(1))
	var lambda float64 = 20

	for i := uint32(0); i < iterations; i++ {
		k := Poisson(uniformRandom, lambda)
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

type intSlice []int

func (self intSlice) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self intSlice) Len() int {
	return len(self)
}

// Runs many iterations of shuffling a `Slice` of `int`s and computes the
// average value found at each index. This average value should match the
// average of all the values in the slice. As the number of iterations
// increases our accuracy will increase and therefore we could decrease epsilon.
// But we have kept the number of iterations reasonably low to ensure the
// tests run quickly.
func TestShuffle(t *testing.T) {
	counts := make([]int, 9)
	var iterations int = 10E3
	var epsilon float64 = 0.06
	var expectedValue float64 = 4
	uniformRandom := rand.New(rand.NewSource(1))
	xs := intSlice{0, 1, 2, 3, 4, 5, 6, 7, 8}

	for i := 0; i < iterations; i++ {
		Shuffle(uniformRandom, xs)

		for j := 0; j < 9; j++ {
			counts[j] += xs[j]
		}
	}

	for i, count := range counts {
		actualValue := float64(count) / float64(iterations)

		delta := math.Abs(expectedValue - actualValue)
		if delta > epsilon {
			t.Error("For i =", i, " and actual value", actualValue, "and expected value", expectedValue, "differ by", delta, "which is more than tolerance", epsilon)
		}
	}
}
