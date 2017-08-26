package pop

import (
	"math/rand"
	"sync"
)

// PopulationPart is a construct used to partition the population for the purpose mating parts of the population
// in different go routines in a thread-safe way. Each go routine is assigned 1 instance of PopulationPart and
// only writes to that object. (We could instead use mutexes to coordinate writing to shared objects, but that
// reduces performance, and in the mating operation performance is key because it is the most time consuming operation.
type PopulationPart struct {
	Indivs                    []*Individual // the offspring of this part of the population
	Pop                       *Population   // a reference back to the whole population, but that object should only be read

	/* These are saved at the Population level, not at the part level...
	ActualAvgOffspring        float64       // The average number of offspring each parent actually had in this partition
	PreSelGenoFitnessMean     float64       // The average fitness of all of the children in this partition (before selection) due to their genomic mutations
	PreSelGenoFitnessVariance float64       //
	PreSelGenoFitnessStDev    float64       // The standard deviation from the GenoFitnessMean
	*/
}


// PopulationPartFactory returns an instance of PopulationPart
func PopulationPartFactory(initialSize uint32, pop *Population) *PopulationPart {
	p := &PopulationPart{Pop: pop}

	if initialSize > 0 {
		p.Indivs = make([]*Individual, 0, initialSize)
		for i:=uint32(1); i<= initialSize; i++ { p.Append(IndividualFactory(p.Pop, true)) }
	}

	return p
}


// FreeIndivs drops references to the Indivs
func (p *PopulationPart) FreeIndivs() { p.Indivs = []*Individual{} }


// Size returns the current number of individuals in this part of the population
func (p *PopulationPart) GetCurrentSize() uint32 { return uint32(len(p.Indivs)) }


// Mate mates the parents passed in (which is a slice of the individuals in the population) and collects the children
// in this PopulationPart object. This function is called in a go routine so it must be thread-safe.
// Note: since parentIndices is a slice (not the actual array), passing it as a param does not copy all of the elements.
func (p *PopulationPart) Mate(parentPop *Population, parentIndices []int, uniformRandom *rand.Rand, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	if len(parentIndices) == 0 { return }
	// Note: the caller already shuffled the parents

	estimatedNumChildren := uint32(float64(len(parentIndices)) * p.Pop.Num_offspring)
	p.Indivs = make([]*Individual, 0, estimatedNumChildren)	// It is ok if we underestimate the size a little, because we will add individuals with p.Append() anyway.

	// Mate pairs and create the offspring. Now that we have shuffled the parent indices, we can just go 2 at a time thru the indices.
	for i := 0; i < len(parentIndices) - 1; i += 2 {
		dadI := parentIndices[i]
		momI := parentIndices[i+1]
		// dadI and momI are just indices into the combined Indivs array in the Population object, so we index into that.
		// Each PopulationPart has a distinct subset of indices, so this is thread-safe.
		newChildren := parentPop.IndivRefs[dadI].Indiv.Mate(parentPop.IndivRefs[momI].Indiv, uniformRandom)
		p.Append(newChildren...)

		// Eliminate the reference to these parents so gc can reclaim them because we don't need them any more
		// Note: the 2 lines to nil out the indiv reference seem to reduce memory usage, but i'm not sure why because the PopulationPart of the parent population still has
		//		a ptr to the individual, and we have no way of finding that index (w/o storing more info).
		//parentPop.IndivRefs[dadI].Indiv.Free()	// <- this doesn't help any more than setting the Indiv ptr to nil
		parentPop.IndivRefs[dadI].Indiv = nil
		//parentPop.IndivRefs[momI].Indiv.Free()
		parentPop.IndivRefs[momI].Indiv = nil
	}
}


// Append adds a person to this population part. This is our function (instead of using append() directly), in case in
// the future we want to allocate additional individuals in bigger chunks for efficiency. See https://blog.golang.org/go-slices-usage-and-internals
func (p *PopulationPart) Append(indivs ...*Individual) {
	// Note: the initial make of the Children array is approximately big enough to avoid append having to copy the array in most cases
	p.Indivs = append(p.Indivs, indivs ...)
}
