package pop

import (
	"github.com/genetic-algorithms/mendel-go/config"
	"github.com/genetic-algorithms/mendel-go/random"
	"math/rand"
	"github.com/genetic-algorithms/mendel-go/utils"
	"fmt"
	"log"
)

// Species tracks all of the populations (tribes) and holds attributes common to the whole species.
type Species struct {
	Populations []*Population		// the tribes that make up this species
	PartsPerPop uint32 			// the number of population parts (threads) each population should have
}

func SpeciesFactory() *Species {
	s := &Species{
		Populations: make([]*Population, config.Cfg.Tribes.Num_tribes),
		PartsPerPop: uint32(utils.RoundUpInt(float64(config.Cfg.Computation.Num_threads) / float64(config.Cfg.Tribes.Num_tribes))),  // we round up because its ok to have more go threads than system threads
	}
	return s
}

// Initialize inits the populations for gen 0
func (s *Species) Initialize(maxGenNum uint32, uniformRandom *rand.Rand) *Species {
	defer utils.Measure.Start("InitializePopulations").Stop("InitializePopulations")
	config.Verbose(1, "Running with %d population(s), each with a size of %d, for %d generations with %d total threads", s.GetNumPopulations(), config.Cfg.Basic.Pop_size, maxGenNum, config.Cfg.Computation.Num_threads)
	for i := range s.Populations {
		var newRandom *rand.Rand
		if i == 0 {
			// Let the 1st pop use the main uniformRandom generator so tribes=1 is the same as pre-tribes
			newRandom = uniformRandom
		} else {
			newRandom = random.RandFactory()
		}
		s.Populations[i] = PopulationFactory(nil, 0, uint32(i+1), s.PartsPerPop) 		// genesis population
		Mdl.GenerateInitialAlleles(s.Populations[i], newRandom)
	}
	s.ReportInitial()
	return s 		// so we can chain calls
}

// GetNumPopulations returns the number of populations in this species
func (s *Species) GetNumPopulations() uint32 {
	return uint32(len(s.Populations))
}

// GetCurrentSize returns the sum of all of the pop sizes
func (s *Species) GetCurrentSize() (size uint32) {
	for _, p := range s.Populations {
		size += p.GetCurrentSize()
	}
	return
}

// GetNextGeneration prepares all of the populations for the next gen and returns them in a new Species object
func (parentS *Species) GetNextGeneration(gen uint32) (childrenS *Species) {
	random.NextSeed = config.Cfg.Computation.Random_number_seed + 1		// reset the seed to 1 above our initial seed, so when we call RandFactory() in Mate() for additional threads it will work like it did before
	childrenS = SpeciesFactory()
	for i := range parentS.Populations {
		childrenS.Populations[i] = PopulationFactory(parentS.Populations[i], gen, uint32(i+1), parentS.PartsPerPop)	// this creates the PopulationParts too
	}
	return
}

// Mate mates all of the populations
func (parentS *Species) Mate(childrenS *Species, uniformRandom *rand.Rand) {
	defer utils.Measure.Start("Mate").Stop("Mate")
	for i := range parentS.Populations {
		var newRandom *rand.Rand
		if i == 0 {
			// Let the 1st pop use the main uniformRandom generator so tribes=1 is the same as pre-tribes
			newRandom = uniformRandom
		} else {
			newRandom = random.RandFactory()
		}
		parentS.Populations[i].Mate(childrenS.Populations[i], newRandom)
	}
}

// Select does selection on all of the populations
func (s *Species) Select(uniformRandom *rand.Rand) {
	defer utils.Measure.Start("Select").Stop("Select")
	for i, p := range s.Populations {
		var newRandom *rand.Rand
		if i == 0 {
			// Let the 1st pop use the main uniformRandom generator so tribes=1 is the same as pre-tribes
			newRandom = uniformRandom
		} else {
			newRandom = random.RandFactory()
		}
		p.Select(newRandom)
	}
}

// Go thru all pops and see if they all have gone extinct or reached their pop max
func (s *Species) AllPopsDone() bool {
	for _, p := range s.Populations {
		if !p.Done && !p.IsDone(false) { return false }
	}
	return true
}

// Go thru all pops and mark as done any that have gone extinct or reached its pop max
func (s *Species) MarkDonePops() {
	for _, p := range s.Populations {
		if !p.Done && p.IsDone(true) { p.Done = true }
	}
}

// ReportInitial prints out stuff at the beginning, usually headers for data files, or a summary of the run we are about to do
func (s *Species) ReportInitial() {
	for _, p := range s.Populations {
		p.ReportInitial()
	}

	if config.Cfg.Tribes.Num_tribes > 1 {
		// Also initialize the summary/average files for the whole species
		if histWriter0 := config.FMgr.GetFile(config.HISTORY_FILENAME, 0); histWriter0 != nil {
			// Write header for this file
			fmt.Fprintln(histWriter0, "# Generation  Avg-deleterious Avg-neutral  Avg-favorable")
		}

		if fitWriter0 := config.FMgr.GetFile(config.FITNESS_FILENAME, 0); fitWriter0 != nil {
			// Write header for this file
			fmt.Fprintln(fitWriter0, "# Generation  Pop-size  Avg Offspring  Avg-fitness  Min-fitness  Max-fitness  Total Mutns  Mean Mutns  Noise")
		}
	}
}
// GetFitnessStats returns the average of all the individuals fitness levels across the pops, as well as the min and max, and total and mean mutations.
func (s *Species) GetFitnessStats() (meanFitness float64, minFitness float64, maxFitness float64, totalNumMutations uint64, meanNumMutations float64, speciesSize uint64) {
	//todo: consider caching these values, once we are doing runs with lots of tribes
	meanFitness = 0.0
	minFitness = 99.0
	maxFitness = -99.0
	totalNumMutations = uint64(0)
	meanNumMutations = 0.0
	speciesSize = uint64(0)
	for _, p := range s.Populations {
		popSize := p.GetCurrentSize()
		speciesSize += uint64(popSize)
		meanFit, minFit, maxFit, totalNumMuts, meanNumMuts := p.GetFitnessStats()
		meanFitness += meanFit * float64(popSize)	// we want meanFitness to be the mean of all indivs in all pops
		if minFit < minFitness { minFitness = minFit }
		if maxFit > maxFitness { maxFitness = maxFit }
		totalNumMutations += totalNumMuts
		meanNumMutations += meanNumMuts * float64(popSize)
	}
	meanFitness = meanFitness / float64(speciesSize)
	meanNumMutations = meanNumMutations / float64(speciesSize)
	return
}

func (s *Species) GetMutationStats() (meanNumDeleterious float64, meanNumNeutral float64, meanNumFavorable float64) {
	meanNumDeleterious = 0.0
	meanNumNeutral = 0.0
	meanNumFavorable = 0.0
	speciesSize := uint64(0)
	for _, p := range s.Populations {
		popSize := p.GetCurrentSize()
		speciesSize += uint64(popSize)
		meanNumDel, meanNumNeut, meanNumFav := p.GetMutationStats()
		meanNumDeleterious += meanNumDel * float64(popSize)	// we want meanFitness to be the mean of all indivs in all pops
		meanNumNeutral += meanNumNeut * float64(popSize)
		meanNumFavorable += meanNumFav * float64(popSize)
	}
	meanNumDeleterious = meanNumDeleterious / float64(speciesSize)
	meanNumNeutral = meanNumNeutral / float64(speciesSize)
	meanNumFavorable = meanNumFavorable / float64(speciesSize)
	return
}

// ReportEachGen reports stats on each population
func (s *Species) ReportEachGen(genNum uint32, lastGen bool, totalInterimTime, genTime float64) {
	defer utils.Measure.Start("ReportEachGen").Stop("ReportEachGen")
	// Report the fitness and mutations of each pop
	memUsed := utils.Measure.GetAmountMemoryUsed()
	for _, p := range s.Populations {
		p.ReportEachGen(genNum, lastGen, totalInterimTime, genTime, memUsed)
	}

	// Report the overall species stats
	if config.Cfg.Tribes.Num_tribes > 1 {
		perGenMinimalVerboseLevel := uint32(1) // level at which we will print only the info that is very quick to gather
		finalVerboseLevel := uint32(1)         // level at which we will print species summary info at the end of the run
		if config.IsVerbose(perGenMinimalVerboseLevel) || (lastGen && config.IsVerbose(finalVerboseLevel)) {
			aveFit, minFit, maxFit, totalMutns, meanMutns, speciesSize := s.GetFitnessStats()
			log.Printf("Species: Gen: %d, Time: %.4f, Gen time: %.4f, Mem: %.3f MB, Pop size: %v, Indiv mean fitness: %v, min fitness: %v, max fitness: %v, total num mutations: %v, mean num mutations: %v", genNum, totalInterimTime, genTime, memUsed, speciesSize, aveFit, minFit, maxFit, totalMutns, meanMutns)
		}
		// Not currently logging the mutations stats for any verbose level

		if fitWriter := config.FMgr.GetFile(config.FITNESS_FILENAME, 0); fitWriter != nil {
			config.Verbose(5, "Writing to file %v", config.FITNESS_FILENAME)
			aveFit, minFit, maxFit, totalMutns, meanMutns, speciesSize := s.GetFitnessStats() // GetFitnessStats() caches its values so it's ok to call it multiple times
			// If you change this line, you must also change the header in ReportInitial()
			fmt.Fprintf(fitWriter, "%d  %d  %v  %v  %v  %v  %v  %v  %v\n", genNum, speciesSize, 0, aveFit, minFit, maxFit, totalMutns, meanMutns, 0)
			//histWriter.Flush()  // <-- don't need this because we don't use a buffer for the file
			if lastGen {
				//todo: put summary stats in comments at the end of the file?
			}
		}

		if histWriter := config.FMgr.GetFile(config.HISTORY_FILENAME, 0); histWriter != nil {
			config.Verbose(5, "Writing to file %v", config.HISTORY_FILENAME)
			d, n, f := s.GetMutationStats() // GetMutationStats() caches its values so it's ok to call it multiple times
			// If you change this line, you must also change the header in ReportInitial()
			fmt.Fprintf(histWriter, "%d  %v  %v  %v\n", genNum, d, n, f)
			//histWriter.Flush()  // <-- don't need this because we don't use a buffer for the file
			if lastGen {
				//todo: put summary stats in comments at the end of the file?
			}
		}
	}

	// Count and output the alleles for each pop
	// This needs to come last if the lastGen because we free the individuals references to make memory room for the allele count
	utils.Measure.Start("allele-count")
	for _, p := range s.Populations {
		p.CountAlleles(genNum, lastGen)
		utils.Measure.CheckAmountMemoryUsed()
	}
	utils.Measure.Stop("allele-count")
}

// GetAverageFitness gets the overall fitness of the species to determine if it has gone extinct
func (s *Species) GetAverageFitness() (averageFitness float64) {
	if s.GetNumPopulations() == 0 { return }
	for _, p := range s.Populations {
		aveFit, _, _, _, _ := p.GetFitnessStats()	// its ok that this gets called multiple times, because it caches the data
		averageFitness += aveFit
	}
	return averageFitness / float64(s.GetNumPopulations())
}

/* don't think this is useful...
func (parentS *Species) MoveToNextGeneration(childrenS *Species, gen uint32, lastGen bool) *Species {
	for i := range parentS.Populations {
		childrenS.Populations[i].ReportEachGen(gen, lastGen)
	}
	return childrenS
}
*/
