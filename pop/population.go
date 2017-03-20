package pop

import (
	"bitbucket.org/geneticentropy/mendel-go/utils"
	"bitbucket.org/geneticentropy/mendel-go/config"
	"bitbucket.org/geneticentropy/mendel-go/random"
	"math/rand"
	"log"
)

type RecombinationType uint8
const (
	CLONAL RecombinationType = 1
	SUPPRESSED RecombinationType = 2
	FULL_SEXUAL RecombinationType = 3
)

// Population tracks the tribes and global info about the population. It also handles population-wide actions
// like mating and selection.
type Population struct {
	Indivs []*Individual 	//todo: do we need to track males vs. females?
	Num_offspring float64
}

// PopulationFactory creates a new population (either the initial pop, or the next generation).
func PopulationFactory(size int) *Population {
	//if size == 0 { size = config.Cfg.Basic.Pop_size }
	p := &Population{
		Indivs: make([]*Individual, size),  //todo: there are other growth models that make the pop size bigger
	}

	fertility_factor := 1. - config.Cfg.Selection.Fraction_random_death
	p.Num_offspring    = 2.0 * config.Cfg.Basic.Reproductive_rate * fertility_factor 	// the default for Num_offspring is 4

	// This initialization will only happen for the 1st population/generation
	//todo: there is probably a faster way to initilize these arrays
	for i := range p.Indivs { p.Indivs[i] = IndividualFactory(p) }

	return p
}

// Size returns the current number of individuals in this population
func (p *Population) Size() int { return len(p.Indivs) }

// Append adds a person to this population
func (p *Population) Append(indivs []*Individual) {
	p.Indivs = append(p.Indivs, indivs ...)
}

//todo: get this moved into the random pkg and made public
type intSlice []int
func (ins intSlice) Swap(i, j int) {
	ins[i], ins[j] = ins[j], ins[i]
}
func (ins intSlice) Len() int {
	return len(ins)
}

// Mate mates all the pairs of the population, combining their linkage blocks randomly and returns the new/resulting population.
func (p *Population) Mate() *Population {
	utils.Verbose(9, "Mating the population of %d individuals...\n", p.Size)

	// Create the next generation population that we will fill in as a result of mating
	newGenerationSize := (p.Size / 2) * p.Num_offspring
	newP := PopulationFactory(newGenerationSize)

	// To prepare for mating, create a slice of indices into the parent population and shuffle them
	parentIndices := make(intSlice, p.Size())
	for i := range parentIndices { parentIndices[i] = i }
	random.Shuffle(rand.New(random.GetSeed()), parentIndices)

	// Mate pairs and create the offspring. Now that we have shuffled the parent indices, we can just go 2 at a time.
	for i:=0; i< p.Size; i=i+2 {
		dadI := parentIndices[i]
		momI := parentIndices[i+1]
		newIndivs := p.Indivs[dadI].Mate(p.Indivs[momI], i)
		newP.Append(newIndivs)
	}

	return newP
}

// Select removes the least fit individuals in the population
func (p *Population) Select() *Population {
	utils.Verbose(9, "Selecting individuals to maintain a population of %d...\n", p.Size)
	return p    //todo:
}

// Report prints out statistics of this population
func (p *Population) Report() {
	log.Printf("Population size: %v", p.Size())
}
