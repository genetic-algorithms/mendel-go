package pop

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"log"
	"sort"
	"math/rand"
	"fmt"
	"math"
	"bitbucket.org/geneticentropy/mendel-go/utils"
)

type RecombinationType uint8
const (
	//CLONAL RecombinationType = 1   <-- have not needed these yet, uncomment when we do
	//SUPPRESSED RecombinationType = 2
	FULL_SEXUAL RecombinationType = 3
)

// Population tracks the tribes and global info about the population. It also handles population-wide actions
// like mating and selection.
type Population struct {
	Indivs []*Individual		//todo: we currently don't track males vs. females. Eventually we should: assign each offspring a sex, maintaining a prescribed sex ratio for the population. Only allow opposite sex individuals to mate.

	Size uint32                       // the specified size of this population. 0 if no specific size.
	Num_offspring float64             // Average number of offspring each mating pair should have. Calculated from config values Fraction_random_death and Reproductive_rate. Note: Reproductive_rate and ActualAvgOffspring are per individual and Num_offspring is per pair.
	ActualAvgOffspring        float64 // The average number of offspring each individual from last generation actually had in this generation
	PreSelGenoFitnessMean     float64 // The average fitness of all of the individuals (before selection) due to their genomic mutations
	PreSelGenoFitnessVariance float64 //
	PreSelGenoFitnessStDev    float64 // The standard deviation from the GenoFitnessMean
	EnvironNoise              float64 // randomness applied to geno fitness calculated from PreSelGenoFitnessVariance, heritability, and non_scaling_noise
}


// PopulationFactory creates a new population (either the initial pop, or the next generation).
func PopulationFactory(initialSize uint32) *Population {
	p := &Population{
		Indivs: make([]*Individual, 0, initialSize), 	// allocate the array for the ptrs to the indivs. The actual indiv objects will be appended either in Initialize or as the population grows during mating
		Size: config.Cfg.Basic.Pop_size,
	}

	fertility_factor := 1. - config.Cfg.Selection.Fraction_random_death
	p.Num_offspring = 2.0 * config.Cfg.Population.Reproductive_rate * fertility_factor 	// the default for Num_offspring is 4

	return p
}


// Initialize creates individuals (with no mutations) for the 1st generation
func (p *Population) Initialize() {
	//todo: there is probably a faster way to initialize these arrays
	for i:=uint32(1); i<=p.Size; i++ { p.Append(IndividualFactory(p)) }
}


// Size returns the current number of individuals in this population
func (p *Population) GetCurrentSize() uint32 { return uint32(len(p.Indivs)) }


// Append adds a person to this population. This is our function (instead of using append() directly, in case in
// the future we want to allocate additional individuals in bigger chunks for efficiency.
func (p *Population) Append(indivs ...*Individual) {
	p.Indivs = append(p.Indivs, indivs ...)
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
func (p *Population) Mate(uniformRandom *rand.Rand) *Population {
	config.Verbose(4, "Mating the population of %d individuals...\n", p.GetCurrentSize())

	// Create the next generation population object that we will fill in as a result of mating. It is ok if we underestimate the
	// size a little, because we will add individuals with p.Append() anyway.
	newGenerationSize := uint32((float64(p.GetCurrentSize()) / 2) * p.Num_offspring)
	newP := PopulationFactory(newGenerationSize)

	// To prepare for mating, create a shuffled slice of indices into the parent population
	parentIndices := uniformRandom.Perm(int(p.GetCurrentSize()))
	config.Verbose(9, "parentIndices: %v\n", parentIndices)

	// Mate pairs and create the offspring. Now that we have shuffled the parent indices, we can just go 2 at a time thru the indices.
	for i := uint32(0); i < p.GetCurrentSize() - 1; i += 2 {
		dadI := parentIndices[i]
		momI := parentIndices[i+1]
		newIndivs := p.Indivs[dadI].Mate(p.Indivs[momI], uniformRandom)
		newP.Append(newIndivs...)
	}

	// Save off the average num offspring for stats, before we select out individuals
	newP.ActualAvgOffspring = float64(newP.GetCurrentSize()) / float64(p.GetCurrentSize())

	newP.PreSelGenoFitnessMean, newP.PreSelGenoFitnessVariance, newP.PreSelGenoFitnessStDev = newP.CalcFitnessStats()

	return newP
}


// Select removes the least fit individuals in the population
func (p *Population) Select(uniformRandom *rand.Rand) {
	config.Verbose(4, "Select: eliminating %d individuals to try to maintain a population of %d...\n", p.GetCurrentSize()-p.Size, p.Size)

	// Calculate noise factor to get pheno fitness of each individual
	herit := config.Cfg.Selection.Heritability
	p.EnvironNoise = math.Sqrt(p.PreSelGenoFitnessVariance * (1.0-herit) / herit + math.Pow(config.Cfg.Selection.Non_scaling_noise,2))
	Mdl.ApplySelectionNoise(p, p.EnvironNoise, uniformRandom) 		// this sets PhenoFitness in each of the individuals

	// Sort the indexes of the Indivs array by fitness, and mark the least fit individuals as dead
	indexes := p.sortIndexByPhenoFitness()
	numDead := p.numAlreadyDead(indexes)

	if numDead > 0 {
		config.Verbose(3, "%d individuals died (fitness below 0) as a result of mutations added during mating", numDead)
	}

	currentSize := uint32(len(indexes))

	if currentSize > p.Size {
		numEliminate := currentSize - p.Size

		if numDead < numEliminate {
			// Mark those that should be eliminated dead. They are sorted by fitness in ascending order, so mark the 1st ones dead.
			for i := uint32(0); i < numEliminate; i++ {
				p.Indivs[indexes[i].Index].Dead = true
			}
		}
	}

	p.ReportDeadStats()

	// Compact the Indivs array by moving the live individuals to the 1st p.Size elements. Accumulate stats on the dead along the way.
	nextIndex := 0
	for i := 0; i < len(p.Indivs); i++ {
		ind := p.Indivs[i]
		if !ind.Dead {
			if i > nextIndex {
				// copy it into the next open spot
				p.Indivs[nextIndex] = ind 		// I think a shallow copy is ok, we only copy the pointers to the LB arrays
			}
			// else there are no open slots yet, because we have not encountered dead ones yet
			nextIndex++
		}
	}

	p.Indivs = p.Indivs[0:nextIndex] 		// readjust the slice to be only the live individuals

	return
}


// ApplySelectionNoiseType functions add environmental noise and selection noise to the GenoFitness to set the PhenoFitness of all of the individuals of the population
type ApplySelectionNoiseType func(p *Population, envNoise float64, uniformRandom *rand.Rand)

// ApplyTruncationNoise only adds environmental noise (no selection noise)
func ApplyFullTruncationNoise(p *Population, envNoise float64, uniformRandom *rand.Rand) {
	for _, ind := range p.Indivs {
		ind.PhenoFitness = ind.GenoFitness + uniformRandom.Float64() * envNoise
	}
}

// ApplyUnrestrictProbNoise adds environmental noise and unrestricted probability selection noise
func ApplyUnrestrictProbNoise(p *Population, envNoise float64, uniformRandom *rand.Rand) {
	/*
	For unrestricted probability selection, divide the phenotypic fitness by a uniformly distributed random number prior to
	ranking and truncation.  This procedure allows the probability of surviving and reproducing in the next generation to be
	directly related to phenotypic fitness and also for the correct number of individuals to be eliminated to maintain a constant
	population size.
	*/
	for _, ind := range p.Indivs {
		ind.PhenoFitness = ind.GenoFitness + uniformRandom.Float64() * envNoise
		ind.PhenoFitness = ind.PhenoFitness / (uniformRandom.Float64() + 1.0e-15)
	}
}

// ApplyProportProbNoise adds environmental noise and strict proportionality probability selection noise
func ApplyProportProbNoise(_ *Population, _ float64, _ *rand.Rand) {
	/*
	For strict proportionality probability selection, rescale the phenotypic fitness values such that the maximum value is one.
	Then divide the scaled phenotypic fitness by a uniformly distributed random number prior to ranking and truncation.
	Allow only those individuals to reproduce whose resulting ratio of scaled phenotypic fitness to the random number value
	exceeds one.  This approach ensures that no individual automatically survives to reproduce regardless of the value
	of the random number.  But it restricts the fraction of the offspring that can survive.  Therefore, when the reproduction
	rate is low, the number of surviving offspring may not be large enough to sustain a constant population size.
	 */
	utils.NotImplementedYet("ApplyProportProbNoise not implemented yet")
}

// ApplyPartialTruncationNoise adds environmental noise and partial truncation selection noise
func ApplyPartialTruncationNoise(_ *Population, _ float64, _ *rand.Rand) {
	/*
	For partial truncation selection, divide the phenotypic fitness by the sum of theta and (1. - theta) times a random
	number distributed uniformly between 0.0 and 1.0 prior to ranking and truncation, where theta is the parameter
	partial_truncation_value.  This selection scheme is intermediate between truncation selection and unrestricted
	probability selection.  The procedure allows for the correct number of individuals to be eliminated to maintain a constant
	population size.
	*/
	utils.NotImplementedYet("ApplyPartialTruncationNoise not implemented yet")
}


// CalcFitnessStats returns the mean geno fitness and std deviation
func (p *Population) CalcFitnessStats() (genoFitnessMean, genoFitnessVariance, genoFitnessStDev float64) {
	// Calc mean (average)
	for _, ind := range p.Indivs {
		genoFitnessMean += ind.GenoFitness
	}
	genoFitnessMean = genoFitnessMean / float64(p.GetCurrentSize())

	for _, ind := range p.Indivs {
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
	for _, ind := range p.Indivs {
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


// numAlreadyDead finds how many individuals (if any) have already been marked dead due to their fitness falling below the
// allowed threshold when mutations were added during mating. They will be at the beginning.
func (p *Population) numAlreadyDead(sortedIndexes []IndivFit) (numDead uint32) {
	for _, index := range sortedIndexes {
		if ! p.Indivs[index.Index].Dead { return } 		// since it is sorted by fitness in ascending order, once we hit a live indiv, they all will be, so we can stop counting
		numDead++
	}
	return
}


// GetFitnessStats returns the average of all the individuals fitness levels
func (p *Population) GetFitnessStats() (averageFitness, minFitness, maxFitness float64) {
	minFitness = 99.0
	maxFitness = -99.0
	for _, ind := range p.Indivs {
		averageFitness += ind.GenoFitness
		if ind.GenoFitness > maxFitness { maxFitness = ind.GenoFitness
		}
		if ind.GenoFitness < minFitness { minFitness = ind.GenoFitness
		}
	}
	averageFitness = averageFitness / float64(p.GetCurrentSize())
	return
}


// GetMutationStats returns the average number of deleterious, neutral, favorable mutations, and the average fitness factor of each
func (p *Population) GetMutationStats() (deleterious, neutral, favorable,  avDelFit, avFavFit float64) {
	// Get the average fitness factor of type of mutation, example: 20 @ .2 and 5 @ .4 = (20 * .2) + (5 * .4) / 25
	var delet, neut, fav uint32
	for _, ind := range p.Indivs {
		d, n, f, avD, avF := ind.GetMutationStats()
		config.Verbose(9, "pop: avD=%v, avF=%v", avD, avF)
		delet += d
		neut += n
		fav += f
		avDelFit += float64(d) * avD
		avFavFit += float64(f) * avF
	}
	size := float64(p.GetCurrentSize())
	if size > 0 {
		deleterious = float64(delet) / size
		neutral = float64(neut) / size
		favorable = float64(fav) / size
	}
	if delet > 0 { avDelFit = avDelFit / float64(delet) }
	if fav > 0 { avFavFit = avFavFit / float64(fav) }
	return
}


// Report prints out statistics of this population. If final==true is prints more details.
func (p *Population) ReportInitial(genNum, maxGenNum uint32) {
	config.Verbose(3, "Starting mendel simulation with a population size of %d at generation %d and continuing to generation %d", p.GetCurrentSize(), genNum, maxGenNum)

	if histWriter := config.FMgr.GetFile(config.HISTORY_FILENAME); histWriter != nil {
		// Write header for this file
		fmt.Fprintln(histWriter, "# Generation  Pop-size  Avg Offspring  Avg-deleterious Avg-neutral  Avg-favorable  Avg-del-fit Avg-neut-fit  Avg-fav-fit  Avg-fitness  Min-fitness  Max-fitness  Noise")
	}
}


// Report prints out statistics of this population. If final==true is prints more details.
func (p *Population) ReportEachGen(genNum uint32) {
	//todo: for reporting we go thru all individuals and LB multiple times. Make this more efficient
	verboseLevel := uint32(2)            // level at which we will print population level info
	indSumVerboseLevel := uint32(3)            // level at which we will print population level info
	indDetailVerboseLevel := uint32(7)    // level at which we will print individual level info
	popSize := p.GetCurrentSize()

	// Not final
	var d, n, f, avDelFit, avFavFit float64 	// if we get these values once, hold on to them
	var aveFit, minFit, maxFit float64
	if config.IsVerbose(verboseLevel) {
		aveFit, minFit, maxFit = p.GetFitnessStats()
		d, n, f, avDelFit, avFavFit = p.GetMutationStats()
		log.Printf("Gen: %d, Pop size: %v, Mean num offspring %v, Indiv mean fitness: %v, min fitness: %v, max fitness: %v, mean num mutations: %v, noise: %v", genNum, popSize, p.ActualAvgOffspring, aveFit, minFit, maxFit, d+n+f, p.EnvironNoise)
		if config.IsVerbose(indSumVerboseLevel) {
			log.Printf(" Indiv mutation detail means: deleterious: %v, neutral: %v, favorable: %v, del fitness: %v, fav fitness: %v, preselect fitness: %v, preselect fitness SD: %v", d, n, f, avDelFit, avFavFit, p.PreSelGenoFitnessMean, p.PreSelGenoFitnessStDev)
		}
	}
	if config.IsVerbose(indDetailVerboseLevel) {
		log.Println(" Individual Detail:")
		for _, ind := range p.Indivs { ind.Report(false) }
	}

	//if histWriter := config.FMgr.GetFileWriter(config.HISTORY_FILENAME); histWriter != nil {
	if histWriter := config.FMgr.GetFile(config.HISTORY_FILENAME); histWriter != nil {
		config.Verbose(5, "Writing to file %v", config.HISTORY_FILENAME)
		if d==0.0 && n==0.0 && f==0.0 { d, n, f, avDelFit, avFavFit = p.GetMutationStats() }
		if aveFit==0.0 && minFit==0.0 && maxFit==0.0 { aveFit, minFit, maxFit = p.GetFitnessStats() }
		fmt.Fprintf(histWriter, "%d  %d  %v  %v  %v  %v  %v  %v  %v  %v  %v  %v\n", genNum, popSize, p.ActualAvgOffspring, d, n, f, avDelFit, avFavFit, aveFit, minFit, maxFit, p.EnvironNoise)
		//histWriter.Flush()
	}
}


// Report prints out statistics of this population. If final==true is prints more details.
func (p *Population) ReportFinal(genNum uint32) {
	perGenVerboseLevel := uint32(2)            // level at which we already printed this info for each gen
	finalVerboseLevel := uint32(1)            // level at which we will print population level info
	perGenIndSumVerboseLevel := uint32(3)            // level at which we will print population level info
	finalIndSumVerboseLevel := uint32(2)            // level at which we will print population level info
	perGenIndDetailVerboseLevel := uint32(7)    // level at which we will print individual level info
	finalIndDetailVerboseLevel := uint32(6)    // level at which we will print individual level info
	popSize := p.GetCurrentSize()

	if !config.IsVerbose(perGenVerboseLevel) && config.IsVerbose(finalVerboseLevel) {
		log.Println("Final report:")
		aveFit, minFit, maxFit := p.GetFitnessStats()
		d, n, f, avDelFit, avFavFit := p.GetMutationStats()
		log.Printf("After %d generations: Pop size: %v, Mean num offspring %v, Indiv mean fitness: %v, min fitness: %v, max fitness: %v, mean num mutations: %v, noise: %v", genNum, popSize, p.ActualAvgOffspring, aveFit, minFit, maxFit, d+n+f, p.EnvironNoise)
		if !config.IsVerbose(perGenIndSumVerboseLevel) && config.IsVerbose(finalIndSumVerboseLevel) {
			log.Printf(" Indiv mutation detail means: deleterious: %v, neutral: %v, favorable: %v, del fitness: %v, fav fitness: %v", d, n, f, avDelFit, avFavFit)
		}
	}
	if !config.IsVerbose(perGenIndDetailVerboseLevel) && config.IsVerbose(finalIndDetailVerboseLevel) {
		log.Println(" Individual Detail:")
		for _, i := range p.Indivs {
			i.Report(true)
		}
	}
}


// Used as the elements to be sorted for selection
type IndivFit struct {
	Index uint32
	Fitness float64
}
type ByFitness []IndivFit
func (a ByFitness) Len() int           { return len(a) }
func (a ByFitness) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByFitness) Less(i, j int) bool { return a[i].Fitness < a[j].Fitness }


// sortIndexByFitness sorts the indexes of the individuals according to fitness (in ascending order)
func (p *Population) sortIndexByPhenoFitness() []IndivFit {
	// Initialize the index array
	indexes := make([]IndivFit, p.GetCurrentSize())
	for i := range indexes {
		indexes[i].Index = uint32(i)
		indexes[i].Fitness = p.Indivs[i].PhenoFitness
	}

	sort.Sort(ByFitness(indexes)) 		// sort the indexes according to fitness

	// Output the fitnesses to check them
	if config.IsVerbose(9) {
		fitSlice := make([]float64, len(indexes)) 	// create an array of the sorted individual fitness values so we can print them compactly
		for i,ind := range indexes { fitSlice[i] = p.Indivs[ind.Index].PhenoFitness
		}
		config.Verbose(9, "fitSlice: %v", fitSlice)
	}

	return indexes
}
