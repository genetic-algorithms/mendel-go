// Package main is the main program of the golang version of mendel's accountant.
// It handles cmd line args, reads input files, and contains the main generation loop.

/* Order of todos:
- is jon plotting initial alleles in allele plots?
- investigate difference in number of alleles between go and f90
- Compare more runs with mendel-f90
- Figure out the delay at the beginning on r4 (and to a lesser extent on m4)
- (Consider) Add ability for user to specify a change to any input parameter at any generation (need to use reflection)
- (Maybe not) Make pop reuse work for pop growth runs
- Improve initial alleles: imitate diagnostics.f90:1351 ff which uses variable MNP to limit the number of alleles to 100000 for statistical sampling and normalizing that graph to be so we don't report hard numbers but ratios- (bruce) compare run results with mendel-f90
- Tribes
- add stats for length of time mutations have been in population (for both eliminated indivs and current pop)
- (When needed) support num offspring proportional to fitness (fitness_dependent_fertility in mendel-f90)
- stop execution when any of these are reached: extinction_threshold, max_del_mutn_per_indiv, max_neu_mutn_per_indiv, max_fav_mutn_per_indiv
 */
package main

import (
	"log"
	"os"
	// "github.com/davecgh/go-spew/spew"
	"github.com/genetic-algorithms/mendel-go/config"
	"github.com/genetic-algorithms/mendel-go/utils"
	"github.com/genetic-algorithms/mendel-go/pop"
	"github.com/genetic-algorithms/mendel-go/random"
	"math/rand"
	"github.com/genetic-algorithms/mendel-go/dna"
	"github.com/pkg/profile"
	"strings"
	"runtime"
	"runtime/debug"
)

// Initialize initializes variables, objects, and settings.
func initialize() *rand.Rand {
	config.Verbose(5, "Initializing...\n")

	if config.Cfg.Computation.Force_gc {
		debug.SetGCPercent(-1)
	}

	utils.MeasurerFactory(config.Cfg.Computation.Verbosity)
	utils.Measure.Start("Total")

	utils.GlobalUniqueIntFactory()
	pop.PoolFactory()

	// Set all of the function ptrs for the algorithms we want to use.
	dna.SetModels(config.Cfg)
	pop.SetModels(config.Cfg)

	// Initialize random number generator
	var uniformRandom *rand.Rand
	if config.Cfg.Computation.Random_number_seed != 0 {
		uniformRandom = rand.New(rand.NewSource(config.Cfg.Computation.Random_number_seed))
	} else {
		uniformRandom = rand.New(rand.NewSource(random.GetSeed()))
	}

	return uniformRandom
}

// Shutdown does all the stuff necessary at the end of the run.
func shutdown() {
	config.Verbose(5, "Shutting down...\n")
	//config.FMgr.CloseAllFiles()  // <- done with defer instead
}

// Main handles cmd line args, reads input files, and contains the main generation loop.
func main() {
	log.SetOutput(os.Stdout) 	// needs to be done very early

	config.ReadCmdArgs()    // Get/check cmd line options and load specified input file - flags are accessible in config.CmdArgs, config values in config.Cfg

	// Handle the different input file choices
	if config.CmdArgs.InputFileToCreate != "" {
		if err := utils.CopyFile(config.FindDefaultFile(), config.CmdArgs.InputFileToCreate); err != nil { log.Fatalln(err) }
		os.Exit(0)

	} else if config.CmdArgs.InputFile != ""{
		if err := config.ReadFromFile(config.CmdArgs.InputFile); err != nil { log.Fatalln(err) }
		config.Verbose(3, "Case_id: %v\n", config.Cfg.Basic.Case_id)

	} else { config.Usage(0) }		// this will exit

	// ReadFromFile() opened the output files, so arrange for them to be closed at the end
	defer config.FMgr.CloseAllFiles()

	// Initialize profiling, if requested
	switch strings.ToLower(config.Cfg.Computation.Performance_profile) {
	case "":
		// no profiling, do nothing
	case "cpu":
		defer profile.Start(profile.CPUProfile, profile.ProfilePath("./pprof")).Stop()
	case "mem":
		defer profile.Start(profile.MemProfile, profile.ProfilePath("./pprof")).Stop()
	case "block":
		defer profile.Start(profile.BlockProfile, profile.ProfilePath("./pprof")).Stop()
	default:
		log.Fatalf("Error: unrecognized value for performance_profile: %v", config.Cfg.Computation.Performance_profile)
	}

	uniformRandom := initialize()

	parentPop := pop.PopulationFactory(nil, 0) 		// genesis population
	parentPop.GenerateInitialAlleles(uniformRandom)
	maxGenNum := config.Cfg.Basic.Num_generations
	parentPop.ReportInitial(maxGenNum)
	//var prevPop *pop.Population

	// Main generation loop.
	popMaxSet := pop.PopulationGrowthModelType(strings.ToLower(config.Cfg.Population.Pop_growth_model))==pop.EXPONENTIAL_POPULATON_GROWTH && config.Cfg.Population.Max_pop_size>0
	popMax := config.Cfg.Population.Max_pop_size
	for gen := uint32(1); (maxGenNum == 0 || gen <= maxGenNum) && (!popMaxSet || parentPop.GetCurrentSize() < popMax); gen++ {
		utils.Measure.Start("Generations")		// this is stopped in ReportEachGen() so it can report each delta
		//var newP *pop.Population
		//if config.Cfg.Computation.Reuse_populations && prevPop != nil {
		//	newP = prevPop.Reinitialize(population, gen)
		//} else {
		//	prevPop = nil 		// give GC a chance to free that population
		//	newP = pop.PopulationFactory(population, gen)
		//}
		childrenPop := pop.PopPool.GetNextGeneration(parentPop, gen)
		parentPop.Mate(childrenPop, uniformRandom)		// this fills in the next gen population object with the offspring
		//parentPop = nil 	// give GC a chance to reclaim the previous generation
		if config.Cfg.Computation.Force_gc {
			utils.Measure.Start("GC")
			runtime.GC()
			utils.Measure.Stop("GC")
		}
		childrenPop.Select(uniformRandom)

		if (pop.RecombinationType(config.Cfg.Population.Recombination_model) == pop.FULL_SEXUAL && childrenPop.GetCurrentSize() < 2) || childrenPop.GetCurrentSize() == 0 {
			log.Println("Population is extinct. Stopping simulation.")
			break
		}

		childrenPop.ReportEachGen(gen, gen == maxGenNum)
		pop.PopPool.RecyclePopulation(parentPop) 		// save this for reuse in the next gen
		parentPop = childrenPop        // for the next iteration
	}

	// Finish up
	//population.ReportFinal(maxGenNum)  // <- this is now handled by pop.ReportEachGen()
	shutdown()
}
