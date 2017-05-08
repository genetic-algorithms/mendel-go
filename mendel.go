// Package  main is main program of the golang version of mendel's accountant.
// It handles cmd line args, reads input files, handles restarts, and contains the main generation loop.

/* Order of todos:
- figure out why average fitness starts to climb after a while
- figure out what to do about dynamic linkage blocks
- cache averages in pop and ind objects for reuse
- use subclasses for Mutation and stop using MutationType
- can i use interfaces for the non-class model functions?
- support num offspring proportional to fitness (fitness_dependent_fertility in mendel-f90)
- what is genome_size used for besides weibull?
- add tracking id for each mutation
- review all of mendel fortran help.html
- stop execution when any of these are reached: extinction_threshold, max_del_mutn_per_indiv, max_neu_mutn_per_indiv, max_fav_mutn_per_indiv
- combine mutation effects according to Multiplicative_weighting
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
	if config.Cfg.Computation.Track_neutrals { config.Cfg.Computation.Tracking_threshold = 0 } 	//todo: we do not honor this yet
	if config.Cfg.Mutations.Allow_back_mutn { config.Cfg.Computation.Tracking_threshold = 0 }  // If back mutations are allowed, set the tracking threshold to zero so that all mutations are tracked

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

	// Read restart file, if specified
	config.ReadRestartFile("")

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
		population.Select(uniformRandom)
		population.ReportEachGen(gen)
	}

	// Finish up
	population.ReportFinal(config.Cfg.Basic.Num_generations)
	shutdown()
}
