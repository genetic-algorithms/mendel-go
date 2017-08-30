package pop

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"log"
	"sort"
	"math/rand"
	"fmt"
	"math"
	"bitbucket.org/geneticentropy/mendel-go/utils"
	"bitbucket.org/geneticentropy/mendel-go/dna"
	"bitbucket.org/geneticentropy/mendel-go/random"
	"sync"
	"os"
	"strconv"
	"runtime"
)

type RecombinationType uint8
const (
	//CLONAL RecombinationType = 1   <-- have not needed these yet, uncomment when we do
	//SUPPRESSED RecombinationType = 2
	FULL_SEXUAL RecombinationType = 3
)


// Used as the elements for the Sort routine used for selection, and as indirection to point to individuals in PopulationPart objects
type IndivRef struct {
	//Index uint32
	//Fitness float64
	Indiv *Individual
}
type ByFitness []IndivRef
func (a ByFitness) Len() int           { return len(a) }
func (a ByFitness) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
//func (a ByFitness) Less(i, j int) bool { return a[i].Fitness < a[j].Fitness }
func (a ByFitness) Less(i, j int) bool { return a[i].Indiv.PhenoFitness < a[j].Indiv.PhenoFitness }


// Population tracks the tribes and global info about the population. It also handles population-wide actions
// like mating and selection.
type Population struct {
	//indivs []*Individual 	// delete - The backing array for IndexRefs. This ends up being sparse after selection, with many individuals marked dead. Note: we currently don't track males vs. females.
	Parts []*PopulationPart
	IndivRefs []IndivRef	// References to individuals in the indivs array. This level of indirection allows us to sort this list, truncate it after selection, and refer to indivs in PopulationParts, all w/o copying Individual objects.

	TargetSize uint32        // the target size of this population after selection
	Num_offspring float64       // Average number of offspring each individual should have (so need to multiple by 2 to get it for the mating pair). Calculated from config values Fraction_random_death and Reproductive_rate.
	ActualAvgOffspring float64       // The average number of offspring each individual from last generation actually had in this generation
	PreSelGenoFitnessMean float64                                       // The average fitness of all of the individuals (before selection) due to their genomic mutations
	PreSelGenoFitnessVariance float64                                   //
	PreSelGenoFitnessStDev    float64                                   // The standard deviation from the GenoFitnessMean
	EnvironNoise              float64                                   // randomness applied to geno fitness calculated from PreSelGenoFitnessVariance, heritability, and non_scaling_noise
	LBsPerChromosome uint32                                             // How many linkage blocks in each chromosome. For now the total number of LBs must be an exact multiple of the number of chromosomes

	MeanFitness, MinFitness, MaxFitness float64                         // cache summary info about the individuals
	TotalNumMutations uint32
	MeanNumMutations float64

	MeanNumDeleterious, MeanNumNeutral, MeanNumFavorable  float64       // cache some of the stats we usually gather
	MeanDelFit, MeanFavFit                                float64

	MeanNumDelAllele, MeanNumNeutAllele, MeanNumFavAllele float64       // cache some of the stats we usually gather
	MeanDelAlleleFit, MeanFavAlleleFit                    float64
}


// PopulationFactory creates a new population (either the initial pop, or the next generation).
//func PopulationFactory(initialSize uint32) *Population {
func PopulationFactory(prevPop *Population, genNum uint32) *Population {
	var targetSize uint32
	if prevPop != nil {
		targetSize = Mdl.PopulationGrowth(prevPop, genNum)
	} else {
		// This is the 1st generation, so set the size from the config param
		targetSize = config.Cfg.Basic.Pop_size
	}
	p := &Population{
		//indivs: make([]*Individual, 0, initialSize), 	// allocate the array for the ptrs to the indivs. The actual indiv objects will be appended either in Initialize or as the population grows during mating
		Parts: make([]*PopulationPart, 0, config.Cfg.Computation.Num_threads), 	// allocate the array for the ptrs to the parts. The actual part objects will be appended either in Initialize or as the population grows during mating
		TargetSize: targetSize,
	}

	fertility_factor := 1. - config.Cfg.Selection.Fraction_random_death
	//p.Num_offspring = 2.0 * config.Cfg.Population.Reproductive_rate * fertility_factor 	// the default for Num_offspring is 4
	p.Num_offspring = config.Cfg.Population.Reproductive_rate * fertility_factor 	// the default for Num_offspring is 2

	p.LBsPerChromosome = uint32(config.Cfg.Population.Num_linkage_subunits / config.Cfg.Population.Haploid_chromosome_number)	// main.initialize() already confirmed it was a clean multiple

	if genNum == 0 {
		// Create individuals (with no mutations) for the 1st generation. (For subsequent generations, individuals are added to the Population object via Mate().
		//for i:=uint32(1); i<= initialSize; i++ { p.Append(IndividualFactory(p, true)) }
		p.Parts = append(p.Parts, PopulationPartFactory(targetSize, p))	// for gen 0 we only need 1 part because that doesn't have offspring added to it during Mate()
		p.makeAndFillIndivRefs()
	} else {
		for i:=uint32(1); i<= config.Cfg.Computation.Num_threads; i++ { p.Parts = append(p.Parts, PopulationPartFactory(0, p)) }
		// Mate() will populate PopulationPart with Individuals and run makeAndFillIndivRefs()
	}

	return p
}


// Size returns the current number of individuals in this population
func (p *Population) GetCurrentSize() uint32 {
	//return uint32(len(p.indivs))
	config.Verbose(9, "Population.GetCurrentSize(): %v", len(p.IndivRefs))
	return uint32(len(p.IndivRefs)) }

/*
// Append adds a person to this population. This is our function (instead of using append() directly), in case in
// the future we want to allocate additional individuals in bigger chunks for efficiency. See https://blog.golang.org/go-slices-usage-and-internals
func (p *Population) Append(indivs ...*Individual) {
	// Note: the initial make of the Indivs array is approximately big enough avoid append having to copy the array in most cases
	p.indivs = append(p.indivs, indivs ...)
}
*/


// GenerateInitialAlleles creates the initial contrasting allele pairs (if specified by the config file) and adds them to the population
func (p *Population) GenerateInitialAlleles(uniformRandom *rand.Rand) {
	if config.Cfg.Population.Num_contrasting_alleles == 0 || config.Cfg.Population.Initial_alleles_pop_frac <= 0.0 { return }	// nothing to do
	defer utils.Measure.Start("GenerateInitialAlleles").Stop("GenerateInitialAlleles")

	// Loop thru individuals, and skipping or choosing individuals to maintain a ratio close to Initial_alleles_pop_frac
	// Note: config.Validate() already confirms Initial_alleles_pop_frac is > 0 and <= 1.0
	var numWithAlleles uint32 = 0 		// so we can calc the ratio so far of indivs we've given alleles to vs. number of indivs we've processed
	var numLBsWithAlleles uint32
	var numProcessedLBs uint32
	//for i, ind := range p.indivs {
	for i, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		var ratioSoFar float64
		if i > 0 { ratioSoFar = float64(numWithAlleles) / float64(i) }
		// else ratioSoFar = 0
		if ratioSoFar <= config.Cfg.Population.Initial_alleles_pop_frac {
			// Give this indiv alleles to boost the ratio closer to Initial_alleles_pop_frac
			config.Verbose(9, "Giving initial contrasting allele to individual %v", i)
			numLBsWithAlleles, numProcessedLBs = ind.AddInitialContrastingAlleles(config.Cfg.Population.Num_contrasting_alleles, uniformRandom)
			numWithAlleles++
		}
		// else we don't give this indiv alleles to bring the ratio down closer to Initial_alleles_pop_frac
	}

	config.Verbose(2, "Initial alleles given to faction %v of individuals (%v/%v). Each individual got alleles on fraction %v of LBs (%v/%v)", float64(numWithAlleles)/float64(p.GetCurrentSize()), numWithAlleles, p.GetCurrentSize(),float64(numLBsWithAlleles)/float64(numProcessedLBs), numLBsWithAlleles, numProcessedLBs)
}


// Mate mates all the pairs of the population, choosing the linkage block at each linkage block position randomly from
// the mom or dad (as in meiosis), and then returns the new/resulting population.
// The mating process is:
// - randomly choose 2 parents
// - determine number of offspring
// - for each offspring:
//   - for each LB position, choose 1 LB from dad (from either his dad or mom) and 1 LB from mom (from either her dad or mom)
//   - add new mutations to random LBs
//   - add offspring to new population
//func (p *Population) Mate(uniformRandom *rand.Rand) *Population {
func (p *Population) Mate(newP *Population, uniformRandom *rand.Rand) {
	defer utils.Measure.Start("Mate").Stop("Mate")
	config.Verbose(4, "Mating the population of %d individuals...\n", p.GetCurrentSize())

	// Create the next generation population object that we will fill in as a result of mating.
	//newP := PopulationFactory(0)

	// To prepare for mating, create a shuffled slice of indices into the parent population
	parentIndices := uniformRandom.Perm(int(p.GetCurrentSize()))
	//config.Verbose(9, "parentIndices: %v\n", parentIndices)

	//newP.Part.Mate(p, parentIndices, uniformRandom)
	// Divide parentIndices into segments (whose size is an even number) and schedule a go routine to mate each segment
	// Note: runtime.GOMAXPROCS(runtime.NumCPU()) is the default, but this statement can be modified to set a different number of CPUs to use
	segmentSize := utils.RoundToEven( float64(len(parentIndices)) / float64(len(newP.Parts)) )
	if segmentSize <= 0 { segmentSize = 2 }
	segmentStart := 0
	highestIndex := len(parentIndices) - 1
	config.Verbose(4, "Scheduling %v population parts concurrently with segmentSize=%v, highestIndex=%v", len(newP.Parts), segmentSize, highestIndex)
	var waitGroup sync.WaitGroup
	for i := range newP.Parts {
		newPart := newP.Parts[i] 		// part can't be declared in the for stmt because it would change before some of the go routines start. See https://github.com/golang/go/wiki/CommonMistakes
		if segmentStart <= highestIndex {
			// We still have more elements in parentIndices to mate
			var newRandom *rand.Rand
			if i == 0 {
				// Let the 1st thread use the main uniformRandom generator. This has the effect that if there is only 1 thread, we will have the same
				// sequence of random numbers that we had before concurrency was added (so we can verify the results).
				newRandom = uniformRandom
			} else if config.Cfg.Computation.Random_number_seed != 0 {
				newRandom = rand.New(rand.NewSource(config.Cfg.Computation.Random_number_seed+int64(i)))
			} else {
				newRandom = rand.New(rand.NewSource(random.GetSeed()))
			}

			beginIndex := segmentStart		// just to be careful, copying segmentStart to a local var so it is different from each go routine invocation
			var endIndex int
			if i < len(newP.Parts) - 1 {
				endIndex = utils.MinInt(segmentStart + segmentSize - 1, highestIndex)
			} else {
				// the last partition, so do everything that is left
				endIndex = highestIndex
			}
			waitGroup.Add(1)
			go newPart.Mate(p, parentIndices[beginIndex:endIndex +1], newRandom, &waitGroup)
			segmentStart = endIndex + 1
		}
		// else we are out of elements in parentIndices so do not do anything
	}

	// Wait for all of the Mate functions to complete
	waitGroup.Wait()

	/* this was before concurrency...
	// Mate pairs and create the offspring. Now that we have shuffled the parent indices, we can just go 2 at a time thru the indices.
	for i := uint32(0); i < p.GetCurrentSize() - 1; i += 2 {
		dadI := parentIndices[i]
		momI := parentIndices[i+1]
		//newIndivs := p.indivs[dadI].Mate(p.indivs[momI], uniformRandom)
		newIndivs := p.IndivRefs[dadI].Indiv.Mate(p.IndivRefs[momI].Indiv, uniformRandom)
		newP.Append(newIndivs...)
	}
	*/
	newP.makeAndFillIndivRefs()	// now that we are done creating new individuals, fill in the array of references to them
	for _, part := range newP.Parts { part.FreeIndivs() }	// now that we point to all of the individuals in IndivRefs, get rid of the parts references to them, so GC can free individuals as soon as IndivRefs goes away

	// Save off the average num offspring for stats, before we select out individuals
	newP.ActualAvgOffspring = float64(newP.GetCurrentSize()) / float64(p.GetCurrentSize())

	newP.PreSelGenoFitnessMean, newP.PreSelGenoFitnessVariance, newP.PreSelGenoFitnessStDev = newP.CalcFitnessStats()

	//return newP
}


// Select removes the least fit individuals in the population
func (p *Population) Select(uniformRandom *rand.Rand) {
	defer utils.Measure.Start("Select").Stop("Select")
	config.Verbose(4, "Select: eliminating %d individuals to try to maintain a population of %d...\n", p.GetCurrentSize()-p.TargetSize, p.TargetSize)

	// Calculate noise factor to get pheno fitness of each individual
	herit := config.Cfg.Selection.Heritability
	p.EnvironNoise = math.Sqrt(p.PreSelGenoFitnessVariance * (1.0-herit) / herit + math.Pow(config.Cfg.Selection.Non_scaling_noise,2))
	Mdl.ApplySelectionNoise(p, p.EnvironNoise, uniformRandom) 		// this sets PhenoFitness in each of the individuals

	// Sort the indexes of the Indivs array by fitness, and mark the least fit individuals as dead
	p.sortIndexByPhenoFitness()		// this sorts p.IndivRefs
	//numDead := p.numAlreadyDead(p.IndivRefs)
	numAlreadyDead := p.getNumDead()

	if numAlreadyDead > 0 {
		config.Verbose(3, "%d individuals died (fitness < 0, or < 1 when using spps) as a result of mutations added during mating", numAlreadyDead)
	}

	currentSize := uint32(len(p.IndivRefs))

	if currentSize > p.TargetSize {
		numEliminate := currentSize - p.TargetSize

		if numAlreadyDead < numEliminate {
			// Mark those that should be eliminated dead. They are sorted by fitness in ascending order, so mark the 1st ones dead.
			for i := uint32(0); i < numEliminate; i++ {
				//p.Indivs[indexes[i].Index].Dead = true
				p.IndivRefs[i].Indiv.Dead = true
			}
		}
	}
	numDead := p.getNumDead()		// under certain circumstances this could be > the number we wanted to select out
	p.ReportDeadStats()
	p.IndivRefs = p.IndivRefs[numDead:]		// re-slice IndivRefs to eliminate the dead individuals
	for i, indRef := range p.IndivRefs { if indRef.Indiv.Dead { log.Fatalf("System Error: individual IndivRefs[%v] with pheno-fitness %v and geno-fitness %v is dead but still in IndivRefs.", i, indRef.Indiv.PhenoFitness, indRef.Indiv.GenoFitness) } }	//todo: comment out

	/* We can leave the indivs array sparse (with dead individuals in it), because the IndivRefs array only points to live entries in indivs.
	// Compact the Indivs array by moving the live individuals to the 1st p.Size elements. Accumulate stats on the dead along the way.
	nextIndex := 0
	for i := 0; i < len(p.indivs); i++ {
		ind := p.indivs[i]
		if !ind.Dead {
			if i > nextIndex {
				// copy it into the next open spot
				p.indivs[nextIndex] = ind 		// I think a shallow copy is ok, we only copy the pointers to the LB arrays
			}
			// else there are no open slots yet, because we have not encountered dead ones yet
			nextIndex++
		}
	}
	p.indivs = p.indivs[0:nextIndex] 		// readjust the slice to be only the live individuals
	*/

	return
}


// getNumDead returns the current number of dead individuals in this population
func (p *Population) getNumDead() uint32 {
	// We assume at this point p.IndivRefs is sorted by ascending fitness, so only need to count until we hit a non-dead individual
	for i, indRef := range p.IndivRefs {
		if !indRef.Indiv.Dead { return uint32(i) }
	}
	return uint32(len(p.IndivRefs))
}


// ApplySelectionNoiseType functions add environmental noise and selection noise to the GenoFitness to set the PhenoFitness of all of the individuals of the population
type ApplySelectionNoiseType func(p *Population, envNoise float64, uniformRandom *rand.Rand)

// ApplyTruncationNoise only adds environmental noise (no selection noise)
func ApplyFullTruncationNoise(p *Population, envNoise float64, uniformRandom *rand.Rand) {
	/*
	Full truncation only adds a small amount of randomness, envNoise, which is calculated the 2 input parameters heritability and non_scaling_noise.
	Then the individuals are ordered by the resulting PhenoFitness, and the least fit are eliminated to achieve the desired population size.
	This makes full truncation the most efficient selection model, and unrealistic unless envNoise is set rather high.
	*/
	//for _, ind := range p.indivs {
	for _, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		if ind.Dead {
			ind.PhenoFitness = 0.0
		} else {
			ind.PhenoFitness = ind.GenoFitness + uniformRandom.Float64() * envNoise
		}
	}
}

// ApplyUnrestrictProbNoise adds environmental noise and unrestricted probability selection noise
func ApplyUnrestrictProbNoise(p *Population, envNoise float64, uniformRandom *rand.Rand) {
	/*
	For unrestricted probability selection (UPS), divide the phenotypic fitness by a uniformly distributed random number prior to
	ranking and truncation.  This procedure allows the probability of surviving and reproducing in the next generation to be
	related to phenotypic fitness and also for the correct number of individuals to be eliminated to maintain a constant
	population size.
	*/

	/* Apply the environmental noise in a separate loop if you want the random num sequence to match that of spps (in which case the 2
		models give exactly the same results if reproductive_rate is not very small.
	for _, ind := range p.Indivs {
		ind.PhenoFitness = ind.GenoFitness + (uniformRandom.Float64() * envNoise)
	}
	*/

	//for _, ind := range p.indivs {
	for _, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		if ind.Dead {
			ind.PhenoFitness = 0.0
		} else {
			//rnd1 := uniformRandom.Float64()
			ind.PhenoFitness = ind.GenoFitness + (uniformRandom.Float64() * envNoise)
			//rnd2 := uniformRandom.Float64()
			ind.PhenoFitness = ind.PhenoFitness / (uniformRandom.Float64() + 1.0e-15)
		}
		//config.Verbose(9, " Individual %v: %v + (%v * %v) = %v, UnrestrictProbNoise: %v / (%v + 1.0e-15) = %v", i, ind.GenoFitness, rnd1, envNoise, fit, fit, rnd2, ind.PhenoFitness)
		//ind.PhenoFitness = fit * uniformRandom.Float64()  // this has also been suggested instead of the line above, the results are pretty similar
	}
}

// ApplyProportProbNoise adds environmental noise and strict proportionality probability selection noise
func ApplyProportProbNoise(p *Population, envNoise float64, uniformRandom *rand.Rand) {
	/*
	For strict proportionality probability selection (SPPS), rescale (normalize) the phenotypic fitness values such that the maximum value is one.
	Then divide the scaled phenotypic fitness by a uniformly distributed random number prior to ranking and truncation.
	Allow only those individuals to reproduce whose resulting ratio of scaled phenotypic fitness to the random number value
	exceeds one.  This approach ensures that no individual automatically survives to reproduce regardless of their GenoFitness.
	But it restricts the percentage of the offspring that can survive to approximately 70-80% (it depends on the spread of the fitness).
	Therefore, when the reproductive_rate is low (approx < 1.4), the number of surviving offspring may not be large enough to sustain a constant population size.
	 */
	//config.Verbose(2, "Using ApplyProportProbNoise...")

	// First find max individual fitness (after applying the environmental noise)
	var maxFitness float64 = 0.0
	//for _, ind := range p.indivs {
	for _, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		if ind.Dead {
			ind.PhenoFitness = 0.0
		} else {
			ind.PhenoFitness = ind.GenoFitness + (uniformRandom.Float64() * envNoise)
		}
		maxFitness = utils.MaxFloat64(maxFitness, ind.PhenoFitness)
	}
	// Verify maxFitness is not zero so we can divide by it below
	if maxFitness <= 0.0 { log.Fatalf("Max individual fitness is < 0 (%v), so whole population must be dead. Exiting.", maxFitness) }

	// Normalize the pheno fitness
	//for _, ind := range p.indivs {
	for _, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		// The 1st division below produces values in the range (minFitness/maxFitness) - 1. The 2nd division gets the ratio of the
		// result and a random num 0 - 1. Since minFitness-maxFitness tends to be small (typically about 0.1), on average most results
		// will be > than the random num, therefore the ratio of most will be > 1.0, so few will get marked dead.
		ind.PhenoFitness = ind.PhenoFitness / maxFitness / (uniformRandom.Float64() + 1.0e-15)
		if ind.PhenoFitness < 1.0 { ind.Dead = true }
	}
}

// ApplyPartialTruncationNoise adds environmental noise and partial truncation selection noise
func ApplyPartialTruncationNoise(p *Population, envNoise float64, uniformRandom *rand.Rand) {
	/*
	For partial truncation selection, divide the phenotypic fitness by theta + ((1. - theta) * uniformRandom)
	prior to ranking and truncation, where theta is the parameter
	partial_truncation_value.  This selection scheme is intermediate between full truncation selection and unrestricted
	probability selection.  The procedure allows for the correct number of individuals to be eliminated to maintain a constant
	population size.
	*/
	//for _, ind := range p.indivs {
	for _, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		if ind.Dead {
			ind.PhenoFitness = 0.0
		} else {
			ind.PhenoFitness = ind.GenoFitness + (uniformRandom.Float64() * envNoise)
			ind.PhenoFitness = ind.PhenoFitness / (config.Cfg.Selection.Partial_truncation_value + ((1. - config.Cfg.Selection.Partial_truncation_value) * uniformRandom.Float64()))
		}
	}
}


// PopulationGrowthType takes in the current population and generation number and returns the target pop size for the next gen
type PopulationGrowthType func(prevPop *Population, genNum uint32) uint32

// NoPopulationGrowth returns the same pop size as the previous generation
func NoPopulationGrowth(prevPop *Population, _ uint32) uint32 {
	return prevPop.TargetSize
}

// ExponentialPopulationGrowth returns the previous pop size times the growth rate
func ExponentialPopulationGrowth(prevPop *Population, _ uint32) uint32 {
	return uint32(math.Ceil(config.Cfg.Population.Pop_growth_rate * float64(prevPop.TargetSize)))
}

// CapacityPopulationGrowth uses an equation in which the pop size approaches the carrying capacity
func CapacityPopulationGrowth(prevPop *Population, _ uint32) uint32 {
	// mendel-f90 calculates the new pop target size as ceiling(pop_size * (1. + pop_growth_rate * (1. - pop_size/carrying_capacity) ) )
	newTargetSize := uint32(math.Ceil( float64(prevPop.TargetSize) * (1.0 + config.Cfg.Population.Pop_growth_rate * (1.0 - float64(prevPop.TargetSize)/float64(config.Cfg.Population.Carrying_capacity)) ) ))
	return newTargetSize
}

// FoundersPopulationGrowth increases the pop size exponentially until it reaches the carrying capacity
func FoundersPopulationGrowth(prevPop *Population, genNum uint32) uint32 {
	var newTargetSize uint32
	if config.Cfg.Population.Bottleneck_generation == 0 || genNum < config.Cfg.Population.Bottleneck_generation {
		// We are before the bottleneck so use 1st growth rate
		newTargetSize = uint32(math.Ceil(config.Cfg.Population.Pop_growth_rate * float64(prevPop.TargetSize)))
	} else if genNum >= config.Cfg.Population.Bottleneck_generation && genNum < config.Cfg.Population.Bottleneck_generation + config.Cfg.Population.Num_bottleneck_generations {
		// We are in the bottleneck range
		newTargetSize = config.Cfg.Population.Bottleneck_pop_size
	} else {
		// We are after the bottleneck so use 2nd growth rate
		newTargetSize = uint32(math.Ceil(config.Cfg.Population.Pop_growth_rate2 * float64(prevPop.TargetSize)))
	}
	newTargetSize = utils.MinUint32(newTargetSize, config.Cfg.Population.Carrying_capacity) 	// do not want it exceeding the carrying capacity
	return newTargetSize
}

// CalcFitnessStats returns the mean geno fitness and std deviation
func (p *Population) CalcFitnessStats() (genoFitnessMean, genoFitnessVariance, genoFitnessStDev float64) {
	// Calc mean (average)
	//for _, ind := range p.indivs {
	for _, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		genoFitnessMean += ind.GenoFitness
	}
	genoFitnessMean = genoFitnessMean / float64(p.GetCurrentSize())

	//for _, ind := range p.indivs {
	for _, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		genoFitnessVariance += math.Pow(ind.GenoFitness-genoFitnessMean, 2)
	}
	genoFitnessVariance = genoFitnessVariance / float64(p.GetCurrentSize())
	genoFitnessStDev = math.Sqrt(genoFitnessVariance)
	return
}


// ReportDeadStats reports means of all the individuals that are being eliminated by selection
func (p *Population) ReportDeadStats() {
	elimVerboseLevel := uint32(4)            // level at which we will collect and print stats about dead/eliminated individuals
	if !config.IsVerbose(elimVerboseLevel) { return }
	var avgDel, avgNeut, avgFav, avgDelFit, avgFavFit, avgFitness, minFitness, maxFitness float64 	// these are stats for dead/eliminated individuals
	minFitness = 99.0
	maxFitness = -99.0
	var numDead, numDel, numNeut, numFav uint32
	//for _, ind := range p.indivs {
	for _, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		//todo: stop when i hit the 1st non-dead individual
		if ind.Dead {
			// This is a dead individual. Capture some stats before we overwrite it.
			numDead++
			avgFitness += ind.GenoFitness
			if ind.GenoFitness > maxFitness {
				maxFitness = ind.GenoFitness
			}
			if ind.GenoFitness < minFitness {
				minFitness = ind.GenoFitness
			}
			d, n, f, avD, avF := ind.GetMutationStats()
			numDel += d
			numNeut += n
			numFav += f
			avgDelFit += float64(d) * avD
			avgFavFit += float64(f) * avF
		}
	}

	// Calculate and print the elimination stats
	avgFitness = avgFitness / float64(numDead)
	if numDead > 0 {
		avgDel = float64(numDel) / float64(numDead)
		avgNeut = float64(numNeut) / float64(numDead)
		avgFav = float64(numFav) / float64(numDead)
	}
	if numDel > 0 { avgDelFit = avgDelFit / float64(numDel) }
	if numFav > 0 { avgFavFit = avgFavFit / float64(numFav) }
	config.Verbose(elimVerboseLevel, "Avgs of the %d indivs eliminated: avg fitness: %v, min fitness: %v, max fitness: %v, del: %v, neut: %v, fav: %v, del fitness: %v, fav fitness: %v", numDead, avgFitness, minFitness, maxFitness, avgDel, avgNeut, avgFav, avgDelFit, avgFavFit)
}


/*
// numAlreadyDead finds how many individuals (if any) have already been marked dead due to their fitness falling below the
// allowed threshold when mutations were added during mating. They will be at the beginning.
func (p *Population) numAlreadyDead(sortedIndexes []IndivRef) (numDead uint32) {
	for _, index := range sortedIndexes {
		//if ! p.Indivs[index.Index].Dead { return } 		// since it is sorted by fitness in ascending order, once we hit a live indiv, they all will be, so we can stop counting
		if ! index.Indiv.Dead { return } 		// since it is sorted by fitness in ascending order, once we hit a live indiv, they all will be, so we can stop counting
		numDead++
	}
	return
}
*/


// GetFitnessStats returns the average of all the individuals fitness levels, as well as the min and max, and total and mean mutations.
// Note: this function should only get stats that the individuals already have, because it is called in a minimal verbose level that is meant to be fast.
func (p *Population) GetFitnessStats() (float64, float64, float64, uint32, float64) {
	// See if we already calculated and cached the values
	if p.MeanFitness > 0.0 { return p.MeanFitness, p.MinFitness, p.MaxFitness, p.TotalNumMutations, p.MeanNumMutations }
	p.MinFitness = 99.0
	p.MaxFitness = -99.0
	p.MeanFitness = 0.0
	p.TotalNumMutations = 0
	p.MeanNumMutations = 0.0
	//for _, ind := range p.indivs {
	for _, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		p.MeanFitness += ind.GenoFitness
		if ind.GenoFitness > p.MaxFitness { p.MaxFitness = ind.GenoFitness
		}
		if ind.GenoFitness < p.MinFitness { p.MinFitness = ind.GenoFitness
		}
		p.TotalNumMutations += ind.NumMutations
	}
	popSize := p.GetCurrentSize()
	p.MeanFitness = p.MeanFitness / float64(popSize)
	p.MeanNumMutations = float64(p.TotalNumMutations) / float64(popSize)
	return p.MeanFitness, p.MinFitness, p.MaxFitness, p.TotalNumMutations, p.MeanNumMutations
}


// GetMutationStats returns the average number of deleterious, neutral, favorable mutations, and the average fitness factor of each
func (p *Population) GetMutationStats() (float64, float64, float64,  float64, float64) {
	// See if we already calculated and cached the values. Note: we only check deleterious, because fav and neutral could be 0
	//config.Verbose(9, "p.MeanNumDeleterious=%v, p.MeanDelFit=%v", p.MeanNumDeleterious, p.MeanDelFit)
	if p.MeanNumDeleterious > 0 && p.MeanDelFit < 0.0 { return p.MeanNumDeleterious, p.MeanNumNeutral, p.MeanNumFavorable, p.MeanDelFit, p.MeanFavFit }
	p.MeanNumDeleterious=0.0;  p.MeanNumNeutral=0.0;  p.MeanNumFavorable=0.0;  p.MeanDelFit=0.0;  p.MeanFavFit=0.0

	// For each type of mutation, get the average fitness factor and number of mutation for every individual and combine them. Example: 20 @ .2 and 5 @ .4 = (20 * .2) + (5 * .4) / 25
	var delet, neut, fav uint32
	//for _, ind := range p.indivs {
	for _, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		d, n, f, avD, avF := ind.GetMutationStats()
		//config.Verbose(9, "pop: avD=%v, avF=%v", avD, avF)
		delet += d
		neut += n
		fav += f
		p.MeanDelFit += float64(d) * avD
		p.MeanFavFit += float64(f) * avF
	}
	size := float64(p.GetCurrentSize())
	if size > 0 {
		p.MeanNumDeleterious = float64(delet) / size
		p.MeanNumNeutral = float64(neut) / size
		p.MeanNumFavorable = float64(fav) / size
	}
	if delet > 0 { p.MeanDelFit = p.MeanDelFit / float64(delet) }
	if fav > 0 { p.MeanFavFit = p.MeanFavFit / float64(fav) }
	return p.MeanNumDeleterious, p.MeanNumNeutral, p.MeanNumFavorable, p.MeanDelFit, p.MeanFavFit
}


// GetInitialAlleleStats returns the average number of deleterious, neutral, favorable initial alleles, and the average fitness factor of each
func (p *Population) GetInitialAlleleStats() (float64, float64, float64,  float64, float64) {
	// See if we already calculated and cached the values. Note: we only check deleterious, because fav and neutral could be 0
	if p.MeanNumDelAllele > 0 && p.MeanDelAlleleFit < 0.0 { return p.MeanNumDelAllele, p.MeanNumNeutAllele, p.MeanNumFavAllele, p.MeanDelAlleleFit, p.MeanFavAlleleFit	}
	p.MeanNumDelAllele=0.0;  p.MeanNumNeutAllele=0.0;  p.MeanNumFavAllele=0.0;  p.MeanDelAlleleFit=0.0;  p.MeanFavAlleleFit=0.0

	// For each type of allele, get the average fitness factor and number of alleles for every individual and combine them. Example: 20 @ .2 and 5 @ .4 = (20 * .2) + (5 * .4) / 25
	var delet, neut, fav uint32
	//for _, ind := range p.indivs {
	for _, indRef := range p.IndivRefs {
		ind := indRef.Indiv
		d, n, f, avD, avF := ind.GetInitialAlleleStats()
		//config.Verbose(9, "pop: avD=%v, avF=%v", avD, avF)
		delet += d
		neut += n
		fav += f
		p.MeanDelAlleleFit += float64(d) * avD
		p.MeanFavAlleleFit += float64(f) * avF
	}
	size := float64(p.GetCurrentSize())
	if size > 0 {
		p.MeanNumDelAllele = float64(delet) / size
		p.MeanNumNeutAllele = float64(neut) / size
		p.MeanNumFavAllele = float64(fav) / size
	}
	if delet > 0 { p.MeanDelAlleleFit = p.MeanDelAlleleFit / float64(delet) }
	if fav > 0 { p.MeanFavAlleleFit = p.MeanFavAlleleFit / float64(fav) }
	return p.MeanNumDelAllele, p.MeanNumNeutAllele, p.MeanNumFavAllele, p.MeanDelAlleleFit, p.MeanFavAlleleFit
}


// ReportInitial prints out stuff at the beginning, usually headers for data files, or a summary of the run we are about to do
func (p *Population) ReportInitial(maxGenNum uint32) {
	config.Verbose(1, "Running with a population size of %d for %d generations with %d threads", p.GetCurrentSize(), maxGenNum, config.Cfg.Computation.Num_threads)

	if histWriter := config.FMgr.GetFile(config.HISTORY_FILENAME); histWriter != nil {
		// Write header for this file
		fmt.Fprintln(histWriter, "# Generation  Pop-size  Avg Offspring  Avg-deleterious Avg-neutral  Avg-favorable  Avg-del-fit  Avg-fav-fit  Avg-fitness  Min-fitness  Max-fitness  Total Mutns  Mean Mutns  Noise")
	}

	if fitWriter := config.FMgr.GetFile(config.FITNESS_FILENAME); fitWriter != nil {
		// Write header for this file
		fmt.Fprintln(fitWriter, "# Generation  Pop-size  Avg Offspring  Avg-fitness  Min-fitness  Max-fitness  Total Mutns  Mean Mutns  Noise")
	}

	/*
	if alleleWriter := config.FMgr.GetFile(config.ALLELES_COUNT_FILENAME); alleleWriter != nil {
		// Write the outer json object and the array that will contain the output of each generation
		if _, err := alleleWriter.Write([]byte(`{"allelesForEachGen":[`)); err != nil {
			log.Fatalf("error writing alleles to %v: %v", config.ALLELES_COUNT_FILENAME, err)
		}
	}
	*/
}


// Report prints out statistics of this population
func (p *Population) ReportEachGen(genNum uint32, lastGen bool) {
	utils.Measure.Start("ReportEachGen")
	perGenMinimalVerboseLevel := uint32(1)            // level at which we will print only the info that is very quick to gather
	perGenVerboseLevel := uint32(2)            // level at which we will print population summary info each generation
	finalVerboseLevel := uint32(1)            // level at which we will print population summary info at the end of the run
	perGenIndSumVerboseLevel := uint32(3) 		// level at which we will print individuals summary info each generation
	finalIndSumVerboseLevel := uint32(2) // Note: if you change this value, change the verbose level used to calc the values in Mate(). Level at which we will print individuals summary info at the end of the run
	perGenIndDetailVerboseLevel := uint32(7)    // level at which we will print info about each individual each generation
	finalIndDetailVerboseLevel := uint32(6)    // level at which we will print info about each individual at the end of the run
	popSize := p.GetCurrentSize()
	totalTime := utils.Measure.GetInterimTime("Total")
	genTime := utils.Measure.Stop("Generations")

	if config.IsVerbose(perGenVerboseLevel) || (lastGen && config.IsVerbose(finalVerboseLevel)) {
		aveFit, minFit, maxFit, totalMutns, meanMutns := p.GetFitnessStats()
		log.Printf("Gen: %d, Run time: %.4f, Gen time: %.4f, Pop size: %v, Indiv mean fitness: %v, min fitness: %v, max fitness: %v, total num mutations: %v, mean num mutations: %v, Mean num offspring %v, noise: %v", genNum, totalTime, genTime, popSize, aveFit, minFit, maxFit, totalMutns, meanMutns, p.ActualAvgOffspring, p.EnvironNoise)
		if config.IsVerbose(perGenIndSumVerboseLevel) || (lastGen && config.IsVerbose(finalIndSumVerboseLevel)) {
			d, n, f, avDelFit, avFavFit := p.GetMutationStats()
			log.Printf(" Indiv mutation detail means: deleterious: %v, neutral: %v, favorable: %v, del fitness: %v, fav fitness: %v, preselect fitness: %v, preselect fitness SD: %v", d, n, f, avDelFit, avFavFit, p.PreSelGenoFitnessMean, p.PreSelGenoFitnessStDev)
			ad, an, af, avDelAlFit, avFavAlFit := p.GetInitialAlleleStats()
			log.Printf(" Indiv initial allele detail means: deleterious: %v, neutral: %v, favorable: %v, del fitness: %v, fav fitness: %v", ad, an, af, avDelAlFit, avFavAlFit)
		}
	} else if config.IsVerbose(perGenMinimalVerboseLevel) {
		aveFit, minFit, maxFit, totalMutns, meanMutns := p.GetFitnessStats()		// this is much faster than p.GetMutationStats()
		log.Printf("Gen: %d, Time: %.4f, Gen time: %.4f, Pop size: %v, Indiv mean fitness: %v, min fitness: %v, max fitness: %v, total num mutations: %v, mean num mutations: %v, Mean num offspring %v", genNum, totalTime, genTime, popSize, aveFit, minFit, maxFit, totalMutns, meanMutns, p.ActualAvgOffspring)
	}
	if config.IsVerbose(perGenIndDetailVerboseLevel) || (lastGen && config.IsVerbose(finalIndDetailVerboseLevel)) {
		log.Println(" Individual Detail:")
		//for _, ind := range p.indivs { ind.Report(lastGen) }
		for _, indRef := range p.IndivRefs {
			ind := indRef.Indiv
			ind.Report(lastGen)
		}
	}

	if histWriter := config.FMgr.GetFile(config.HISTORY_FILENAME); histWriter != nil {
		config.Verbose(5, "Writing to file %v", config.HISTORY_FILENAME)
		d, n, f, avDelFit, avFavFit := p.GetMutationStats()		// GetMutationStats() caches its values so it's ok to call it multiple times
		aveFit, minFit, maxFit, totalMutns, meanMutns := p.GetFitnessStats()		// GetFitnessStats() caches its values so it's ok to call it multiple times
		// If you change this line, you must also change the header in ReportInitial()
		fmt.Fprintf(histWriter, "%d  %d  %v  %v  %v  %v  %v  %v  %v  %v  %v  %v  %v  %v\n", genNum, popSize, p.ActualAvgOffspring, d, n, f, avDelFit, avFavFit, aveFit, minFit, maxFit, totalMutns, meanMutns, p.EnvironNoise)
		//histWriter.Flush()  // <-- don't need this because we don't use a buffer for the file
		if lastGen {
			//todo: put summary stats in comments at the end of the file?
		}
	}

	if fitWriter := config.FMgr.GetFile(config.FITNESS_FILENAME); fitWriter != nil {
		config.Verbose(5, "Writing to file %v", config.FITNESS_FILENAME)
		aveFit, minFit, maxFit, totalMutns, meanMutns := p.GetFitnessStats()		// GetFitnessStats() caches its values so it's ok to call it multiple times
		// If you change this line, you must also change the header in ReportInitial()
		fmt.Fprintf(fitWriter, "%d  %d  %v  %v  %v  %v  %v  %v  %v\n", genNum, popSize, p.ActualAvgOffspring, aveFit, minFit, maxFit, totalMutns, meanMutns, p.EnvironNoise)
		//histWriter.Flush()  // <-- don't need this because we don't use a buffer for the file
		if lastGen {
			//todo: put summary stats in comments at the end of the file?
		}
	}

	// This should come last if the lastGen because we free the individuals references to make memory room for the allele count
	if config.FMgr.IsDir(config.ALLELE_BINS_DIRECTORY) && (lastGen || (config.Cfg.Computation.Plot_allele_gens > 0 && (genNum % config.Cfg.Computation.Plot_allele_gens) == 0)) {
		utils.Measure.Start("allele-count")
		// Count the alleles from all individuals. We end up with maps of mutation ids and the number of times each occurred
		//alleles := dna.AlleleCountFactory(genNum, popSize)
		alleles := dna.AlleleCountFactory() 		// as we count, the totals are gathered in this struct
		for i := range p.IndivRefs {
			ind := p.IndivRefs[i].Indiv
			ind.CountAlleles(alleles)

			// Counting the alleles takes a lot of memory when there are a lot of mutations. We are concerned that after doing the whole
			// run, we could blow the memory limit counting the alleles and lose all of the run results. So if this is the last gen
			// we don't need the individuals after we have counted them, so nil the reference to them so GC can reclaim.
			if lastGen {
				p.IndivRefs[i].Indiv = nil
				if config.Cfg.Computation.Force_gc && (i % 100) == 0 {
					utils.Measure.Start("GC")
					//config.Verbose(1, "Running GC dur allele count")
					runtime.GC()
					utils.Measure.Stop("GC")
				}
			}
		}

		// Write the plot file for each type of mutation/allele
		bucketCount := uint32(100)		// we could put this in the config file if we need to
		if alleleWriter := config.FMgr.GetDirFile(config.ALLELE_BINS_DIRECTORY, config.DELETERIOUS_CSV); alleleWriter != nil {
			fillAndOutputBuckets(alleles.Deleterious, popSize, bucketCount, alleleWriter)
		}
		// Do not even write the neutrals file if we know we don't have any
		if config.Cfg.Computation.Track_neutrals && config.Cfg.Mutations.Fraction_neutral != 0.0 {
			if alleleWriter := config.FMgr.GetDirFile(config.ALLELE_BINS_DIRECTORY, config.NEUTRAL_CSV); alleleWriter != nil {
				fillAndOutputBuckets(alleles.Neutral, popSize, bucketCount, alleleWriter)
			}
		}
		// Do not even write the favorables file if we know we don't have any
		if config.Cfg.Mutations.Frac_fav_mutn != 0.0 {
			if alleleWriter := config.FMgr.GetDirFile(config.ALLELE_BINS_DIRECTORY, config.FAVORABLE_CSV); alleleWriter != nil {
				fillAndOutputBuckets(alleles.Favorable, popSize, bucketCount, alleleWriter)
			}
		}
		// Do not even write the alleles files if we know we don't have any
		if config.Cfg.Population.Num_contrasting_alleles != 0 {
			if alleleWriter := config.FMgr.GetDirFile(config.ALLELE_BINS_DIRECTORY, config.DEL_ALLELE_CSV); alleleWriter != nil {
				fillAndOutputBuckets(alleles.DelInitialAlleles, popSize, bucketCount, alleleWriter)
			}
			if alleleWriter := config.FMgr.GetDirFile(config.ALLELE_BINS_DIRECTORY, config.FAV_ALLELE_CSV); alleleWriter != nil {
				fillAndOutputBuckets(alleles.FavInitialAlleles, popSize, bucketCount, alleleWriter)
			}
		}

		/*
		newJson, err := json.Marshal(alleles)
		if err != nil { log.Fatalf("error marshaling alleles to json: %v", err) }

		// Wrap the json in an outer json object and write it to the file
		if lastGen {
			newJson = append(newJson, "]}"...)		// no more allele outputs, so end the array and wrapping object
		} else {
			newJson = append(newJson, ","...)		// more json objects to come in the array so append a comma
		}
		if _, err := alleleWriter.Write(newJson); err != nil {
			log.Fatalf("error writing alleles to %v: %v", config.ALLELES_COUNT_FILENAME, err)
		}
		*/
		utils.Measure.Stop("allele-count")
	}

	utils.Measure.Stop("ReportEachGen")
	if lastGen {
		utils.Measure.Stop("Total")
		utils.Measure.LogSummary() 		// it checks the verbosity level itself
	}
}


func fillAndOutputBuckets(counts map[uintptr]uint32, popSize uint32, bucketCount uint32, file *os.File) {
	buckets := make([]uint32, bucketCount)

	for _, count := range counts {
		percentage := float64(count) / float64(popSize)
		var i uint32
		floati := percentage * float64(bucketCount)
		// At this point, if we just converted floati to uint32 (by truncating), bucket i would contain all float values: i <= floati < i+1
		// But we really want the buckets to contain: i < floati <= i+1
		// Remember that when we output the buckets below, we add 1 to the index of the bucket, so e.g. bucket 5 will contain: 4 < count <= 5
		// (The issue is does a count that is exactly 5 end up in buckt 5 or 6. It should go in bucket 5.)
		if math.Floor(floati) == floati {
			i = uint32(floati) - 1
		} else {
			i = uint32(floati)
		}
		// The way the calcs above are done, neither of these 2 cases should ever actually happen, but just a safeguard...
		if i < 0 {
			log.Printf("Warning: bucket index %d is out of range, putting it back in range.", i)
			i = 0
		} else if i >= bucketCount {
			log.Printf("Warning: bucket index %d is out of range, putting it back in range.", i)
			i = bucketCount - 1
		}

		buckets[i] += 1
	}

	for bucketIndex, bucketValue := range buckets {
		bucketNumberString := strconv.Itoa(bucketIndex + 1)
		bucketValueString := strconv.FormatUint(uint64(bucketValue), 10)
		file.WriteString(bucketNumberString + "\t" + bucketValueString + "\n")
	}

	return
}


// makeAndFillIndivRefs fills in the p.IndivRefs array from all of the p.Part.Indivs array
func (p *Population) makeAndFillIndivRefs() {
	// Find the total num of individuals so we can initialize the refs array
	size := 0
	for _, part := range p.Parts { size += int(part.GetCurrentSize()) }
	p.IndivRefs = make([]IndivRef, size)

	// Now populate the refs array
	irIndex := 0
	for _, part := range p.Parts {
		for j := range part.Indivs {
			p.IndivRefs[irIndex].Indiv = part.Indivs[j]
			part.Indivs[j] = nil 	// eliminate this reference to the individual so garbage collection can delete the individual as soon as we use and eliminate the reference in IndivRefs in Mate() of next gen
			irIndex++
		}
	}
}


// sortIndexByFitness sorts the references to the individuals (p.IndivRefs) according to the individual's fitness (in ascending order)
func (p *Population) sortIndexByPhenoFitness() {
	sort.Sort(ByFitness(p.IndivRefs)) 		// sort the p.IndivRefs according to fitness

	// Output the fitnesses to check them, if verbosity is high enough
	if config.IsVerbose(9) {
		fitSlice := make([]float64, len(p.IndivRefs)) 	// create an array of the sorted individual fitness values so we can print them compactly
		for i,ind := range p.IndivRefs { fitSlice[i] = ind.Indiv.PhenoFitness
		}
		config.Verbose(9, "fitSlice: %v", fitSlice)
	}
}
