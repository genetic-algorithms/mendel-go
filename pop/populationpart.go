package pop

import (
	"math/rand"
	"sync"
	"github.com/genetic-algorithms/mendel-go/utils"
	"log"
)

// PopulationPart is a construct used to partition the population for the purpose mating parts of the population
// in different go routines in a thread-safe way. Each go routine is assigned 1 instance of PopulationPart and
// only writes to that object. (We could instead use mutexes to coordinate writing to shared objects, but that
// reduces performance, and in the mating operation performance is key because it is the most time consuming operation.
type PopulationPart struct {
	Indivs         []*Individual    // the offspring of this part of the population
	NextIndivIndex int              // supports reusing the Individual objects in a repurposed part
	Pop            *Population      // a reference back to the whole population, but that object should only be read
	MyUniqueInt    *utils.UniqueInt // this part gets its own range for mutation id's that can be manipulated concurrently with the gloabl one. This is set in Mate().

									// Note: fitness stats are saved at the Population level, not at the part level...
}


// PopulationPartFactory returns an instance of PopulationPart
func PopulationPartFactory(numIndivs uint32, pop *Population) *PopulationPart {
	p := &PopulationPart{Pop: pop}

	if numIndivs > 0 {
		p.Indivs = make([]*Individual, 0, numIndivs)
		for i:=uint32(1); i<= numIndivs; i++ { p.Indivs = append(p.Indivs, IndividualFactory(p, true)) }
	}

	return p
}


// Reinitialize repurposes a part for another generation. This is never called for gen 0.
func (p *PopulationPart) Reinitialize() {
	// This part already has an array of Individual objects which we want to reuse. Pool.GetIndividual will enlarge the arry as necessary
	//todo: support pop growth by extending the Indivs array if necessary
	p.NextIndivIndex = 0
}


// FreeIndivs drops references to the Indivs
func (p *PopulationPart) FreeIndivs() { p.Indivs = []*Individual{} }


// Size returns the current number of individuals in this part of the population
func (p *PopulationPart) GetCurrentSize() uint32 { return uint32(len(p.Indivs)) }


// Mate mates the parents passed in (which is a slice of the individuals in the parent population) and adds the children
// to this PopulationPart object. This function is called in a go routine so it must be thread-safe.
// Note: since parentIndices is a slice (not the actual array), passing it as a param does not copy all of the elements, which is good.
func (p *PopulationPart) Mate(parentPop *Population, parentIndices []int, uniqueInt *utils.UniqueInt, uniformRandom *rand.Rand, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	p.MyUniqueInt = uniqueInt 		// hold this for use by all of the objects i contain
	if len(parentIndices) == 0 { return }
	// Note: the caller already shuffled the parents

	//PopPool.SetEstimatedNumIndivs(p, uint32(float64(len(parentIndices)) * p.Pop.Num_offspring))
	p.SetEstimatedNumIndivs(uint32(float64(len(parentIndices)) * p.Pop.Num_offspring))

	// Mate pairs and create the offspring. Now that we have shuffled the parent indices, we can just go 2 at a time thru the indices.
	for i := 0; i < len(parentIndices) - 1; i += 2 {
		dadI := parentIndices[i]
		momI := parentIndices[i+1]
		// dadI and momI are just indices into the combined Indivs array in the Population object, so we index into that.
		// Each PopulationPart has a distinct subset of indices, so this is thread-safe.
		/*newChildren :=*/ parentPop.IndivRefs[dadI].Indiv.Mate(parentPop.IndivRefs[momI].Indiv, p, uniformRandom)
		//p.Append(newChildren...) 		// <- Mate() already adds the Individuals to this part

		parentPop.FreeParentRefs(dadI, momI)
		/*
		if !config.Cfg.Computation.Reuse_populations {
			//parentPop.IndivRefs[dadI].Indiv.Free()	// <- this doesn't help any more than setting the Indiv ptr to nil
			parentPop.IndivRefs[dadI].Indiv = nil
			//parentPop.IndivRefs[momI].Indiv.Free()
			parentPop.IndivRefs[momI].Indiv = nil
		}
		*/
	}
}


// SetEstimatedNumIndivs allocates the array of ptrs to Individuals once to an approx size, instead of appending multiple times.
// Note: this has to use the member vars of the part (instead of having our own) so it is thread safe.
func (p *PopulationPart) SetEstimatedNumIndivs(estimatedNumIndivs uint32) {
	if cap(p.Indivs) == 0 {
		// This is a brand new part
		p.Indivs = make([]*Individual, 0, estimatedNumIndivs)    // It is ok if we underestimate the size a little, because we will add individuals with append() anyway.
		//} else if cap(part.Indivs) < estimatedNumIndivs {
		//	// This is a repurposed part, but the Indivs array is not big enough. Let append() enlarge it.
		//	part.Indivs = make([]*Individual, len(part.Indivs), estimatedNumIndivs) 	// setting len to the original array so we can reuse the Individual objects
	}
	// else the Indivs array is approx big enough and we will reuse the Individual objects it points to and enlarge as necessary
	//todo: in the pop growth case we could do a better job of estimating the size we need.
}


// GetIndividual returns the next available Individual to reuse, or creates one if necessary.
// Note: this has to use the member vars of the part (instead of having our own) so it is thread safe.
func (p *PopulationPart) GetIndividual() (ind *Individual) {
	// should we look at cap() instead of len() and test if the ptr to the Individual is nil? Not sure if we ever shrink this slice
	if p.NextIndivIndex < len(p.Indivs) {
		if p.Indivs[p.NextIndivIndex] == nil { log.Fatalf("Error: part.Indivs[%v] is nil even though index is < length %v", p.NextIndivIndex, len(p.Indivs))}
		// We are still within the allocated array and there is an existing Individual object we can reuse
		//config.Verbose(1, " reusing individual at index %d", part.NextIndivIndex)
		//ind = p.Indivs[p.NextIndivIndex].Reinitialize()
	} else {
		// The Indivs slice is too small. Create an indiv and append it.
		//config.Verbose(1, " creating new individual at index %d", part.NextIndivIndex)
		ind = IndividualFactory(p, false)
		p.Indivs = append(p.Indivs, ind)
	}

	p.NextIndivIndex++
	return
}


/*
// GetNextIndiv returns a pointer to the next Individual object that can be used, either a reused one, or one
// created and added to the array.
func (p *PopulationPart) GetNextIndiv() (ind *Individual) {
	//todo: should we look at cap() instead of len() and test if the ptr to the Individual is nil? Don't know if we ever shrink this slice
	if p.NextIndivIndex < len(p.Indivs) && p.Indivs[p.NextIndivIndex] != nil {
		// We are still within the allocated array and there is an existing Individual object we can reuse
		ind = p.Indivs[p.NextIndivIndex].Reinitialize()
	} else {
		// Either the array is too small, or there is no Individual object there. Either way, create and append it.
		ind = IndividualFactory(p, false)
		p.Indivs = append(p.Indivs, ind)
	}

	p.NextIndivIndex++
	return
}


// Append adds children to this population part. This is our function (instead of using append() directly), in case in
// the future we want to allocate additional individuals in bigger chunks for efficiency. See https://blog.golang.org/go-slices-usage-and-internals
func (p *PopulationPart) Append(indivs ...*Individual) {
	// Note: the initial make of the Children array is approximately big enough to avoid append having to copy the array in most cases
	p.Indivs = append(p.Indivs, indivs ...)
}
*/
