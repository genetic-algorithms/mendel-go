package pop

import (
	"bitbucket.org/geneticentropy/mendel-go/config"
	"log"
	"sort"
	"math/rand"
	"fmt"
	"math"
	"bitbucket.org/geneticentropy/mendel-go/utils"
	"encoding/json"
	"bitbucket.org/geneticentropy/mendel-go/dna"
	"os"
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
	Indivs []*Individual		// Note: we currently don't track males vs. females. Eventually we should: assign each offspring a sex, maintaining a prescribed sex ratio for the population. Only allow opposite sex individuals to mate.

	Size uint32                       // the target size of this population after selection
	//Num_offspring float64             // Average number of offspring each mating pair should have. Calculated from config values Fraction_random_death and Reproductive_rate. Note: Reproductive_rate and ActualAvgOffspring are per individual and Num_offspring is per pair.
	Num_offspring float64             // Average number of offspring each individual should have (so need to multiple by 2 to get it for the mating pair). Calculated from config values Fraction_random_death and Reproductive_rate.
	ActualAvgOffspring        float64 // The average number of offspring each individual from last generation actually had in this generation
	PreSelGenoFitnessMean     float64 // The average fitness of all of the individuals (before selection) due to their genomic mutations
	PreSelGenoFitnessVariance float64 //
	PreSelGenoFitnessStDev    float64 // The standard deviation from the GenoFitnessMean
	EnvironNoise              float64 // randomness applied to geno fitness calculated from PreSelGenoFitnessVariance, heritability, and non_scaling_noise
	LBsPerChromosome uint32		// How many linkage blocks in each chromosome. For now the total number of LBs must be an exact multiple of the number of chromosomes

	MeanFitness, MinFitness, MaxFitness float64		// cache summary info about the individuals

	MeanNumDeleterious, MeanNumNeutral, MeanNumFavorable float64		// cache some of the stats we usually gather
	MeanDelFit, MeanFavFit float64

	MeanNumDelAllele, MeanNumNeutAllele, MeanNumFavAllele float64		// cache some of the stats we usually gather
	MeanDelAlleleFit, MeanFavAlleleFit float64
}


// PopulationFactory creates a new population (either the initial pop, or the next generation).
func PopulationFactory(initialSize uint32, initialize bool) *Population {
	p := &Population{
		Indivs: make([]*Individual, 0, initialSize), 	// allocate the array for the ptrs to the indivs. The actual indiv objects will be appended either in Initialize or as the population grows during mating
		Size: config.Cfg.Basic.Pop_size,
	}

	fertility_factor := 1. - config.Cfg.Selection.Fraction_random_death
	//p.Num_offspring = 2.0 * config.Cfg.Population.Reproductive_rate * fertility_factor 	// the default for Num_offspring is 4
	p.Num_offspring = config.Cfg.Population.Reproductive_rate * fertility_factor 	// the default for Num_offspring is 2

	p.LBsPerChromosome = uint32(config.Cfg.Population.Num_linkage_subunits / config.Cfg.Population.Haploid_chromosome_number)	// main.initialize() already confirmed it was a clean multiple

	if initialize {
		// Create individuals (with no mutations) for the 1st generation. (For subsequent generations, individuals are added to the Population object via Mate().
		for i:=uint32(1); i<=initialSize; i++ { p.Append(IndividualFactory(p, true)) }
	}

	return p
}


// Size returns the current number of individuals in this population
func (p *Population) GetCurrentSize() uint32 { return uint32(len(p.Indivs)) }


// Append adds a person to this population. This is our function (instead of using append() directly), in case in
// the future we want to allocate additional individuals in bigger chunks for efficiency. See https://blog.golang.org/go-slices-usage-and-internals
func (p *Population) Append(indivs ...*Individual) {
	// Note: the initial make of the Indivs array is approximately big enough avoid append having to copy the array in most cases
	p.Indivs = append(p.Indivs, indivs ...)
}


// GenerateInitialAlleles creates the initial contrasting allele pairs (if specified by the config file) and adds them to the population
func (p *Population) GenerateInitialAlleles(uniformRandom *rand.Rand) {
	if config.Cfg.Population.Num_contrasting_alleles == 0 || config.Cfg.Population.Initial_alleles_pop_frac <= 0.0 { return }	// nothing to do

	//numIndivs := utils.RoundInt(float64(p.GetCurrentSize()) * config.Cfg.Population.Initial_alleles_pop_frac)

	// Loop thru individuals, and skipping or choosing individuals to maintain a ratio close to Initial_alleles_pop_frac
	// Note: config.Validate() already confirms Initial_alleles_pop_frac is > 0 and <= 1.0
	var numWithAlleles uint32 = 0 		// so we can calc the ratio so far of indivs we've given alleles to vs. number of indivs we've processed
	var numLBsWithAlleles uint32
	var numProcessedLBs uint32
	for i, ind := range p.Indivs {
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
func (p *Population) Mate(uniformRandom *rand.Rand) *Population {
	config.Verbose(4, "Mating the population of %d individuals...\n", p.GetCurrentSize())

	// Create the next generation population object that we will fill in as a result of mating. It is ok if we underestimate the
	// size a little, because we will add individuals with p.Append() anyway.
	//newGenerationSize := uint32((float64(p.GetCurrentSize()) / 2) * p.Num_offspring)
	newGenerationSize := uint32(float64(p.GetCurrentSize()) * p.Num_offspring)
	newP := PopulationFactory(newGenerationSize, false)

	// To prepare for mating, create a shuffled slice of indices into the parent population
	parentIndices := uniformRandom.Perm(int(p.GetCurrentSize()))
	//config.Verbose(9, "parentIndices: %v\n", parentIndices)

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
		config.Verbose(3, "%d individuals died (fitness < 0, or < 1 when using spps) as a result of mutations added during mating", numDead)
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
	/*
	Full truncation only adds a small amount of randomness, envNoise, which is calculated the 2 input parameters heritability and non_scaling_noise.
	Then the individuals are ordered by the resulting PhenoFitness, and the least fit are eliminated to achieve the desired population size.
	This makes full truncation the most efficient selection model, and unrealistic unless envNoise is set rather high.
	*/
	for _, ind := range p.Indivs {
		ind.PhenoFitness = ind.GenoFitness + uniformRandom.Float64() * envNoise
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

	for _, ind := range p.Indivs {
		//rnd1 := uniformRandom.Float64()
		ind.PhenoFitness = ind.GenoFitness + (uniformRandom.Float64() * envNoise)
		//rnd2 := uniformRandom.Float64()
		ind.PhenoFitness = ind.PhenoFitness / (uniformRandom.Float64() + 1.0e-15)
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
	for _, ind := range p.Indivs {
		ind.PhenoFitness = ind.GenoFitness + (uniformRandom.Float64() * envNoise)
		maxFitness = utils.MaxFloat64(maxFitness, ind.PhenoFitness)
	}
	// Verify maxFitness is not zero so we can divide by it below
	if maxFitness <= 0.0 { log.Fatalf("Max individual fitness is < 0 (%v), so whole population must be dead. Exiting.", maxFitness) }

	for _, ind := range p.Indivs {
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
	for _, ind := range p.Indivs {
		ind.PhenoFitness = ind.GenoFitness + (uniformRandom.Float64() * envNoise)
		ind.PhenoFitness = ind.PhenoFitness / (config.Cfg.Selection.Partial_truncation_value + ((1. - config.Cfg.Selection.Partial_truncation_value) * uniformRandom.Float64()))
	}
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


// GetFitnessStats returns the average of all the individuals fitness levels, as well as the min and max
func (p *Population) GetFitnessStats() (float64, float64, float64) {
	// See if we already calculated and cached the values
	if p.MeanFitness > 0.0 { return p.MeanFitness, p.MinFitness, p.MaxFitness }
	p.MinFitness = 99.0
	p.MaxFitness = -99.0
	p.MeanFitness = 0.0
	for _, ind := range p.Indivs {
		p.MeanFitness += ind.GenoFitness
		if ind.GenoFitness > p.MaxFitness { p.MaxFitness = ind.GenoFitness
		}
		if ind.GenoFitness < p.MinFitness { p.MinFitness = ind.GenoFitness
		}
	}
	p.MeanFitness = p.MeanFitness / float64(p.GetCurrentSize())
	return p.MeanFitness, p.MinFitness, p.MaxFitness
}


// GetMutationStats returns the average number of deleterious, neutral, favorable mutations, and the average fitness factor of each
func (p *Population) GetMutationStats() (float64, float64, float64,  float64, float64) {
	// See if we already calculated and cached the values. Note: we only check deleterious, because fav and neutral could be 0
	//config.Verbose(9, "p.MeanNumDeleterious=%v, p.MeanDelFit=%v", p.MeanNumDeleterious, p.MeanDelFit)
	if p.MeanNumDeleterious > 0 && p.MeanDelFit < 0.0 { return p.MeanNumDeleterious, p.MeanNumNeutral, p.MeanNumFavorable, p.MeanDelFit, p.MeanFavFit }
	p.MeanNumDeleterious=0.0;  p.MeanNumNeutral=0.0;  p.MeanNumFavorable=0.0;  p.MeanDelFit=0.0;  p.MeanFavFit=0.0

	// For each type of mutation, get the average fitness factor and number of mutation for every individual and combine them. Example: 20 @ .2 and 5 @ .4 = (20 * .2) + (5 * .4) / 25
	var delet, neut, fav uint32
	for _, ind := range p.Indivs {
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
	for _, ind := range p.Indivs {
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
func (p *Population) ReportInitial(genNum, maxGenNum uint32) {
	config.Verbose(3, "Starting mendel simulation with a population size of %d at generation %d and continuing to generation %d", p.GetCurrentSize(), genNum, maxGenNum)

	if histWriter := config.FMgr.GetFile(config.HISTORY_FILENAME); histWriter != nil {
		// Write header for this file
		fmt.Fprintln(histWriter, "# Generation  Pop-size  Avg Offspring  Avg-deleterious Avg-neutral  Avg-favorable  Avg-del-fit  Avg-fav-fit  Avg-fitness  Min-fitness  Max-fitness  Noise")
	}

	var allelesFilename string
	var alleleWriter *os.File
	if alleleWriter = config.FMgr.GetFile(config.ALLELES_COUNT_FILENAME); alleleWriter != nil {
		allelesFilename = config.ALLELES_COUNT_FILENAME
	} else if alleleWriter = config.FMgr.GetFile(config.ALLELES_FILENAME); alleleWriter != nil {
		allelesFilename = config.ALLELES_FILENAME
	}
	if alleleWriter != nil {
		// Write the outer json object and the array that will contain the output of each generation
		if _, err := alleleWriter.Write([]byte(`{"allelesForEachGen":[`)); err != nil {
			log.Fatalf("error writing alleles to %v: %v", allelesFilename, err)
		}
	}
}


// Report prints out statistics of this population
func (p *Population) ReportEachGen(genNum uint32) {
	lastGen := genNum == config.Cfg.Basic.Num_generations
	perGenVerboseLevel := uint32(2)            // level at which we will print population summary info each generation
	finalVerboseLevel := uint32(1)            // level at which we will print population summary info at the end of the run
	perGenIndSumVerboseLevel := uint32(3)            // level at which we will print individuals summary info each generation
	finalIndSumVerboseLevel := uint32(2)            // level at which we will print individuals summary info at the end of the run
	perGenIndDetailVerboseLevel := uint32(7)    // level at which we will print info about each individual each generation
	finalIndDetailVerboseLevel := uint32(6)    // level at which we will print info about each individual at the end of the run
	popSize := p.GetCurrentSize()

	//var d, n, f, avDelFit, avFavFit float64 	// if we get these values once, hold on to them
	//var aveFit, minFit, maxFit float64
	if config.IsVerbose(perGenVerboseLevel) || (lastGen && config.IsVerbose(finalVerboseLevel)) {
		aveFit, minFit, maxFit := p.GetFitnessStats()
		d, n, f, avDelFit, avFavFit := p.GetMutationStats()
		//d, n, f, avDelFit, avFavFit = p.GetMutationStats()  // <- just testing that the caching works properly
		log.Printf("Gen: %d, Pop size: %v, Mean num offspring %v, Indiv mean fitness: %v, min fitness: %v, max fitness: %v, mean num mutations: %v, noise: %v", genNum, popSize, p.ActualAvgOffspring, aveFit, minFit, maxFit, d+n+f, p.EnvironNoise)
		if config.IsVerbose(perGenIndSumVerboseLevel) || (lastGen && config.IsVerbose(finalIndSumVerboseLevel)) {
			log.Printf(" Indiv mutation detail means: deleterious: %v, neutral: %v, favorable: %v, del fitness: %v, fav fitness: %v, preselect fitness: %v, preselect fitness SD: %v", d, n, f, avDelFit, avFavFit, p.PreSelGenoFitnessMean, p.PreSelGenoFitnessStDev)
			ad, an, af, avDelAlFit, avFavAlFit := p.GetInitialAlleleStats()
			log.Printf(" Indiv initial allele detail means: deleterious: %v, neutral: %v, favorable: %v, del fitness: %v, fav fitness: %v", ad, an, af, avDelAlFit, avFavAlFit)
		}
	}
	if config.IsVerbose(perGenIndDetailVerboseLevel) || (lastGen && config.IsVerbose(finalIndDetailVerboseLevel)) {
		log.Println(" Individual Detail:")
		for _, ind := range p.Indivs { ind.Report(lastGen) }
	}

	if histWriter := config.FMgr.GetFile(config.HISTORY_FILENAME); histWriter != nil {
		config.Verbose(5, "Writing to file %v", config.HISTORY_FILENAME)
		//if d==0.0 && n==0.0 && f==0.0 { d, n, f, avDelFit, avFavFit = p.GetMutationStats() }
		d, n, f, avDelFit, avFavFit := p.GetMutationStats()		// GetMutationStats() caches its values so it's ok to call it multiple times
		//if aveFit==0.0 && minFit==0.0 && maxFit==0.0 { aveFit, minFit, maxFit = p.GetFitnessStats() }
		aveFit, minFit, maxFit := p.GetFitnessStats()		// GetFitnessStats() caches its values so it's ok to call it multiple times
		// If you change this line, you must also change the header in ReportInitial()
		fmt.Fprintf(histWriter, "%d  %d  %v  %v  %v  %v  %v  %v  %v  %v  %v  %v\n", genNum, popSize, p.ActualAvgOffspring, d, n, f, avDelFit, avFavFit, aveFit, minFit, maxFit, p.EnvironNoise)
		//histWriter.Flush()  // <-- don't need this because we don't use a buffer for the file
		if lastGen {
			//todo: put summary stats in comments at the end of the file?
		}
	}

	// Note: this allele json objects in wrapped in an outer json object and array, which we form manually so we don't have to gather the alleles from
	//		all generations into a single huge json object
	var allelesFilename string
	var alleleWriter *os.File
	if alleleWriter = config.FMgr.GetFile(config.ALLELES_COUNT_FILENAME); alleleWriter != nil {
		allelesFilename = config.ALLELES_COUNT_FILENAME
	} else if alleleWriter = config.FMgr.GetFile(config.ALLELES_FILENAME); alleleWriter != nil {
		allelesFilename = config.ALLELES_FILENAME
	}
	if alleleWriter != nil && (lastGen || (config.Cfg.Computation.Plot_allele_gens > 0 && (genNum % config.Cfg.Computation.Plot_allele_gens) == 0)) {
		var newJson []byte
		if allelesFilename == config.ALLELES_COUNT_FILENAME {
			// Count the alleles from all individuals
			alleles := dna.AlleleCountFactory(genNum)
			for _, ind := range p.Indivs {
				ind.CountAlleles(alleles)
			}
			//newJson, err := json.MarshalIndent(alleles, "", "  ")
			var err error
			newJson, err = json.Marshal(alleles)
			if err != nil { log.Fatalf("error marshaling alleles to json: %v", err) }
		} else {
			// Gather the alleles from all individuals
			alleles := &dna.Alleles{GenerationNumber: genNum}
			for _, ind := range p.Indivs {
				ind.GatherAlleles(alleles)
			}
			var err error
			newJson, err = json.Marshal(alleles)
			if err != nil { log.Fatalf("error marshaling alleles to json: %v", err) }
		}

		// Wrap the json in an outer json object and write it to the file
		if lastGen {
			newJson = append(newJson, "]}"...)		// no more allele outputs, so end the array and wrapping object
		} else {
			newJson = append(newJson, ","...)		// more json objects to come in the array so append a comma
		}
		if _, err := alleleWriter.Write(newJson); err != nil {
			log.Fatalf("error writing alleles to %v: %v", allelesFilename, err)
		}

	}
}


/* combined this with ReportEachGen()...
// ReportFinal prints out summary statistics of this population.
func (p *Population) ReportFinal(genNum uint32) {
	perGenVerboseLevel := uint32(2)            // level at which we already printed this info for each gen
	finalVerboseLevel := uint32(1)            // level at which we will print population info
	perGenIndSumVerboseLevel := uint32(3)            // level at which we already printed this info for each gen
	finalIndSumVerboseLevel := uint32(2)            // level at which we will print individuals summary info
	perGenIndDetailVerboseLevel := uint32(7)    // level at which we will already printed this info for each gen
	finalIndDetailVerboseLevel := uint32(6)    // level at which we will print info about each individual
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
*/


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
		//config.Verbose(9, "fitSlice: %v", fitSlice)
	}

	return indexes
}
