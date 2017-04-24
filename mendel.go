// Package  main is main program of the golang version of mendel's accountant.
// It handles cmd line args, reads input files, handles restarts, and contains the main generation loop.

/* Order of todos:
- stop execution when any of these are reached: extinction_threshold, max_del_mutn_per_indiv, max_neu_mutn_per_indiv, max_fav_mutn_per_indiv
- combine mutation effects according to Multiplicative_weighting
- (jon) apply correct Weibull fitness factor distribution to mutation - see wes' comments in slack
- (jon) the num mutations should be a poisson distribution with the mean being Mutn_rate
- run with more linkage blocks
 */
package main

import (
	"log"
	"os"

	// "github.com/davecgh/go-spew/spew"

	"bitbucket.org/geneticentropy/mendel-go/config"
	"bitbucket.org/geneticentropy/mendel-go/utils"
	"math"
	"bitbucket.org/geneticentropy/mendel-go/pop"
	"bitbucket.org/geneticentropy/mendel-go/random"
	"math/rand"
	"bitbucket.org/geneticentropy/mendel-go/dna"
)

// Initialize initializes variables, objects, and settings for either an initial run or a restart.
func initialize() *rand.Rand {
	config.Verbose(5, "Initializing...\n")

	//todo: support parallel

	// Adjust certain config values
	config.Cfg.Selection.Heritability = math.Max(1.e-20, config.Cfg.Selection.Heritability)   // Limit the minimum value of heritability to be 10**-20
	if config.Cfg.Mutations.Fraction_neutral == 0 { config.Cfg.Computation.Track_neutrals = false }   // no neutrals to track
	if config.Cfg.Computation.Track_neutrals { config.Cfg.Computation.Tracking_threshold = 0 }
	if config.Cfg.Mutations.Allow_back_mutn { config.Cfg.Computation.Tracking_threshold = 0 }  // If back mutations are allowed, set the tracking threshold to zero so that all mutations are tracked

	// Set all of the function ptrs for the algorithms we want to use.
	dna.SetAlgorithms(config.Cfg)
	pop.SetAlgorithms(config.Cfg)

	// Initialize random number generator
	var uniformRandom *rand.Rand
	if config.Cfg.Computation.Random_number_seed != 0 {
		uniformRandom = rand.New(rand.NewSource(config.Cfg.Computation.Random_number_seed))
	} else {
		uniformRandom = rand.New(rand.NewSource(random.GetSeed()))
	}

	// Read restart file, if specified
	config.ReadRestartFile("")

	//todo: complete this function

	return uniformRandom
}

// Shutdown does all the stuff necessary at the end of the run.
func shutdown() {
	config.Verbose(5, "Shutting down...\n")
	config.FMgr.CloseFiles()
}

// Main handles cmd line args, reads input files, handles restarts, and contains the main generation loop.
func main() {
	log.SetOutput(os.Stdout) 	// needs to be done very early

	config.ReadCmdArgs()    // Get/check cmd line options and load specified input file - flags are accessible in config.CmdArgs, config values in config.Cfg

	// Handle the different input file choices
	if config.CmdArgs.InputFileToCreate != "" {
		if err := utils.CopyFile(config.DEFAULT_INPUT_FILE, config.CmdArgs.InputFileToCreate); err != nil { log.Fatalln(err) }
		os.Exit(0)

	} else if config.CmdArgs.InputFile != ""{
		if err := config.ReadFromFile(config.CmdArgs.InputFile); err != nil { log.Fatalln(err) }
		config.Verbose(3, "Case_id: %v\n", config.Cfg.Basic.Case_id)

	} else { config.Usage(0) }

	// Initialize
	uniformRandom := initialize()
	population := pop.PopulationFactory(config.Cfg.Basic.Pop_size) 		// time 0 population
	population.Initialize()
	population.ReportInitial(config.Restart.Gen_0, config.Cfg.Basic.Num_generations)

	// Main generation loop. config.Restart.Gen_0 allows us to restart some number of generations into the simulation.
	for gen := config.Restart.Gen_0+1; gen <= config.Cfg.Basic.Num_generations; gen++ {
		population = population.Mate(uniformRandom)
		population.Select()
		population.ReportEachGen(gen)
	}

	// Finish up
	population.ReportFinal(config.Cfg.Basic.Num_generations)
	shutdown()
}
