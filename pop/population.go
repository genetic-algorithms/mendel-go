package pop

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"log"
	"sort"
	"math/rand"
	"fmt"
)

type RecombinationType uint8
const (
	CLONAL RecombinationType = 1
	//SUPPRESSED RecombinationType = 2   <-- have not needed these yet, uncomment when we do
	//FULL_SEXUAL RecombinationType = 3
)

// Population tracks the tribes and global info about the population. It also handles population-wide actions
// like mating and selection.
type Population struct {
	//todo: we currently don't track males vs. females. Eventually we should: assign each offspring a sex, maintaining a prescribed
	//		sex ratio for the population. Only allow opposite sex individuals to mate.
	Indivs []*Individual

	Size int 		// the specified size of this population. 0 if no specific size.
	Num_offspring float64 		// calculated from config values Fraction_random_death and Reproductive_rate
}


// PopulationFactory creates a new population (either the initial pop, or the next generation).
func PopulationFactory(initialSize int) *Population {
	p := &Population{
		Indivs: make([]*Individual, 0, initialSize), 	// allocate the array for the ptrs to the indivs. The actual indiv objects will be appended either in Initialize or as the population grows during mating
		Size: config.Cfg.Basic.Pop_size,
	}

	fertility_factor := 1. - config.Cfg.Selection.Fraction_random_death
	p.Num_offspring = 2.0 * config.Cfg.Basic.Reproductive_rate * fertility_factor 	// the default for Num_offspring is 4

	return p
}


// Initialize creates individuals (with no mutations) for the 1st generation
func (p *Population) Initialize() {
	//todo: there is probably a faster way to initialize these arrays
	for i:=1; i<=p.Size; i++ { p.Append(IndividualFactory(p)) }
}


// Size returns the current number of individuals in this population
func (p *Population) GetCurrentSize() int { return len(p.Indivs) }


// Append adds a person to this population
func (p *Population) Append(indivs ...*Individual) {
	p.Indivs = append(p.Indivs, indivs ...)
}


// Mate mates all the pairs of the population, combining their linkage blocks randomly and returns the new/resulting population.
// The mating process is:
// - randomly choose 2 parents
// - determine number of offspring
// - for each offspring:
//   - for each LB position, choose 1 LB from dad and 1 from mom
//   - add new mutations to random LBs
//   - add offspring to new population
func (p *Population) Mate(uniformRandom *rand.Rand) *Population {
	config.Verbose(4, "Mating the population of %d individuals...\n", p.GetCurrentSize())

	// Create the next generation population object that we will fill in as a result of mating. It is ok if we underestimate the
	// size a little, because we will add individuals with p.Append() anyway.
	newGenerationSize := int((float64(p.GetCurrentSize()) / 2) * p.Num_offspring)
	newP := PopulationFactory(newGenerationSize)

	// To prepare for mating, create a shuffled slice of indices into the parent population
	parentIndices := uniformRandom.Perm(p.GetCurrentSize())
	config.Verbose(9, "parentIndices: %v\n", parentIndices)

	// Mate pairs and create the offspring. Now that we have shuffled the parent indices, we can just go 2 at a time thru the indices.
	for i:=0; i< p.GetCurrentSize(); i=i+2 {
		dadI := parentIndices[i]
		momI := parentIndices[i+1]
		newIndivs := p.Indivs[dadI].Mate(p.Indivs[momI], uniformRandom)
		newP.Append(newIndivs...)
	}

	return newP
}


// Select removes the least fit individuals in the population
func (p *Population) Select() {
	config.Verbose(4, "Select: eliminating %d individuals to maintain a population of %d...\n", p.GetCurrentSize()-p.Size, p.Size)

	// Sort the indexes of the Indivs array by fitness, and mark the least fit individuals as dead
	indexes := p.sortIndexByFitness()
	numEliminate := len(indexes) - p.Size		//todo: handle the case in which more are already dead due to deleterious mutations than you want to select out
	for i:=0; i<numEliminate; i++ { p.Indivs[indexes[i].Index].Dead = true } 	// sorted by fitness in ascending order, so mark the 1st ones dead

	// Compact the Indivs array by moving the live individuals to the 1st p.Size elements
	nextIndex := 0
	for i:=0; i<len(p.Indivs); i++ {
		if !p.Indivs[i].Dead {
			if i > nextIndex {
				// copy it into the next open spot
				p.Indivs[nextIndex] = p.Indivs[i] 		// I think a shallow copy is ok, we only copy the pointers to the LB arrays
			}
			nextIndex++
		}
	}

	p.Indivs = p.Indivs[0:nextIndex] 		// readjust the slice to be only the live individuals

	return
}


// GetAverageFitness returns the average of all the individuals fitness levels
func (p *Population) GetAverageFitness() (fitness float32) {
	for _, ind := range p.Indivs { fitness += ind.Fitness }
	fitness = fitness / float32(p.GetCurrentSize())
	return
}


// GetAverageNumMutations returns the average number of deleterious, neutral, favorable mutations in the individuals
func (p *Population) GetAverageNumMutations() (deleterious, neutral, favorable float64) {
	var delet, neut, fav int
	for _, ind := range p.Indivs {
		d, n, f := ind.GetNumMutations()
		delet += d
		neut += n
		fav += f
	}
	size := float64(p.GetCurrentSize())
	deleterious = float64(delet) / size
	neutral = float64(neut) / size
	favorable = float64(fav) / size
	return
}


// Report prints out statistics of this population. If final==true is prints more details.
func (p *Population) Report(genNum int, final bool) {
	//todo: for reporting we go thru all individuals and LB multiple times. Make this more efficient
	perGenVerboseLevel := 3			// level at which we will print population level info
	perGenIndivVerboseLevel := 7	// level at which we will print individual level info
	popSize := p.GetCurrentSize()

	if final {
		if !config.IsVerbose(perGenVerboseLevel) {
			log.Println("Final report:")
			log.Printf("Population size: %v, Individuals' average fitness: %v", p.GetCurrentSize(), p.GetAverageFitness())
			d, n, f := p.GetAverageNumMutations()
			log.Printf(" Individuals' average number of deleterious mutations: %v, neutral mutations: %v, favorable mutations: %v", d, n, f)
		}
		if !config.IsVerbose(perGenIndivVerboseLevel) && config.IsVerbose(4) {
			log.Println(" Individual Detail:")
			for _, i := range p.Indivs { i.Report(final) }
		}

	} else {
		// Not final
		var d, n, f float64 	// if we get these values once, hold on to them
		var fitness float32
		if config.IsVerbose(perGenVerboseLevel) {
			fitness = p.GetAverageFitness()
			log.Printf("Generation: %d, Population size: %v, Individuals' average fitness: %v", genNum, popSize, fitness)
			d, n, f = p.GetAverageNumMutations()
			log.Printf(" Individuals' average number of deleterious mutations: %v, neutral mutations: %v, favorable mutations: %v", d, n, f)
		}
		if config.IsVerbose(perGenIndivVerboseLevel) {
			log.Println(" Individual Detail:")
			for _, ind := range p.Indivs { ind.Report(final) }
		}

		//if histWriter := config.FMgr.GetFileWriter(config.HISTORY_FILENAME); histWriter != nil {
		if histWriter := config.FMgr.GetFile(config.HISTORY_FILENAME); histWriter != nil {
			config.Verbose(5, "Writing to file %v", config.HISTORY_FILENAME)
			if d==0.0 && n==0.0 && f==0.0 { d, n, f = p.GetAverageNumMutations() }
			if fitness == 0.0 { fitness = p.GetAverageFitness() }
			fmt.Fprintf(histWriter, "Generation: %d, Population size: %d, Average mutations: deleterious: %v, neutral: %v, favorable %v, Average fitness: %v\n", genNum, popSize, d, n, f, fitness)
			//histWriter.Flush()
		}
	}
}


// Used as the elements to be sorted for selection
type IndivFit struct {
	Index int
	Fitness float32
}
type ByFitness []IndivFit
func (a ByFitness) Len() int           { return len(a) }
func (a ByFitness) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByFitness) Less(i, j int) bool { return a[i].Fitness < a[j].Fitness }


// sortIndexByFitness sorts the indexes of the individuals according to fitness (in ascending order)
func (p *Population) sortIndexByFitness() []IndivFit {
	// Initialize the index array
	indexes := make([]IndivFit, p.GetCurrentSize())
	for i := range indexes {
		indexes[i].Index = i
		indexes[i].Fitness = p.Indivs[i].Fitness
	}

	sort.Sort(ByFitness(indexes)) 		// sort the indexes according to fitness

	// Output the fitnesses to check them
	if config.IsVerbose(9) {
		fitSlice := make([]float32, len(indexes)) 	// create an array of the sorted individual fitness values so we can print them compactly
		for i,ind := range indexes { fitSlice[i] = p.Indivs[ind.Index].Fitness }
		config.Verbose(9, "fitSlice: %v", fitSlice)
	}

	return indexes
}
