package random


import (
	crand "crypto/rand"
	"math"
	"math/big"
	"math/rand"
)

var NextSeed int64 // is initialized to config.Cfg.Computation.Random_number_seed to avoid circular imports

/*
type Rnd struct {
	Rnd *rand.Rand
	Seed int64		// the seed this random number generator was created with
}
*/

// RandFactory returns a newly created random number generator with a new seed.
// Note: this is *not* thread safe, we assume you call this before starting the threads to give each its own RNG
func RandFactory() *rand.Rand {
	if NextSeed != 0 {
		r := rand.New(rand.NewSource(NextSeed))
		NextSeed++
		return r
	} else {
		return rand.New(rand.NewSource(GetSeed()))
	}
}

//func (r *Rnd) Float64() float64 { return r.Rnd.Float64() }
//func (r *Rnd) Intn(n int) int   { return r.Rnd.Intn(n) }


// Round randomly rounds an int either up or down, weighting the odds according to how far away from the integer it is.
// If the float is a perfect int, it always chooses that.
func Round(uniformRandom *rand.Rand, num float64) int {
	intNum := int(num)
	wholeNum := float64(intNum)
	if num == wholeNum { return intNum }
	if uniformRandom.Float64() > num - wholeNum {
		return intNum
	} else {
		return intNum + 1
	}
}


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


/* math.rand.Perm() is used instead of this...
type Shuffleable interface {
	Swap(i, j uint32)
	Len() uint32
}

type Uint32Slice []uint32

func (xs Uint32Slice) Swap(i, j uint32) {
	xs[i], xs[j] = xs[j], xs[i]
}

func (xs Uint32Slice) Len() uint32 {
	return uint32(len(xs))
}

// Fisher-Yates shuffle
// (https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle#The_modern_algorithm)
func Shuffle(uniformRandom *rand.Rand, xs Shuffleable) {
	for i := xs.Len() - 1; i > 0; i-- {
		j := uint32(uniformRandom.Intn(int(i + 1)))
		xs.Swap(i, j)
	}
}
*/
