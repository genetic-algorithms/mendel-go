package pop

import (
	"math/rand"
)

// PopulationPart is a construct used to partition the population for the purpose mating parts of the population
// in different go routines in a thread-safe way. Each go routine is assigned 1 instance of PopulationPart and
// only writes to that object. (We could instead use mutexes to coordinate writing to shared objects, but that
// reduces performance, and in the mating operation performance is key because it is the most time consuming operation.
type PopulationPart struct {
	Children []*Individual	// the offspring of this part of the population
	Pop *Population		// a reference back to the whole population, but that object should only be read

	ActualAvgOffspring        float64 // The average number of offspring each parent actually had in this partition
	PreSelGenoFitnessMean     float64 // The average fitness of all of the children in this partition (before selection) due to their genomic mutations
	PreSelGenoFitnessVariance float64 //
	PreSelGenoFitnessStDev    float64 // The standard deviation from the GenoFitnessMean
}


// PopulationPartFactory returns an instance of PopulationPart
func PopulationPartFactory(estimatedSize uint32, pop *Population) *PopulationPart {
	p := &PopulationPart{
		Children: make([]*Individual, 0, estimatedSize),	// It is ok if we underestimate the size a little, because we will add individuals with p.Append() anyway.
		Pop: pop,
	}

	return p
}


// Append adds a person to this population part. This is our function (instead of using append() directly), in case in
// the future we want to allocate additional individuals in bigger chunks for efficiency. See https://blog.golang.org/go-slices-usage-and-internals
func (p *PopulationPart) Append(indivs ...*Individual) {
	// Note: the initial make of the Children array is approximately big enough avoid append having to copy the array in most cases
	p.Children = append(p.Children, indivs ...)
}


// Mate mates the parents passed in (which is a slice of the individuals in the population) and collects the children
// in this PopulationPart object. This function is called in a go routine so it must be thread-safe.
//todo: pass the parents w/o copying the whole slice
func (p *PopulationPart) Mate(parentIndices []int, uniformRandom *rand.Rand) {
	if len(parentIndices) == 0 { return }
	// Note: the caller already shuffled the parents

	// Mate pairs and create the offspring. Now that we have shuffled the parent indices, we can just go 2 at a time thru the indices.
	for i := 0; i < len(parentIndices) - 1; i += 2 {
		dadI := parentIndices[i]
		momI := parentIndices[i+1]
		// dadI and momI are just indices into the combined Indivs array in the Population object, so we index into that.
		// Each PopulationPart has a distinct subset of indices, so this is thread-safe.
		newChildren := p.Pop.indivs[dadI].Mate(p.Pop.indivs[momI], uniformRandom)
		p.Append(newChildren...)
	}
}