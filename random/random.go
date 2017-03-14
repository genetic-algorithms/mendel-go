package random


import (
	crand "crypto/rand"
	"math"
	"math/big"
	"math/rand"
)


// Algorithm taken from Wikipedia
// (https://en.wikipedia.org/wiki/Poisson_distribution#Generating_Poisson-distributed_random_variables).
// This differs from a naive implementation in avoiding rounding errors for lambda > 700
func Poisson(uniformRandom *rand.Rand, lambda float64) uint32 {
	var STEP float64 = 500
	lambdaLeft := lambda
	var k uint32 = 0
	var p float64 = 1

	for {
		k += 1
		u := uniformRandom.Float64()
		p *= u

		if p < math.E && lambdaLeft > 0 {
			if lambdaLeft > STEP {
				p *= math.Exp(STEP)
				lambdaLeft -= STEP
			} else {
				p *= math.Exp(lambdaLeft)
				lambdaLeft = -1
			}

		}

		if p <= 1 { break }
	}

	return k - 1
}

// Get a random int64 from /dev/urandom to use as a seed
func GetSeed() int64 {
	nBig, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))

	if err != nil {
		panic(err)
	}

	return nBig.Int64()
}
