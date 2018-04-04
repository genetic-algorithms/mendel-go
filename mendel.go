/*
Package main is the main program of the golang version of mendel's accountant. It handles cmd line args, reads input files,
and contains the main generation loop of mating and selection.

The species and genome are modelled with this hierarchy of classes:

- Species
 - Populations (tribes)
  - PopulationPart (to enable parallel writes)
   - Individuals
    - Chromosomes
     - LinkageBlocks
      - Mutations
*/

/* Order of todos:
- Compare more runs with mendel-f90
- Tribes
- Figure out the delay at the beginning on r4 (and to a lesser extent on m4)
- (Consider) Add ability for user to specify a change to any input parameter at any generation (need to use reflection)
- Improve initial alleles: imitate diagnostics.f90:1351 ff which uses variable MNP to limit the number of alleles to 100000 for statistical sampling and normalizing that graph to be so we don't report hard numbers but ratios- (bruce) compare run results with mendel-f90
- add stats for length of time mutations have been in population (for both eliminated indivs and current pop)
- (When needed) support num offspring proportional to fitness (fitness_dependent_fertility in mendel-f90)
 */
package main

import (
	"log"
	"os"
	// "github.com/davecgh/go-spew/spew"
	"github.com/genetic-algorithms/mendel-go/config"
	"github.com/genetic-algorithms/mendel-go/utils"
	"github.com/genetic-algorithms/mendel-go/pop"
	"math/rand"
	"github.com/genetic-algorithms/mendel-go/dna"
	"github.com/pkg/profile"
	"strings"
	"runtime/debug"
	"github.com/genetic-algorithms/mendel-go/random"
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

	// Set all of the function ptrs for the algorithms we want to use.
	dna.SetModels(config.Cfg)
	pop.SetModels(config.Cfg)

	random.NextSeed = config.Cfg.Computation.Random_number_seed
	return random.RandFactory()
}

// Shutdown does all the stuff necessary at the end of the run.
func shutdown() {
	config.Verbose(5, "Shutting down...\n")
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

	maxGenNum := config.Cfg.Basic.Num_generations
	parentPop := pop.PopulationFactory(nil, 0) 		// genesis population
	pop.Mdl.GenerateInitialAlleles(parentPop, uniformRandom)
	parentPop.ReportInitial(maxGenNum)

	// Main generation loop.
	popMaxSet := pop.PopulationGrowthModelType(strings.ToLower(config.Cfg.Population.Pop_growth_model))==pop.EXPONENTIAL_POPULATON_GROWTH && config.Cfg.Population.Max_pop_size>0
	popMax := config.Cfg.Population.Max_pop_size
	//for gen := uint32(1); (maxGenNum == 0 || gen <= maxGenNum) && (!popMaxSet || parentPop.GetCurrentSize() < popMax); gen++ {
	for gen := uint32(1); ; gen++ {
		utils.Measure.Start("Generations")		// this is stopped in ReportEachGen() so it can report each delta
		childrenPop := pop.PopulationFactory(parentPop, gen)	// this creates the PopulationParts too
		random.NextSeed = config.Cfg.Computation.Random_number_seed+1		// reset the seed so tribes=1 is the same as pre-tribes
		parentPop.Mate(childrenPop, uniformRandom)		// this fills in the next gen population object with the offspring
		utils.Measure.CheckAmountMemoryUsed()
		parentPop = nil 	// give GC a chance to reclaim the previous generation
		if config.Cfg.Computation.Force_gc { utils.CollectGarbage() }
		childrenPop.Select(uniformRandom)

		// Check if we should stop the run
		lastGen := false
		if maxGenNum != 0 && gen >= maxGenNum {
			lastGen = true
		} else if popMaxSet && childrenPop.GetCurrentSize() >= popMax {
			log.Printf("Population has reached the max specified value of %d. Stopping simulation.", popMax)
			lastGen = true
		} else if (pop.RecombinationType(config.Cfg.Population.Recombination_model) == pop.FULL_SEXUAL && childrenPop.GetCurrentSize() < 2) || childrenPop.GetCurrentSize() == 0 {
			// Chcek if we don't have enough individuals to mate
			log.Println("Population is extinct. Stopping simulation.")
			lastGen = true
		} else if aveFit, _, _, _, _ := childrenPop.GetFitnessStats(); aveFit < config.Cfg.Computation.Extinction_threshold {
			// Check if the pop fitness is below the threashold
			log.Printf("Population fitness is below the extinction threshold of %.3f. Stopping simulation.", config.Cfg.Computation.Extinction_threshold)
			lastGen = true
		}

		childrenPop.ReportEachGen(gen, lastGen)
		if lastGen { break }
		parentPop = childrenPop        // for the next iteration
	}

	shutdown()	// Finish up
}
