package pop

/* Not currently used, because it is only slightly faster and uses more memory. Kept here for reference...
import (
	"github.com/genetic-algorithms/mendel-go/config"
	"log"
)

Pool is an abstraction on top of object allocation so we can reuse previous populations cleanly. When a new object is
requested, it will return a cleaned up existing one if possible, otherwise it creates a new one.
Note that this abstraction is not used for the genesis population, because that has different requirements.
The object hierarchy of a population is (note the difference between objects and ptrs to objects):

Population
  []*PopulationPart
    []*Individual
      ChromosomesFromDad []*dna.Chromosome
      ChromosomesFromMom []*dna.Chromosome
        []LinkageBlock
          []Mutation
type Pool struct {
	oldPop       *Population // we only support reusing 1 population (and all the objects within it) at a time
	reuse        bool        // whether or not we are supposed to reuse old populations
	oldPopInited bool        // remember if we reinitialized it or not
	parentPop    *Population
	childrenPop  *Population // either this or oldPop are set, not both
}

// Pool is a singleton instance of Pool
var PopPool *Pool


// PoolFactory creates and initializes the global instance of Pool
func PoolFactory() {
	PopPool = &Pool{reuse: config.Cfg.Computation.Reuse_populations} 		// at the beginning we don't have a population to reuse
}


// RecylePopulation holds on to a pop that is done being the parent pop. We will reuse this pop when NextGeneration is called.
func (p *Pool) RecyclePopulation(pop *Population) {
	//config.Verbose(1, " in RecyclePopulation")
	if !p.reuse { return }		// do not store this pop, so we will create a new one when asked for a pop
	//config.Verbose(1, " storing old pop for reuse")
	p.oldPop = pop
	p.oldPopInited = false
}


// NextGeneration returns a pop that can be used for the children of this gen. Either a recycled gen or a new one.
func (p *Pool) GetNextGeneration(parentPop *Population, genNum uint32) *Population {
	p.parentPop = parentPop 		// we need to know the parent pop in some of the methods below
	if p.oldPop != nil {		// oldPop will never get set if reuse is false
		p.childrenPop = nil
		if !p.oldPopInited {
			// This also reinitializes the pop parts. (The individuals, etc. will be reinitialized when asked for.)
			p.oldPop.Reinitialize(parentPop, genNum)
			p.oldPopInited = true
		}
		//config.Verbose(1, " returning recycled pop")
		return p.oldPop
	} else {
		p.oldPop = nil
		p.oldPopInited = false
		p.childrenPop = PopulationFactory(parentPop, genNum)	// this creates the PopulationParts too
		//config.Verbose(1, " returning new pop")
		return p.childrenPop
	}
}


// getChildrenPop returns which pop we are currently using for the children.
func (p *Pool) getChildrenPop() *Population {
	if p.oldPop != nil {
		if p.childrenPop != nil { log.Fatalln("Error: in Pool both oldPop and childrenPop are not nil") }
		return p.oldPop
	} else {
		if p.childrenPop == nil { log.Fatalln("Error: in Pool both oldPop and childrenPop are nil") }
		return p.childrenPop
	}
}

// Note: the PopulationPart objects are created when the Population is created and reinitialized when the Population
// is reinitialized, so we don't need a Get method for that.


// SetEstimatedNumIndivs allocates the array of ptrs to Individuals once to an approx size, instead of appending multiple times.
// Note: this has to use the member vars of the part (instead of having our own) so it is thread safe.
func (p *Pool) SetEstimatedNumIndivs(part *PopulationPart, estimatedNumIndivs uint32) {
	if cap(part.Indivs) == 0 {
		// This is a brand new part
		part.Indivs = make([]*Individual, 0, estimatedNumIndivs)    // It is ok if we underestimate the size a little, because we will add individuals with append() anyway.
	//} else if cap(part.Indivs) < estimatedNumIndivs {
	//	// This is a repurposed part, but the Indivs array is not big enough. Let append() enlarge it.
	//	part.Indivs = make([]*Individual, len(part.Indivs), estimatedNumIndivs) 	// setting len to the original array so we can reuse the Individual objects
	}
	// else the Indivs array is approx big enough and we will reuse the Individual objects it points to and enlarge as necessary
	// in the pop growth case we could do a better job of estimating the size we need.
}


// GetIndividual returns the next available Individual to reuse, or creates one if necessary.
// Note: this has to use the member vars of the part (instead of having our own) so it is thread safe.
func (p *Pool) GetIndividual(part *PopulationPart) (ind *Individual) {
	// should we look at cap() instead of len() and test if the ptr to the Individual is nil? Not sure if we ever shrink this slice
	if part.NextIndivIndex < len(part.Indivs) {
		if part.Indivs[part.NextIndivIndex] == nil { log.Fatalf("Error: part.Indivs[%v] is nil even though index is < length %v", part.NextIndivIndex, len(part.Indivs))}
		// We are still within the allocated array and there is an existing Individual object we can reuse
		//config.Verbose(1, " reusing individual at index %d", part.NextIndivIndex)
		ind = part.Indivs[part.NextIndivIndex].Reinitialize()
	} else {
		// The Indivs slice is too small. Create an indiv and append it.
		//config.Verbose(1, " creating new individual at index %d", part.NextIndivIndex)
		ind = IndividualFactory(part, false)
		part.Indivs = append(part.Indivs, ind)
	}

	part.NextIndivIndex++
	return
}


// FreeParentRefs eliminates the reference to these 2 parents so gc can reclaim them because we don't need them any more
func (p *Pool) FreeParentRefs(dadIndex int, momIndex int) {
	if p.reuse { return }
	// Note: the 2 lines to nil out the indiv reference seem to reduce memory usage, but i'm not sure why because the PopulationPart of the parent population still has
	//		a ptr to the individual, and we have no way of finding that index (w/o storing more info).
	//p.parentPop.IndivRefs[dadI].Indiv.Free()	// <- this doesn't help any more than setting the Indiv ptr to nil
	p.parentPop.IndivRefs[dadIndex].Indiv = nil
	//p.parentPop.IndivRefs[momI].Indiv.Free()
	p.parentPop.IndivRefs[momIndex].Indiv = nil
}


// FreeChildrenPtrs should be called after IndivRefs points to all of the individuals. This gets rid of the parts references to them,
// so GC can free individuals as soon as IndivRefs goes away.
func (p *Pool) FreeChildrenPtrs(lastGen bool) {
	if p.reuse && !lastGen { return }
	for _, part := range p.getChildrenPop().Parts {
		part.FreeIndivs()
	}
}


// FreeBeforeAlleleCount frees everything possible for GC before we count alleles. We assume we won't be called after this
func (p *Pool) FreeBeforeAlleleCount() {
	p.FreeChildrenPtrs(true)
	p.oldPop = nil
	p.oldPopInited = false
	p.parentPop = nil
	p.childrenPop = nil
}
*/
